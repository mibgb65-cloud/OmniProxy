package googlesubscription

import (
	"context"
	"time"

	"omniproxy/internal/provider/antigravity"
)

// observeRefreshInterval 是同一账号两次「429 后刷新」的最小间隔：
// 刷新只为让后续请求看到真实剩余，不需要每条失败请求都打一次官方查询。
const observeRefreshInterval = time.Minute

// ObserveQuota 在 429 之后异步刷新一次额度快照，让同组其它模型与界面都看到真实剩余，
// 不必等用户打开卡片。它按账号去重、不重试，失败只记稳定码。
func (s *Service) ObserveQuota(id string) {
	if s == nil || id == "" || !s.claimObserve(id) {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if _, err := s.ReadQuota(ctx, id); err != nil {
			s.mu.Lock()
			log, closed := s.log, s.closed
			s.mu.Unlock()
			if log != nil && !closed {
				log.Warn("google quota refresh failed", "module", "googlesubscription", "code", antigravity.ErrorCode(err))
			}
		}
	}()
}

// claimObserve 只让同一账号每分钟刷新一次，并在关闭后停止刷新。
func (s *Service) claimObserve(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	if s.observed == nil {
		s.observed = map[string]time.Time{}
	}
	now := time.Now()
	if last, ok := s.observed[id]; ok && now.Sub(last) < observeRefreshInterval {
		return false
	}
	s.observed[id] = now
	// 有界：只清理已经过了去重窗口的条目，不按请求量增长。
	if len(s.observed) > 256 {
		for key, at := range s.observed {
			if now.Sub(at) >= observeRefreshInterval {
				delete(s.observed, key)
			}
		}
	}
	return true
}
