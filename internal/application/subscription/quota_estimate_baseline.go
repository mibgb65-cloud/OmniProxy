package subscription

import (
	"context"
	"database/sql"
	"time"
)

// 只为 5h 提前重置保存消费基线；周额度始终使用自己的完整一周窗口。
func saveEstimateBaselines(ctx context.Context, tx *sql.Tx, id string, previous *Quota, next Quota) error {
	if previous == nil || previous.Plan != next.Plan {
		_, err := tx.ExecContext(ctx, `DELETE FROM subscription_quota_estimate_baselines WHERE account_id=?`, id)
		return err
	}
	fetched, _ := time.Parse(time.RFC3339, next.FetchedAt)
	old, current := previous.FiveHour, next.FiveHour
	if old == nil || current == nil || old.ResetsAt == "" || current.ResetsAt == "" {
		return nil
	}
	oldReset, _ := time.Parse(time.RFC3339, old.ResetsAt)
	newReset, _ := time.Parse(time.RFC3339, current.ResetsAt)
	if !oldReset.After(fetched) {
		_, err := tx.ExecContext(ctx, `DELETE FROM subscription_quota_estimate_baselines WHERE account_id=? AND window_type='five_hour'`, id)
		return err
	}
	if newReset.Equal(oldReset) && quotaUsedPercent(current) >= quotaUsedPercent(old) {
		return nil
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO subscription_quota_estimate_baselines(account_id,window_type,reset_at,start_at)
		VALUES(?,'five_hour',?,?) ON CONFLICT(account_id,window_type) DO UPDATE SET reset_at=excluded.reset_at,start_at=excluded.start_at`,
		id, current.ResetsAt, next.FetchedAt)
	return err
}

func quotaUsedPercent(window *QuotaWindow) float64 {
	if window.UsedPercent != nil {
		return *window.UsedPercent
	}
	return float64(100 - window.RemainingPercent)
}
