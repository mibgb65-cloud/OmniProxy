package wails

import (
	"omniproxy/internal/application/quotaactivation"
	"omniproxy/internal/application/subscription"
)

// SubscriptionQuota 查询订阅额度，返回公开百分比与重置时间，不返回官方身份或凭据。
func (s *AppService) SubscriptionQuota(id string) (subscription.Quota, error) {
	service, err := s.subscriptionService()
	if err != nil {
		s.logSubscriptionQuotaError(err)
		return subscription.Quota{}, err
	}
	ctx, cancel := subscriptionContext()
	defer cancel()
	quota, err := service.ReadDisplayQuota(ctx, id)
	s.logSubscriptionQuotaError(err)
	if err == nil && !quota.FromCache && s.quotaActivation != nil && quotaactivation.NeedsObservation(quota) {
		s.quotaActivation.Wake(false)
	}
	return quota, err
}

// SubscriptionCachedQuota 恢复上次显示快照，不依赖官方 CLI 或网络。
func (s *AppService) SubscriptionCachedQuota(id string) (*subscription.Quota, error) {
	service, err := s.subscriptionService()
	if err != nil {
		return nil, err
	}
	ctx, cancel := subscriptionContext()
	defer cancel()
	value, err := service.CachedQuota(ctx, id)
	s.logSubscriptionQuotaError(err)
	return value, err
}

func (s *AppService) logSubscriptionQuotaError(err error) {
	if err != nil && s.log != nil {
		diagnostic := subscription.QuotaErrorDiagnostic(err)
		s.log.Warn("subscription quota query failed", "module", "subscription", "operation", "quota", "stage", diagnostic.Stage, "code", diagnostic.Code, "cause_code", diagnostic.CauseCode)
	}
}
