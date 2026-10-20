package activity

import (
	"context"
	"math"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/subscription"
	"omniproxy/internal/core/proxy"
)

func estimateFixture(t *testing.T) (*Service, account.Item, subscription.Quota, time.Time) {
	t.Helper()
	s := fixture(t)
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	item := account.Item{ID: "a", ProviderID: "builtin_openai", AuthMethod: account.AuthSubscription, AuthEmail: "a@example.invalid", AuthAccountID: "shared-team"}
	quota := subscription.Quota{FetchedAt: now.Format(time.RFC3339), FiveHourEnabled: true,
		FiveHour: &subscription.QuotaWindow{RemainingPercent: 90, ResetsAt: now.Add(3 * time.Hour).Format(time.RFC3339)},
		Weekly:   &subscription.QuotaWindow{RemainingPercent: 80, ResetsAt: now.Add(3 * 24 * time.Hour).Format(time.RFC3339)}}
	return s, item, quota, now
}

func estimateEvent(t *testing.T, s *Service, id int, at time.Time, model string, cached *int64) proxy.RequestEvent {
	t.Helper()
	e := event(id, "a@example.invalid", "a")
	e.StartedAt, e.FinishedAt = at, at.Add(time.Second)
	e.Model = model
	e.Usage = proxy.UsageSnapshot{Input: 1000000, Output: 100000, InputKnown: true, OutputKnown: true, Complete: true, CachedInput: cached}
	record(t, s, e)
	return e
}

func closeEnough(a, b float64) bool { return math.Abs(a-b) < 1e-8 }

func TestQuotaEstimateUsesActualModelsCacheAndSeparateWindows(t *testing.T) {
	s, item, quota, now := estimateFixture(t)
	cached := int64(800000)
	estimateEvent(t, s, 1, now.Add(-time.Hour), "gpt-6.1-sol", &cached)
	// 周窗口内、5h 窗口外：不能把两个窗口的 Token 混在一起。
	estimateEvent(t, s, 2, now.Add(-3*time.Hour), "gpt-5.3-codex", &cached)
	// 当前周窗口之外的累计用量不能成为回退样本。
	estimateEvent(t, s, 3, now.Add(-8*24*time.Hour), "gpt-6-astra", &cached)
	got, err := s.EstimateQuota(context.Background(), item, quota, now)
	if err != nil || got.FiveHour.Status != "ready" || !closeEnough(got.FiveHour.UsedMin, 1.48) || !closeEnough(got.FiveHour.Total, 14.8) || !closeEnough(got.FiveHour.Remaining, 13.32) {
		t.Fatal("5h cost or legacy scaling incorrect", got, err)
	}
	if got.Weekly.Requests != 2 || !closeEnough(got.Weekly.UsedMax, 3.37) || !closeEnough(got.Weekly.TotalMax, 16.85) {
		t.Fatal("actual model price or window isolation lost", got.Weekly)
	}
	other := item
	other.AuthEmail = "other@example.invalid"
	got, err = s.EstimateQuota(context.Background(), other, quota, now)
	if err != nil || got.Weekly.Status != "no_usage" {
		t.Fatal("estimate crossed member boundary", err)
	}
	other = item
	other.AuthAccountID = "other-workspace"
	got, err = s.EstimateQuota(context.Background(), other, quota, now)
	if err != nil || got.Weekly.Status != "no_usage" {
		t.Fatal("estimate crossed workspace boundary", err)
	}
}

func TestQuotaEstimateHistoricalUnknownCacheUsesUndiscountedPoint(t *testing.T) {
	s, item, quota, now := estimateFixture(t)
	estimateEvent(t, s, 1, now.Add(-time.Hour), "gpt-6.1-sol", nil)
	got, err := s.EstimateQuota(context.Background(), item, quota, now)
	if err != nil || !closeEnough(got.Weekly.TotalMin, 5.5) || !closeEnough(got.Weekly.TotalMax, 15) {
		t.Fatal("legacy bounds changed", got, err)
	}
	if !closeEnough(got.Weekly.Total, 15) || !closeEnough(got.Weekly.Remaining, 12) {
		t.Fatal("point estimate invented a historical cache discount", got.Weekly)
	}
}

