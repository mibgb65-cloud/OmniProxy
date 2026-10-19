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

// ReadQuota 使用官方组件的托管授权读取额度，不强制续期或改变账号状态。
func (s *Service) ReadQuota(ctx context.Context, id string) (Quota, error) {
	// 后台激活也使用此入口；总预算必须从排队前开始，而非仅依赖 Wails 的外层超时。
	readCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	if err := s.operations.Lock(readCtx); err != nil {
		return Quota{}, quotaFailure(readCtx, "queue", "quota_unavailable", err)
	}
	defer s.operations.Unlock()
	if s.accountBusy(id) {
		return Quota{}, quotaFailure(readCtx, "account", "busy", failure("busy"))
	}
	item, profile, err := s.accounts.SubscriptionProfile(readCtx, id)
	if err != nil {
		return Quota{}, quotaFailure(readCtx, "account", "invalid_account", failure("invalid_account"))
	}
	if item.AuthState == "logged_out" || item.AuthState == "reauth_required" {
		return Quota{}, quotaFailure(readCtx, "account", "quota_not_authenticated", failure("quota_not_authenticated"))
	}
	client, err := s.open(readCtx, profile)
	if err != nil {
		return Quota{}, quotaFailure(readCtx, "open", ErrorCode(err), err)
	}
	defer client.Close()
	// true 在托管模式下会强制续期，每次查额度都访问认证服务会额外放大网络失败。
	// 先读取并核验当前成员，正常授权续期交给官方额度读取链路，不绕过身份隔离。
	identity, err := client.Account(readCtx, false)
	if err != nil {
		return Quota{}, quotaFailure(readCtx, "account", "quota_unavailable", err)
	}
	if identity == nil {
		return Quota{}, quotaFailure(readCtx, "account", "quota_not_authenticated", failure("quota_not_authenticated"))
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
