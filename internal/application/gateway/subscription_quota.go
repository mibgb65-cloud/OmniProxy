package gateway

import (
	"time"
)

type subscriptionQuotaAvailability interface {
	QuotaBlockedUntil(string, string, time.Time) (time.Time, bool)
	RecordQuotaRejection(string, string, time.Time)
}

// subscriptionQuotaRefresher 让真正参与流量的账号被按需重新观察额度，
// 不新增定时器，也不查询闲置账号。
type subscriptionQuotaRefresher interface {
	ObserveQuota(string, string)
}

func (s *Service) subscriptionQuotaWait(t target, now time.Time) time.Duration {
	if t.kind != "openai" {
		return 0
	}
	if source, ok := s.subscription.(subscriptionQuotaAvailability); ok {
		if until, blocked := source.QuotaBlockedUntil(t.accountID, t.updatedAt, now); blocked {
			return max(time.Second, until.Sub(now))
		}
	}
	return 0
}

func (s *Service) rememberQuotaRejection(t target, rejected *retryReason) {
	if rejected == nil || rejected.code != "upstream_quota_exhausted" {
		return
	}
	if source, ok := s.subscription.(subscriptionQuotaAvailability); ok {
		source.RecordQuotaRejection(t.accountID, t.updatedAt, s.failover.now().Add(rejected.delay))
	}
}

// refreshQuota 只对刚结束真实请求的 OpenAI 订阅账号刷新额度观察：下一条消息不必再撞
// 一次已知耗尽的账号，同时不做任何后台全量探测。
func (s *Service) refreshQuota(t target) {
	if t.kind != "openai" || t.accountID == "" || t.updatedAt == "" {
		return
	}
	if source, ok := s.subscription.(subscriptionQuotaRefresher); ok {
		source.ObserveQuota(t.accountID, t.updatedAt)
	}
}
