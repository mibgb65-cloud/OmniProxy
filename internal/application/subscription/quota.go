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
	FromCache       bool                `json:"fromCache,omitempty"`
}

type QuotaWindow struct {
	RemainingPercent int      `json:"remainingPercent"`
	ResetsAt         string   `json:"resetsAt,omitempty"`
	UsedPercent      *float64 `json:"usedPercent,omitempty"`
	ObservedAt       string   `json:"observedAt,omitempty"`
}

// ReadQuota 使用官方组件的托管授权读取额度，不强制续期或改变账号状态。
func (s *Service) ReadQuota(ctx context.Context, id string) (Quota, error) {
	return s.readQuota(ctx, id, false)
}

// ReadDisplayQuota 只供界面使用，可保留官方暂缺的旧窗口；不能用来驱动调度或自动激活。
func (s *Service) ReadDisplayQuota(ctx context.Context, id string) (Quota, error) {
	return s.readQuota(ctx, id, true)
}

func (s *Service) readQuota(ctx context.Context, id string, display bool) (Quota, error) {
	// 后台激活也使用此入口；总预算必须从排队前开始，而非仅依赖 Wails 的外层超时。
	readCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	if err := s.operations.LockAccount(readCtx, id); err != nil {
		return Quota{}, quotaFailure(readCtx, "queue", "quota_unavailable", err)
	}
	defer s.operations.UnlockAccount(id)
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
	if s.quotaCache != nil {
		snapshot, err := s.quotaCache.Save(readCtx, item, profile, quota)
		if err != nil {
			return Quota{}, quotaFailure(readCtx, "cache", "quota_cache_unavailable", err)
		}
		if display {
			snapshot.ResetCredits = quota.ResetCredits
			return snapshot, nil
		}
	}
	return quota, nil
}
