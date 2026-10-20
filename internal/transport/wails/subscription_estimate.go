package wails

import (
	"strings"
	"time"
	"unicode"

	"omniproxy/internal/application/activity"
)

// SubscriptionQuotaEstimate 仅组合公开缓存快照与同身份的本机用量，不额外请求官方额度。
func (s *AppService) SubscriptionQuotaEstimate(id string) (activity.QuotaEstimate, error) {
	if s.activity == nil {
		return activity.QuotaEstimate{}, activity.ErrUnavailable
	}
	if id == "" || len(id) > 128 || strings.IndexFunc(id, unicode.IsControl) >= 0 {
		return activity.QuotaEstimate{}, activity.ErrInvalid
	}
	manager, err := s.subscriptionService()
	if err != nil {
		return activity.QuotaEstimate{}, err
	}
	ctx, cancel := subscriptionContext()
	defer cancel()
	accounts, err := s.accountService()
	if err != nil {
		return activity.QuotaEstimate{}, err
	}
	item, err := accounts.Metadata(ctx, id)
	if err != nil {
		return activity.QuotaEstimate{}, err
	}
	quota, err := manager.CachedQuota(ctx, id)
	if err != nil || quota == nil {
		return activity.QuotaEstimate{}, err
	}
	current, err := accounts.Metadata(ctx, id)
	if err != nil {
		return activity.QuotaEstimate{}, err
	}
	if item.UpdatedAt != current.UpdatedAt || item.AuthEmail != current.AuthEmail || item.AuthAccountID != current.AuthAccountID || item.AuthState != current.AuthState {
		return activity.QuotaEstimate{}, activity.ErrUnavailable
	}
	return s.activity.EstimateQuota(ctx, item, *quota, time.Now())
}
