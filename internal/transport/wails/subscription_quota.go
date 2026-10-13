package wails

import (
	"omniproxy/internal/application/quotaactivation"
	"omniproxy/internal/application/subscription"
)

// SubscriptionQuota 查询订阅额度，返回公开百分比与重置时间，不返回官方身份或凭据。
func (s *AppService) SubscriptionQuota(id string) (subscription.Quota, error) {
	service, err := s.subscriptionService()
	if err != nil {
		return subscription.Quota{}, err
	}
	ctx, cancel := subscriptionContext()
	defer cancel()
	quota, err := service.ReadQuota(ctx, id)
	if err == nil && s.quotaActivation != nil && quotaactivation.NeedsActivation(quota) {
		s.quotaActivation.Wake(false)
	}
	return quota, err
}
