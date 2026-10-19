package quotaactivation

import (
	"context"
	"omniproxy/internal/application/account"
	"testing"
	"time"
)

func TestLegacyUncertainClaimCannotBeResetBySlidingTargets(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	now := time.Now().UTC().Truncate(time.Second)
	f.service.now = func() time.Time { return now }
	oldCycle := now.Add(-8 * time.Hour).Format(time.RFC3339)
	_, err := f.db.Exec(`INSERT INTO codex_quota_activation_windows(account_id,window_kind,identity_version,observed_reset,attempted_cycle,attempted_at) VALUES(?,?,?,?,?,?)`, f.accounts.item.ID, "five_hour", identityVersion(f.accounts.item), now.Add(5*time.Hour).Format(time.RFC3339), oldCycle, oldCycle)
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		f.quotas.set(floatingQuota(now))
		f.service.Check(context.Background())
		now = now.Add(5 * time.Minute)
	}
	if f.sends.Load() != 0 {
		t.Fatal("old uncertain record was treated as an unused cycle")
	}
	var attempted string
	f.db.QueryRow(`SELECT attempted_cycle FROM codex_quota_activation_windows WHERE window_kind='five_hour'`).Scan(&attempted)
	if attempted != oldCycle {
		t.Fatal("old send registration changed")
	}
}

func TestConfirmedTargetCannotMoveBackwardAndBlocksPrematureCycle(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	now := time.Now().UTC().Truncate(time.Second)
	f.service.now = func() time.Time { return now }
	active := floatingQuota(now)
	active.FiveHour.RemainingPercent = 99
	active.FiveHour.ResetsAt = now.Add(4 * time.Hour).Format(time.RFC3339)
	f.quotas.set(active)
	f.service.Check(context.Background())
	expected := active.FiveHour.ResetsAt
	active.FiveHour.ResetsAt = now.Add(3 * time.Hour).Format(time.RFC3339)
	f.quotas.set(active)
	f.service.Check(context.Background())
	var confirmed string
	f.db.QueryRow(`SELECT confirmed_reset FROM codex_quota_activation_windows WHERE window_kind='five_hour'`).Scan(&confirmed)
	if confirmed != expected {
		t.Fatal("delayed snapshot moved confirmed cycle backward")
	}
	for range 2 {
		now = now.Add(5 * time.Minute)
		f.quotas.set(floatingQuota(now))
		f.service.Check(context.Background())
	}
	if f.sends.Load() != 0 {
		t.Fatal("known future cycle did not block repeated activation")
	}
}

func TestWeeklyOnlySlidingWindowUsesOneMessage(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	now := time.Now().UTC().Truncate(time.Second)
	f.service.now = func() time.Time { return now }
	f.service.sender = sendFunc(func(context.Context, account.Item, SendPlan) error {
		f.sends.Add(1)
		fresh := activeQuota()
		fresh.FiveHourEnabled = false
		fresh.FiveHour = nil
		f.quotas.set(fresh)
		return nil
	})
	for range 2 {
		q := floatingQuota(now)
		q.FiveHourEnabled = false
		q.FiveHour = nil
		q.Weekly.RemainingPercent = 100
		q.Weekly.ResetsAt = now.Add(7 * 24 * time.Hour).Format(time.RFC3339)
		f.quotas.set(q)
		f.service.Check(context.Background())
		now = now.Add(5 * time.Minute)
	}
	if f.sends.Load() != 1 {
		t.Fatal("weekly sliding timer did not use one message")
	}
	var n int
	f.db.QueryRow(`SELECT COUNT(*) FROM codex_quota_activation_windows WHERE window_kind='five_hour'`).Scan(&n)
	if n != 0 {
		t.Fatal("disabled five-hour window was recorded")
	}
}

func TestMissingTargetAfterUnconfirmedFutureDoesNotBypassFutureGuard(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	now := time.Now().UTC().Truncate(time.Second)
	f.service.now = func() time.Time { return now }
	active := floatingQuota(now)
	active.FiveHour.RemainingPercent = 99
	active.FiveHour.ResetsAt = now.Add(time.Hour).Format(time.RFC3339)
	f.quotas.set(active)
	f.service.Check(context.Background())
	now = now.Add(time.Hour + time.Second)
	f.quotas.set(floatingQuota(now))
	f.service.Check(context.Background())
	missing := floatingQuota(now)
	missing.FiveHour.ResetsAt = ""
	f.quotas.set(missing)
	f.service.Check(context.Background())
	if f.sends.Load() != 0 {
		t.Fatal("a temporary missing target bypassed future-target protection")
	}
	for range 2 {
		now = now.Add(5 * time.Minute)
		f.quotas.set(floatingQuota(now))
		f.service.Check(context.Background())
	}
	if f.sends.Load() != 1 {
		t.Fatal("subsequent proven sliding state did not recover safely")
	}
}
