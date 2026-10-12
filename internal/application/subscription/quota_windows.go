package subscription

import (
	"strings"
	"time"

	"omniproxy/internal/infrastructure/codex"
)

func normalizeQuota(result codex.RateLimits, fallbackPlan string) (Quota, error) {
	snapshot := result.Snapshot
	if preferred := result.ByLimitID["codex"]; preferred != nil {
		snapshot = preferred
	}
	if snapshot == nil || snapshot.LimitID != "" && snapshot.LimitID != "codex" {
		return Quota{}, failure("quota_unavailable")
	}
	plan := snapshot.Plan
	if plan == "" || plan == "unknown" {
		plan = fallbackPlan
	}
	quota := Quota{Plan: plan, FiveHourEnabled: plan == "plus" || plan == "team", FetchedAt: time.Now().UTC().Format(time.RFC3339), ResetCredits: normalizeResetCredits(result.ResetCredits)}
	for _, window := range []*codex.RateLimitWindow{snapshot.Primary, snapshot.Secondary} {
		// 空窗口与缺失持续时间没有足够信息，不凭 primary／secondary 位置猜测额度周期。
		if window == nil || window.DurationMins == nil || window.UsedPercent == nil {
			continue
		}
		value := &QuotaWindow{RemainingPercent: max(0, 100-int(*window.UsedPercent))}
		if window.ResetsAt != nil {
			value.ResetsAt = time.Unix(*window.ResetsAt, 0).UTC().Format(time.RFC3339)
		}
		switch *window.DurationMins {
		case 300:
			if quota.FiveHourEnabled {
				quota.FiveHour = value
			}
		case 10080:
			quota.Weekly = value
		}
	}
	quota.Plan = strings.TrimSpace(quota.Plan)
	return quota, nil
}
