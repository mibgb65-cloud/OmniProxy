package quotaactivation

import (
	"context"
	"errors"
	"omniproxy/internal/application/account"
	"sync"
	"testing"
	"time"
)

func TestStableOrInsufficientTimingEvidenceNeverSends(t *testing.T) {
	for _, name := range []string{"stable", "too-soon", "long-gap", "clock-rollback", "irregular-shift", "exhausted", "missing-fetch", "invalid-reset"} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			f.enable(t)
			now := time.Now().UTC().Truncate(time.Second)
			f.service.now = func() time.Time { return now }
			first := floatingQuota(now)
			if name == "exhausted" {
				first.Weekly.RemainingPercent = 0
			}
			f.quotas.set(first)
			f.service.Check(context.Background())
			gap := 5 * time.Minute
			switch name {
			case "too-soon":
				gap = 20 * time.Second
			case "long-gap":
				gap = 16 * time.Minute
			case "clock-rollback":
				gap = -time.Minute
			}
			now = now.Add(gap)
			next := floatingQuota(now)
			switch name {
			case "stable":
				next.FiveHour.ResetsAt = first.FiveHour.ResetsAt
			case "irregular-shift":
				next.FiveHour.ResetsAt = now.Add(5*time.Hour + 30*time.Second).Format(time.RFC3339)
			case "exhausted":
				next.Weekly.RemainingPercent = 0
			case "missing-fetch":
				next.FetchedAt = ""
			case "invalid-reset":
				next.FiveHour.ResetsAt = "invalid"
			}
			f.quotas.set(next)
			f.service.Check(context.Background())
			if f.sends.Load() != 0 {
				t.Fatal("unsafe evidence sent a message", name)
			}
			if name == "stable" && f.service.Status().Activated != 1 {
				t.Fatal("fixed 100-percent timer was not recognized as active")
			}
		})
	}
}

func TestFiveHourOneMinuteRoundingCanBeObservedWithoutStringMatching(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	now := time.Now().UTC().Truncate(time.Second)
	f.service.now = func() time.Time { return now }
	for range 2 {
		q := floatingQuota(now)
		q.FiveHour.ResetsAt = now.Add(5*time.Hour + time.Minute).Format(time.RFC3339)
		f.quotas.set(q)
		f.service.Check(context.Background())
		now = now.Add(5 * time.Minute)
	}
	if f.sends.Load() != 1 {
		t.Fatal("rounding tolerance did not permit confirmed sliding timer")
	}
}

func TestSlidingPendingSurvivesRestartAndLongDelayWithoutResending(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	now := time.Now().UTC().Truncate(time.Second)
	f.service.now = func() time.Time { return now }
	sender := sendFunc(func(context.Context, account.Item, SendPlan) error {
		f.sends.Add(1)
		return errors.New("unknown network result")
	})
	f.service.sender = sender
	f.quotas.set(floatingQuota(now))
	f.service.Check(context.Background())
	now = now.Add(5 * time.Minute)
	f.quotas.set(floatingQuota(now))
	f.service.Check(context.Background())
	if f.sends.Load() != 1 || f.service.Status().Activated != 0 {
		t.Fatal("uncertain result marked active")
	}
	restarted := New(f.db, f.accounts, f.providers, f.quotas, sender)
	restarted.now = func() time.Time { return now }
	for _, gap := range []time.Duration{5 * time.Minute, 6 * time.Hour, 5 * time.Minute} {
		now = now.Add(gap)
		f.quotas.set(floatingQuota(now))
		restarted.Check(context.Background())
	}
	if f.sends.Load() != 1 {
		t.Fatal("restart or expired placeholder caused duplicate send")
	}
	var attempted string
	f.db.QueryRow(`SELECT attempted_cycle FROM codex_quota_activation_windows WHERE window_kind='five_hour'`).Scan(&attempted)
	if attempted != "initial" {
		t.Fatal("uncertain claim lost")
	}
}

func TestConfirmedCycleExpiresAndOnlyThenMaySend(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	now := time.Now().UTC().Truncate(time.Second)
	f.service.now = func() time.Time { return now }
	active := floatingQuota(now)
	active.FiveHour.RemainingPercent = 99
	reset := now.Add(time.Hour)
	active.FiveHour.ResetsAt = reset.Format(time.RFC3339)
	f.quotas.set(active)
	f.service.Check(context.Background())
	now = reset.Add(time.Second)
	f.quotas.set(floatingQuota(now))
	f.service.Check(context.Background())
	now = now.Add(5 * time.Minute)
	f.quotas.set(floatingQuota(now))
	f.service.Check(context.Background())
	if f.sends.Load() != 1 {
		t.Fatal("new verified cycle not sent")
	}
	var attempted string
	f.db.QueryRow(`SELECT attempted_cycle FROM codex_quota_activation_windows WHERE window_kind='five_hour'`).Scan(&attempted)
	if attempted != reset.Format(time.RFC3339) {
		t.Fatal("verified cycle boundary not used")
	}
}

func TestConcurrentChecksClaimSlidingCycleOnlyOnce(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	now := time.Now().UTC().Truncate(time.Second)
	f.service.now = func() time.Time { return now }
	f.quotas.set(floatingQuota(now))
	f.service.Check(context.Background())
	now = now.Add(5 * time.Minute)
	f.quotas.set(floatingQuota(now))
	var wait sync.WaitGroup
	for range 6 {
		wait.Add(1)
		go func() { defer wait.Done(); f.service.Check(context.Background()) }()
	}
	wait.Wait()
	if f.sends.Load() != 1 {
		t.Fatal("concurrent sliding checks duplicated sends")
	}
}
