package subscription

import (
	"context"
	"testing"
	"time"
)

func TestEstimateBaselineTracksOnlyChangedActiveWindowAndPersists(t *testing.T) {
	f, item, profile, initial := savedQuotaFixture(t)
	now := time.Now().UTC().Truncate(time.Second)
	initial.FiveHour = &QuotaWindow{RemainingPercent: 70, ResetsAt: now.Add(3 * time.Hour).Format(time.RFC3339)}
	initial.Weekly = &QuotaWindow{RemainingPercent: 60, ResetsAt: now.Add(3 * 24 * time.Hour).Format(time.RFC3339)}
	initial.FetchedAt = now.Format(time.RFC3339)
	ctx := context.Background()
	if _, err := f.manager.quotaCache.Save(ctx, item, profile, initial); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`DELETE FROM subscription_quota_estimate_baselines`); err != nil {
		t.Fatal(err)
	}
	next := initial
	next.FiveHour = &QuotaWindow{RemainingPercent: 100, ResetsAt: initial.FiveHour.ResetsAt}
	if _, err := f.manager.quotaCache.Save(ctx, item, profile, next); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM subscription_quota_estimate_baselines WHERE window_type='weekly'`).Scan(&count); err != nil || count != 0 {
		t.Fatal("5h-only reset changed the weekly baseline", err)
	}
	next.Weekly = &QuotaWindow{RemainingPercent: 100, ResetsAt: initial.Weekly.ResetsAt}
	if _, err := f.manager.quotaCache.Save(ctx, item, profile, next); err != nil {
		t.Fatal(err)
	}
	var start string
	if err := f.db.QueryRow(`SELECT start_at FROM subscription_quota_estimate_baselines WHERE account_id=? AND window_type='weekly'`, item.ID).Scan(&start); err != nil || start != next.FetchedAt {
		t.Fatal("weekly early-reset baseline was not persisted", err)
	}
	// 旧响应不能通过基线表绕过快照的时间保护。
	stale := next
	stale.FetchedAt = now.Add(-time.Hour).Format(time.RFC3339)
	stale.Weekly = &QuotaWindow{RemainingPercent: 100, ResetsAt: now.Add(6 * 24 * time.Hour).Format(time.RFC3339)}
	if _, err := f.manager.quotaCache.Save(ctx, item, profile, stale); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRow(`SELECT start_at FROM subscription_quota_estimate_baselines WHERE account_id=? AND window_type='weekly'`, item.ID).Scan(&start); err != nil || start != next.FetchedAt {
		t.Fatal("stale response changed the reset baseline", err)
	}
}
