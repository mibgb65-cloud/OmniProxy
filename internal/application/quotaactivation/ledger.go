package quotaactivation

import (
	"context"
	"database/sql"
	"errors"
	"omniproxy/internal/application/account"
	"omniproxy/internal/application/subscription"
	"time"
)

// 原始观测与已确认周期分开保存，旧版写入的占位目标不能再当作已激活证据。
func (s *Service) observe(ctx context.Context, item account.Item, quota subscription.Quota) (quotaAssessment, error) {
	result := quotaAssessment{quota: copyQuota(quota)}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, failure("record_failed")
	}
	defer tx.Rollback()
	for _, window := range windows(quota) {
		if window.value == nil {
			continue
		}
		if window.value.RemainingPercent < 0 || window.value.RemainingPercent > 100 {
			result.waiting = true
			continue
		}
		if window.value.ResetsAt != "" {
			reset, parseErr := time.Parse(time.RFC3339, window.value.ResetsAt)
			if parseErr != nil || reset.Unix() <= 0 {
				result.waiting = true
				continue
			}
		}
		var previous timerObservation
		var version, confirmed string
		err = tx.QueryRowContext(ctx, `SELECT identity_version,observed_reset,observed_at,confirmed_reset FROM codex_quota_activation_windows WHERE account_id=? AND window_kind=?`, item.ID, window.kind).Scan(&version, &previous.reset, &previous.at, &confirmed)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return result, failure("record_failed")
		}
		if version != identityVersion(item) {
			previous = timerObservation{}
			confirmed = ""
		}
		assessed := assessWindow(window.kind, window.value, previous, quota.FetchedAt, s.now())
		setWindow(&result.quota, window.kind, assessed.value)
		if assessed.floating {
			result.floating = append(result.floating, window.kind)
		}
		result.waiting = result.waiting || assessed.waiting
		if assessed.confirmed != "" {
			next, _ := time.Parse(time.RFC3339, assessed.confirmed)
			prior, parseErr := time.Parse(time.RFC3339, confirmed)
			if confirmed != "" && parseErr != nil {
				return result, failure("record_failed")
			}
			// 延迟快照不能把已经确认的未来周期倒退，导致提前再次激活。
			if confirmed == "" || !next.Before(prior) {
				confirmed = assessed.confirmed
			}
		}
		reset := window.value.ResetsAt
		if reset == "" {
			reset = previous.reset
		}
		at := ""
		if _, valid := snapshotTime(quota.FetchedAt, s.now()); valid && window.value.ResetsAt != "" {
			at = quota.FetchedAt
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO codex_quota_activation_windows(account_id,window_kind,identity_version,observed_reset,observed_at,confirmed_reset) VALUES(?,?,?,?,?,?)
ON CONFLICT(account_id,window_kind) DO UPDATE SET observed_reset=excluded.observed_reset,observed_at=excluded.observed_at,confirmed_reset=excluded.confirmed_reset,
attempted_cycle=CASE WHEN identity_version<>excluded.identity_version THEN '' ELSE attempted_cycle END,
attempted_at=CASE WHEN identity_version<>excluded.identity_version THEN '' ELSE attempted_at END,identity_version=excluded.identity_version`, item.ID, window.kind, identityVersion(item), reset, at, confirmed)
		if err != nil {
			return result, failure("record_failed")
		}
	}
	if err = tx.Commit(); err != nil {
		return result, failure("record_failed")
	}
	return result, nil
}
