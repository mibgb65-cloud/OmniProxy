package subscription

import (
	"context"
	"database/sql"
	"time"
)

// 与快照原子写入，避免刷新卡成功后将重置前用量与重置后百分比相乘。
func saveEstimateBaselines(ctx context.Context, tx *sql.Tx, id string, previous *Quota, next Quota) error {
	if previous == nil || previous.Plan != next.Plan {
		_, err := tx.ExecContext(ctx, `DELETE FROM subscription_quota_estimate_baselines WHERE account_id=?`, id)
		return err
	}
	fetched, _ := time.Parse(time.RFC3339, next.FetchedAt)
	for _, pair := range []struct {
		name         string
		old, current *QuotaWindow
	}{{"five_hour", previous.FiveHour, next.FiveHour}, {"weekly", previous.Weekly, next.Weekly}} {
		if pair.old == nil || pair.current == nil || pair.old.ResetsAt == "" || pair.current.ResetsAt == "" {
			continue
		}
		oldReset, _ := time.Parse(time.RFC3339, pair.old.ResetsAt)
		newReset, _ := time.Parse(time.RFC3339, pair.current.ResetsAt)
		if !oldReset.After(fetched) {
			if _, err := tx.ExecContext(ctx, `DELETE FROM subscription_quota_estimate_baselines WHERE account_id=? AND window_type=?`, id, pair.name); err != nil {
				return err
			}
			continue
		}
		if newReset.Equal(oldReset) && quotaUsedPercent(pair.current) >= quotaUsedPercent(pair.old) {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO subscription_quota_estimate_baselines(account_id,window_type,reset_at,start_at)
			VALUES(?,?,?,?) ON CONFLICT(account_id,window_type) DO UPDATE SET reset_at=excluded.reset_at,start_at=excluded.start_at`,
			id, pair.name, pair.current.ResetsAt, next.FetchedAt); err != nil {
			return err
		}
	}
	return nil
}

func quotaUsedPercent(window *QuotaWindow) float64 {
	if window.UsedPercent != nil {
		return *window.UsedPercent
	}
	return float64(100 - window.RemainingPercent)
}
