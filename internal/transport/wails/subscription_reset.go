package wails

import "omniproxy/internal/application/subscription"

// SubscriptionResetStatus 返回最近操作回执，不触发官方兑换。
func (s *AppService) SubscriptionResetStatus(id string) (subscription.ResetOperation, error) {
	service, err := s.subscriptionService()
	if err != nil {
		return subscription.ResetOperation{}, err
	}
	ctx, cancel := subscriptionContext()
	defer cancel()
	return service.ResetStatus(ctx, id)
}

// SubscriptionConsumeReset 通过稳定幂等键执行用户确认的单次手动兑换。
func (s *AppService) SubscriptionConsumeReset(input subscription.ResetInput) (subscription.ResetResult, error) {
	service, err := s.subscriptionService()
	if err != nil {
		return subscription.ResetResult{}, err
	}
	ctx, cancel := subscriptionContext()
	defer cancel()
	return service.ConsumeReset(ctx, input)
}
