package subscription

import (
	"context"
	"testing"

	"omniproxy/internal/infrastructure/codex"
)

func quotaSnapshot(plan string, five, week int32) *codex.RateLimitSnapshot {
	fiveMins, weekMins, reset := int64(300), int64(10080), int64(2000000000)
	return &codex.RateLimitSnapshot{Plan: plan, LimitID: "codex", Primary: &codex.RateLimitWindow{UsedPercent: &five, DurationMins: &fiveMins, ResetsAt: &reset}, Secondary: &codex.RateLimitWindow{UsedPercent: &week, DurationMins: &weekMins}}
}

func TestQuotaWindowsFollowPlanAndActualDuration(t *testing.T) {
	for _, plan := range []string{"plus", "team", "pro", "prolite", "free", "business", "self_serve_business_usage_based"} {
		t.Run(plan, func(t *testing.T) {
			quota, err := normalizeQuota(codex.RateLimits{Snapshot: quotaSnapshot(plan, 25, 60)}, "")
			if err != nil || quota.Weekly == nil || quota.Weekly.RemainingPercent != 40 {
				t.Fatal("weekly quota missing or incorrectly normalized")
			}
			dual := plan == "plus" || plan == "team"
			if quota.FiveHourEnabled != dual || (quota.FiveHour != nil) != dual {
				t.Fatal("five-hour window does not follow requested plan rule")
			}
			if dual && (quota.FiveHour.RemainingPercent != 75 || quota.FiveHour.ResetsAt == "") {
				t.Fatal("five-hour quota or reset time incorrect")
			}
		})
	}
}

func TestQuotaSelectsCodexBucketAndDoesNotInventMissingWindows(t *testing.T) {
	other := quotaSnapshot("pro", 80, 90)
	other.LimitID = "codex_other"
	primary := quotaSnapshot("plus", 120, 0)
	primary.Secondary = nil
	quota, err := normalizeQuota(codex.RateLimits{Snapshot: other, ByLimitID: map[string]*codex.RateLimitSnapshot{"codex": primary}}, "team")
	if err != nil || quota.FiveHour.RemainingPercent != 0 || quota.Weekly != nil {
		t.Fatal("selected wrong model bucket or fabricated a weekly window")
	}
	primary.Primary.DurationMins = nil
	quota, err = normalizeQuota(codex.RateLimits{Snapshot: primary}, "")
	if err != nil || quota.FiveHour != nil || quota.Weekly != nil {
		t.Fatal("guessed a missing duration from window position")
	}
	if _, err := normalizeQuota(codex.RateLimits{Snapshot: other}, "team"); err == nil {
		t.Fatal("other model bucket treated as standard Codex quota")
	}
	primary.Plan = "unknown"
	quota, _ = normalizeQuota(codex.RateLimits{Snapshot: primary}, "team")
	if !quota.FiveHourEnabled {
		t.Fatal("confirmed plan fallback was lost")
	}
}

func TestQuotaReadProtectsAccountAssociationAndAuthorization(t *testing.T) {
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.authorize("quota@example.test")
	item := awaitState(t, f.manager, view.ID, "completed").Account
	quota, err := f.manager.ReadQuota(context.Background(), item.ID)
	if err != nil || quota.Weekly == nil || quota.Weekly.RemainingPercent != 100 {
		t.Fatal("managed account quota query failed")
	}
	client.mu.Lock()
	client.rateLimits = &codex.RateLimits{AccountID: "different-account", Snapshot: quotaSnapshot("team", 0, 0)}
	client.mu.Unlock()
	if _, err := f.manager.ReadQuota(context.Background(), item.ID); err == nil || err.Error() != "subscription: identity_changed" {
		t.Fatal("quota associated with a different identity")
	}
	client.mu.Lock()
	client.failQuota = true
	client.mu.Unlock()
	if _, err := f.manager.ReadQuota(context.Background(), item.ID); err == nil || err.Error() != "subscription: quota_unavailable" {
		t.Fatal("raw quota failure exposed")
	}
	saved, _ := f.accounts.Metadata(context.Background(), item.ID)
	if saved.AuthState != "authenticated" || client.logoutCount != 0 {
		t.Fatal("quota failure damaged authorization")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := f.manager.ReadQuota(cancelled, item.ID); err == nil {
		t.Fatal("cancelled quota read continued")
	}
	if _, err := f.manager.Logout(context.Background(), item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.manager.ReadQuota(context.Background(), item.ID); err == nil || err.Error() != "subscription: quota_not_authenticated" {
		t.Fatal("logged-out account queried as authorized")
	}
}
