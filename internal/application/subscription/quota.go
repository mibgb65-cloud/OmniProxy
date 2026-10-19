package subscription

import (
	"context"
	"time"
)

// Quota 返回当前快照的剩余百分比，不表示请求次数或模型调用权限。
type Quota struct {
	Plan            string              `json:"plan"`
	FiveHourEnabled bool                `json:"fiveHourEnabled"`
	FiveHour        *QuotaWindow        `json:"fiveHour,omitempty"`
	Weekly          *QuotaWindow        `json:"weekly,omitempty"`
	FetchedAt       string              `json:"fetchedAt"`
	ResetCredits    *ResetCreditSummary `json:"resetCredits,omitempty"`
}

type QuotaWindow struct {
	RemainingPercent int    `json:"remainingPercent"`
	ResetsAt         string `json:"resetsAt,omitempty"`
}

// ReadQuota 只查询本应用管理的订阅授权，不改变账号状态或注销凭据。
func (s *Service) ReadQuota(ctx context.Context, id string) (Quota, error) {
	s.operations.Lock()
	defer s.operations.Unlock()
	if ctx.Err() != nil {
		return Quota{}, failure("quota_unavailable")
	}
	if s.accountBusy(id) {
		return Quota{}, failure("busy")
	}
	item, profile, err := s.accounts.SubscriptionProfile(ctx, id)
	if err != nil {
		return Quota{}, failure("invalid_account")
	}
	if item.AuthState == "logged_out" || item.AuthState == "reauth_required" {
		return Quota{}, failure("quota_not_authenticated")
	}
	readCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, err := s.open(readCtx, profile)
	if err != nil {
		return Quota{}, err
	}
	defer client.Close()
	result, err := client.RateLimits(readCtx)
	if err != nil {
		return Quota{}, failure("quota_unavailable")
	}
	if item.AuthAccountID != "" && result.AccountID != "" && item.AuthAccountID != result.AccountID {
		return Quota{}, failure("identity_changed")
	}
	if err := verifyMember(readCtx, client, item); err != nil {
		return Quota{}, err
	}
	quota, err := normalizeQuota(result, item.AuthPlan)
	if err == nil {
		s.rememberQuota(item.ID, item.UpdatedAt, quota)
	}
	return quota, err
}
