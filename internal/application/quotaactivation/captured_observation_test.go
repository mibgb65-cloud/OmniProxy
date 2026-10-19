package quotaactivation

import (
	"context"
	"omniproxy/internal/application/account"
	"omniproxy/internal/application/subscription"
	"testing"
	"time"
)

// 真实只读采样的关系重放：时间整体平移、比例改为典型值，不保存真实账号用量或重置目标。
func TestCapturedSlidingQuotaReplayActivatesExactlyOnce(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	f.service.now = func() time.Time { return now }
	quota := subscription.Quota{Plan: "team", FiveHourEnabled: true, FetchedAt: now.Format(time.RFC3339), FiveHour: &subscription.QuotaWindow{RemainingPercent: 100, ResetsAt: now.Add(5*time.Hour + time.Second).Format(time.RFC3339)}, Weekly: &subscription.QuotaWindow{RemainingPercent: 50, ResetsAt: now.Add(6 * 24 * time.Hour).Format(time.RFC3339)}}
	f.quotas.set(quota)
	if err := f.service.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if f.sends.Load() != 0 || f.service.Status().Activated != 0 || f.service.Status().Pending != 1 {
		t.Fatal("captured placeholder was treated as active")
	}
	now = now.Add(40 * time.Second)
	quota.FetchedAt = now.Format(time.RFC3339)
	quota.FiveHour.ResetsAt = now.Add(5*time.Hour + time.Second).Format(time.RFC3339)
	f.quotas.set(quota)
	f.service.sender = sendFunc(func(context.Context, account.Item, SendPlan) error {
		f.sends.Add(1)
		fresh := copyQuota(quota)
		fresh.FiveHour.RemainingPercent = 99
		fresh.FiveHour.ResetsAt = now.Add(5 * time.Hour).Format(time.RFC3339)
		f.quotas.set(fresh)
		return nil
	})
	if err := f.service.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if f.sends.Load() != 1 || f.service.Status().Activated != 1 {
		t.Fatal("captured observations did not activate once")
	}
	var n int
	f.db.QueryRow(`SELECT COUNT(*) FROM codex_quota_activation_windows WHERE attempted_cycle<>''`).Scan(&n)
	if n != 1 {
		t.Fatal("active weekly window incorrectly claimed")
	}
	f.service.Check(context.Background())
	if f.sends.Load() != 1 {
		t.Fatal("confirmed account sent twice")
	}
}