func TestQuotaEstimateUsesEachWindowsObservationTime(t *testing.T) {
	s, item, quota, now := estimateFixture(t)
	quota.Weekly.ObservedAt = now.Add(-2 * time.Hour).Format(time.RFC3339)
	estimateEvent(t, s, 1, now.Add(-3*time.Hour), "gpt-6.1-sol", nil)
	estimateEvent(t, s, 2, now.Add(-time.Hour), "gpt-6.1-sol", nil)
	got, err := s.EstimateQuota(context.Background(), item, quota, now)
	if err != nil || got.Weekly.Requests != 1 || got.FiveHour.Requests != 1 {
		t.Fatal("new tokens were paired with a retained older window percentage", got, err)
	}
}

func TestQuotaEstimateUnknownModelThresholdExpiryAndIncompleteUsage(t *testing.T) {
	for _, kind := range []string{"unknown_price", "insufficient_usage", "expired", "missing_usage", "no_usage", "zero_remaining"} {
		t.Run(kind, func(t *testing.T) {
			s, item, quota, now := estimateFixture(t)
			switch kind {
			case "unknown_price":
				estimateEvent(t, s, 1, now.Add(-time.Hour), "not-priced", nil)
			case "insufficient_usage":
				quota.Weekly.RemainingPercent = 96
			case "expired":
				quota.FetchedAt = now.Add(-4 * 24 * time.Hour).Format(time.RFC3339)
				quota.Weekly.ResetsAt = now.Add(-time.Hour).Format(time.RFC3339)
			case "missing_usage":
				e := event(1, item.AuthEmail, item.ID)
				e.StartedAt, e.FinishedAt = now.Add(-time.Hour), now.Add(-time.Minute)
				record(t, s, e)
			case "zero_remaining":
				quota.Weekly.RemainingPercent = 0
				estimateEvent(t, s, 1, now.Add(-time.Hour), "gpt-6.1-sol", nil)
			}
			got, err := s.EstimateQuota(context.Background(), item, quota, now)
			want := kind
			if kind == "zero_remaining" {
				want = "ready"
			}
			if err != nil || got.Weekly.Status != want {
				t.Fatal("unsafe estimate state", got.Weekly, err)
			}
			if kind == "zero_remaining" && got.Weekly.RemainingMax != 0 {
				t.Fatal("exhausted quota acquired a remaining dollar value")
			}
		})
	}
}

func TestQuotaEstimateEarlyResetCutoffDoesNotAffectOtherWindow(t *testing.T) {
	s, item, quota, now := estimateFixture(t)
	if _, err := s.db.Exec(`INSERT INTO accounts(id,provider_id,name,created_at,updated_at) VALUES('a','builtin_openai','test','now','now')`); err != nil {
		t.Fatal(err)
	}
	estimateEvent(t, s, 1, now.Add(-time.Hour), "gpt-6.1-sol", nil)
	estimateEvent(t, s, 2, now.Add(-10*time.Minute), "gpt-6.1-sol", nil)
	if _, err := s.db.Exec(`INSERT INTO subscription_quota_estimate_baselines VALUES(?,?,?,?)`, item.ID, "weekly", quota.Weekly.ResetsAt, now.Add(-30*time.Minute).Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	got, err := s.EstimateQuota(context.Background(), item, quota, now)
	if err != nil || !got.Weekly.ResetAdjusted || got.Weekly.Requests != 1 || got.FiveHour.ResetAdjusted || got.FiveHour.Requests != 2 {
		t.Fatal("early reset mixed old usage or affected the unrelated window", got, err)
	}
}
