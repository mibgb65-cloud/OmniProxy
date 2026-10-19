package subscription

import "context"

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

// ReadQuota 由官方组件续期后读取本应用管理的订阅额度，不改变账号状态或注销凭据。
func (s *Service) ReadQuota(ctx context.Context, id string) (Quota, error) {
	s.operations.Lock()
	defer s.operations.Unlock()
	if ctx.Err() != nil {
		return Quota{}, quotaFailure(ctx, "queue", "quota_unavailable", ctx.Err())
	}
	if s.accountBusy(id) {
		return Quota{}, quotaFailure(ctx, "account", "busy", failure("busy"))
	}
	item, profile, err := s.accounts.SubscriptionProfile(ctx, id)
	if err != nil {
		return Quota{}, quotaFailure(ctx, "account", "invalid_account", failure("invalid_account"))
	}
	if item.AuthState == "logged_out" || item.AuthState == "reauth_required" {
		return Quota{}, quotaFailure(ctx, "account", "quota_not_authenticated", failure("quota_not_authenticated"))
	}
	readCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	client, err := s.open(readCtx, profile)
	if err != nil {
		return Quota{}, quotaFailure(readCtx, "open", ErrorCode(err), err)
	}
	defer client.Close()
	// 额度读取同样需要可用授权；先由官方组件续期并验证成员，避免将过期令牌直接用于查询。
	identity, err := client.Account(readCtx, true)
	if err != nil {
		return Quota{}, quotaFailure(readCtx, "refresh", "quota_refresh_failed", err)
	}
	if identity == nil {
		return Quota{}, quotaFailure(readCtx, "refresh", "quota_not_authenticated", failure("quota_not_authenticated"))
	}
	if !sameEmail(item.AuthEmail, identity.Email) {
		return Quota{}, quotaFailure(readCtx, "identity", "identity_changed", failure("identity_changed"))
	}
	result, err := client.RateLimits(readCtx)
	if err != nil {
		return Quota{}, quotaFailure(readCtx, "rate_limits", "quota_unavailable", err)
	}
	if item.AuthAccountID != "" && result.AccountID != "" && item.AuthAccountID != result.AccountID {
		return Quota{}, quotaFailure(readCtx, "identity", "identity_changed", failure("identity_changed"))
	}
	quota, err := normalizeQuota(result, item.AuthPlan)
	if err != nil {
		return Quota{}, quotaFailure(readCtx, "normalize", "quota_unavailable", err)
	}
	s.rememberQuota(item.ID, item.UpdatedAt, quota)
	return quota, nil
}
