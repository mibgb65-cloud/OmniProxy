package workbuddy

import (
	"context"
	"time"
)

// CreditWindow 是一个公开额度窗口；字段缺失表示未知，不补 0。
type CreditWindow struct {
	Name      string  `json:"name"`
	Used      float64 `json:"used"`
	Amount    float64 `json:"amount,omitempty"`
	Limit     float64 `json:"limit,omitempty"`
	Display   string  `json:"display,omitempty"`
	Unit      string  `json:"unit,omitempty"`
	Aside     bool    `json:"aside,omitempty"`
	ResetsAt  string  `json:"resetsAt,omitempty"`
	RunsOutAt string  `json:"runsOutAt,omitempty"`
	RunsOut   float64 `json:"runsOut,omitempty"`
	Prepaid   bool    `json:"prepaid,omitempty"`
}

// Quota 是账号的 Credits 快照；FromCache 表示读取失败时展示的最近成功值。
type Quota struct {
	Plan      string         `json:"plan"`
	Windows   []CreditWindow `json:"windows"`
	FetchedAt string         `json:"fetchedAt"`
	FromCache bool           `json:"fromCache,omitempty"`
}

// ReadQuota 读取账号当前的 Credits 三窗口；失败时回退到保留期内的最近快照。
func (s *Service) ReadQuota(ctx context.Context, id string) (Quota, error) {
	quota, err := s.readQuota(ctx, id)
	if err != nil {
		if cached, ok := s.cachedQuota(id); ok {
			cached.FromCache = true
			return cached, nil
		}
		return Quota{}, err
	}
	s.recordQuota(id, quota)
	return quota, nil
}

func (s *Service) readQuota(ctx context.Context, id string) (Quota, error) {
	build, item, err := s.buildOf(ctx, id)
	if err != nil {
		return Quota{}, err
	}
	cred, err := s.credentials(ctx, id, false)
	if err != nil {
		return Quota{}, err
	}
	operation, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	// Credits 接口按提供商代理策略分流，不能只靠目标主机名回退。
	credits, err := s.clientFor(build).ReadCredits(item.OutboundContext(operation), cred)
	if err != nil {
		return Quota{}, err
	}
	quota := Quota{Plan: credits.Plan, FetchedAt: credits.FetchedAt.UTC().Format(time.RFC3339)}
	for _, window := range credits.Windows {
		quota.Windows = append(quota.Windows, CreditWindow{
			Name:      window.Name,
			Used:      window.Used,
			Amount:    window.Amount,
			Limit:     window.Limit,
			Display:   window.Display,
			Unit:      window.Unit,
			Aside:     window.Aside,
			ResetsAt:  window.ResetsAt,
			RunsOutAt: window.RunsOutAt,
			RunsOut:   window.RunsOut,
			Prepaid:   window.Prepaid,
		})
	}
	return quota, nil
}

// CachedQuota 只返回保留期内的内存快照，不发网络请求。
func (s *Service) CachedQuota(id string) (Quota, bool) {
	return s.cachedQuota(id)
}

func (s *Service) cachedQuota(id string) (Quota, bool) {
	s.quotaMu.Lock()
	defer s.quotaMu.Unlock()
	entry, ok := s.quotas[id]
	if !ok || entry.fetchedAt.IsZero() || time.Since(entry.fetchedAt) > quotaFreshness {
		return Quota{}, false
	}
	quota := entry.quota
	quota.Windows = append([]CreditWindow(nil), entry.quota.Windows...)
	return quota, true
}
