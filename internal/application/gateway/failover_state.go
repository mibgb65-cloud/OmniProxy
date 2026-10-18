package gateway

import (
	"sort"
	"sync"
	"time"
)

const maxHealthEntries = 4096

// AccountHealth 独立于用户启用状态，仅描述短期失败及恢复中的账号。
type AccountHealth struct {
	AccountID   string `json:"accountId"`
	AccountName string `json:"accountName"`
	Model       string `json:"model"`
	Code        string `json:"code"`
	RetryAt     string `json:"retryAt"`
	Recovering  bool   `json:"recovering"`
}
type healthKey struct{ account, version, model string }
type healthEntry struct {
	target       target
	until        time.Time
	code         string
	epoch, probe uint64
	active       int
}
type healthLease struct {
	key       healthKey
	id, epoch uint64
}
type failoverState struct {
	mu       sync.Mutex
	entries  map[healthKey]*healthEntry
	sequence uint64
	now      func() time.Time
}

func newFailoverState() *failoverState {
	return &failoverState{entries: map[healthKey]*healthEntry{}, now: time.Now}
}

func (s *failoverState) wait(account, version, model string) time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e := s.entries[healthKey{account, version, model}]; e != nil && (e.until.After(s.now()) || e.probe != 0) {
		return max(time.Second, e.until.Sub(s.now()))
	}
	return 0
}

func (s *failoverState) acquire(t target, model string) (healthLease, bool, time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	key := healthKey{t.accountID, t.updatedAt, model}
	e := s.entries[key]
	if e != nil && (e.until.After(now) || e.probe != 0) {
		return healthLease{}, false, max(time.Second, e.until.Sub(now))
	}
	if e == nil {
		// 仅淘汰无在途请求的条目，保证恢复资格和失败代次不会被并发成功错误清除。
		for k, v := range s.entries {
			if v.active == 0 && (v.until.Before(now) || k.account == key.account && k.version != key.version) {
				delete(s.entries, k)
			}
		}
		if len(s.entries) >= maxHealthEntries {
			for k, v := range s.entries {
				if v.active == 0 {
					delete(s.entries, k)
					break
				}
			}
		}
		if len(s.entries) >= maxHealthEntries {
			return healthLease{}, false, time.Second
		}
		e = &healthEntry{target: t}
		s.entries[key] = e
	}
	s.sequence++
	lease := healthLease{key: key, id: s.sequence, epoch: e.epoch}
	e.active++
	if e.code != "" {
		e.probe = lease.id
	}
	return lease, true, 0
}

func (s *failoverState) finish(lease healthLease, rejected *retryReason, healthy bool) {
	if lease.id == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entries[lease.key]
	if e == nil {
		return
	}
	e.active--
	if e.probe == lease.id {
		e.probe = 0
	}
	if rejected != nil {
		e.epoch++
		e.code = rejected.code
		e.until = maxTime(e.until, s.now().Add(max(time.Second, rejected.delay)))
	} else if healthy && e.epoch == lease.epoch {
		e.code = ""
		e.until = time.Time{}
	}
	if e.active == 0 && e.code == "" {
		delete(s.entries, lease.key)
	}
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// FailoverHealth 返回有界运行快照，不读取凭据或主动探测模型。
func (s *Service) FailoverHealth() []AccountHealth {
	s.failover.mu.Lock()
	defer s.failover.mu.Unlock()
	items := []AccountHealth{}
	for key, e := range s.failover.entries {
		if e.code == "" {
			continue
		}
		items = append(items, AccountHealth{AccountID: key.account, AccountName: e.target.snapshot.AccountName, Model: key.model, Code: e.code, RetryAt: e.until.UTC().Format(time.RFC3339), Recovering: e.probe != 0})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].AccountID != items[j].AccountID {
			return items[i].AccountID < items[j].AccountID
		}
		return items[i].Model < items[j].Model
	})
	return items
}
