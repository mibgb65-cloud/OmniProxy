package gateway

import (
	"time"
)

type subscriptionQuotaAvailability interface {
	QuotaBlockedUntil(string, string, time.Time) (time.Time, bool)
	RecordQuotaRejection(string, string, time.Time)
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
