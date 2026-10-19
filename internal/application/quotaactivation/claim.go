package quotaactivation

import (
	"context"
	"omniproxy/internal/application/account"
	"omniproxy/internal/application/subscription"
	"slices"
	"time"
)

type windowClaim struct{ kind, cycle string }

// claim 在请求前登记；没有已确认周期时，未知旧结果继续阻止重发。
func (s *Service) claim(ctx context.Context, item account.Item, quota subscription.Quota, floating []string) ([]windowClaim, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, failure("record_failed")
	}
	defer tx.Rollback()
	claimed := []windowClaim{}
	for _, window := range windows(quota) {
		if !notStarted(window.value) {
			continue
		}
		var observed, attempted, version, confirmed string
		if err = tx.QueryRowContext(ctx, `SELECT observed_reset,attempted_cycle,identity_version,confirmed_reset FROM codex_quota_activation_windows WHERE account_id=? AND window_kind=?`, item.ID, window.kind).Scan(&observed, &attempted, &version, &confirmed); err != nil {
			return nil, failure("record_failed")
		}
		if version != identityVersion(item) {
			return nil, failure("identity_changed")
		}
		cycle := observed
		if slices.Contains(floating, window.kind) {
			cycle = confirmed
			if cycle == "" {
				cycle = attempted
			}
		} else if confirmed != "" {
			// 明确无目标路径仍尊重此前未来目标，防止一次短暂缺失覆盖已启动窗口的保护。
			prior, parseErr := time.Parse(time.RFC3339, observed)
			if observed == "" || parseErr == nil && !prior.After(s.now()) {
				cycle = confirmed
			}
		}
		if cycle == "" {
			cycle = "initial"
		}
		if cycle != "initial" {
			reset, e := time.Parse(time.RFC3339, cycle)
			if e != nil {
				return nil, failure("record_failed")
			}
			if reset.After(s.now()) {
				continue
			}
		}
		if cycle == attempted {
			continue
		}
		if _, err = tx.ExecContext(ctx, `UPDATE codex_quota_activation_windows SET attempted_cycle=?,attempted_at=? WHERE account_id=? AND window_kind=?`, cycle, s.now().UTC().Format(time.RFC3339), item.ID, window.kind); err != nil {
			return nil, failure("record_failed")
		}
		claimed = append(claimed, windowClaim{window.kind, cycle})
	}
	if err = tx.Commit(); err != nil {
		return nil, failure("record_failed")
	}
	return claimed, nil
}

func buildSendPlan(before subscription.Quota, claims []windowClaim, floating []string) SendPlan {
	plan := SendPlan{Before: before}
	for _, claim := range claims {
		plan.Claimed = append(plan.Claimed, claim.kind)
		if slices.Contains(floating, claim.kind) {
			plan.Floating = append(plan.Floating, claim.kind)
		}
	}
	return plan
}
