package quotaactivation

import (
	"errors"
	"omniproxy/internal/application/subscription"
	"slices"
	"time"
)

// SendPlan 只由已持久化观测的后台生成，不接收前端传入的激活许可。
type SendPlan struct {
	Before   subscription.Quota
	Claimed  []string
	Floating []string
}

// prepareFailure 保留发送方给出的具体原因；只有拿不到稳定码时才退回通用码。
func prepareFailure(sendErr error) error {
	if codeOf(sendErr) != "" {
		return sendErr
	}
	return errors.Join(sendErr, failure("prepare_failed"))
}

// 发送前仍需确认占位目标继续滑动；稳定／未知／耗尽的窗口不能被强行转换为未开始。
func plannedNeed(fresh subscription.Quota, plan SendPlan) (bool, error) {
	if len(plan.Claimed) < 1 || len(plan.Claimed) > 2 || len(plan.Floating) > 2 {
		return false, errors.Join(ErrNotSent, failure("invalid_plan"))
	}
	seen := map[string]bool{}
	for _, kind := range plan.Claimed {
		if (kind != "five_hour" && kind != "weekly") || seen[kind] {
			return false, errors.Join(ErrNotSent, failure("invalid_plan"))
		}
		seen[kind] = true
	}
	if !usableQuota(fresh) {
		return false, nil
	}
	normalized := copyQuota(fresh)
	for _, window := range windows(fresh) {
		if !seen[window.kind] {
			setWindow(&normalized, window.kind, nil)
		}
	}
	for _, kind := range plan.Claimed {
		if windowFor(fresh, kind) == nil {
			return false, errors.Join(ErrNotSent, ErrAwaitingQuota)
		}
	}
	for _, kind := range plan.Floating {
		if !slices.Contains(plan.Claimed, kind) {
			return false, errors.Join(ErrNotSent, ErrAwaitingQuota)
		}
		before, after := windowFor(plan.Before, kind), windowFor(fresh, kind)
		if before == nil || after == nil {
			return false, errors.Join(ErrNotSent, ErrAwaitingQuota)
		}
		if after.RemainingPercent != 100 || after.ResetsAt == "" {
			continue
		}
		oldAt, e1 := time.Parse(time.RFC3339, plan.Before.FetchedAt)
		at, e2 := time.Parse(time.RFC3339, fresh.FetchedAt)
		oldReset, e3 := time.Parse(time.RFC3339, before.ResetsAt)
		reset, e4 := time.Parse(time.RFC3339, after.ResetsAt)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil || before.RemainingPercent != 100 {
			return false, errors.Join(ErrNotSent, ErrAwaitingQuota)
		}
		gap := at.Sub(oldAt)
		if gap < 2*time.Second || gap > activationTimeout {
			return false, errors.Join(ErrNotSent, ErrAwaitingQuota)
		}
		if !nearFullWindow(reset, at, kind) {
			continue
		}
		if !nearFullWindow(oldReset, oldAt, kind) {
			return false, errors.Join(ErrNotSent, ErrAwaitingQuota)
		}
		delta := reset.Sub(oldReset)
		if delta == 0 {
			continue
		}
		if delta < 0 || absolute(delta-gap) > timerDrift {
			return false, errors.Join(ErrNotSent, ErrAwaitingQuota)
		}
		normalizedWindow := windowFor(normalized, kind)
		normalizedWindow.ResetsAt = ""
	}
	return NeedsActivation(normalized), nil
}
