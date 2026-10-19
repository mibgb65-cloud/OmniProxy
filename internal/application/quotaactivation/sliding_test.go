package quotaactivation

import (
	"context"
	"testing"
	"time"

	"omniproxy/internal/application/subscription"
)

func floatingQuota(at time.Time) subscription.Quota {
	return subscription.Quota{FiveHourEnabled: true, FetchedAt: at.UTC().Format(time.RFC3339), FiveHour: &subscription.QuotaWindow{RemainingPercent: 100, ResetsAt: at.Add(5 * time.Hour).UTC().Format(time.RFC3339)}, Weekly: &subscription.QuotaWindow{RemainingPercent: 55, ResetsAt: at.Add(6 * 24 * time.Hour).UTC().Format(time.RFC3339)}}
}

func TestSlidingUnstartedTimerIsNotMistakenForActive(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	now := time.Now().UTC().Truncate(time.Second)
	f.service.now = func() time.Time { return now }
	f.quotas.set(floatingQuota(now))
	if err := f.service.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if f.sends.Load() != 0 || f.service.Status().Activated != 0 || f.service.Status().Pending != 1 {
		t.Fatal("first ambiguous timer was treated as activated", f.service.Status())
	}
	now = now.Add(5 * time.Minute)
	f.quotas.set(floatingQuota(now))
	if err := f.service.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if f.sends.Load() != 1 || f.service.Status().Sent != 1 || f.service.Status().Activated != 1 {
		t.Fatal("confirmed sliding timer did not activate", f.service.Status())
	}
}

func TestQuotaReadCanWakeSlidingObservationWithoutAuthorizingAnImmediateSend(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	q := floatingQuota(now)
	if NeedsActivation(q) || !NeedsObservation(q) {
		t.Fatal("sliding read did not distinguish observation and send authorization")
	}
	q.FiveHour.ResetsAt = now.Add(time.Hour).Format(time.RFC3339)
	if NeedsObservation(q) {
		t.Fatal("fixed active timer requested observation")
	}
	q = floatingQuota(now)
	q.Weekly.RemainingPercent = 0
	if NeedsObservation(q) {
		t.Fatal("exhausted account woke activation")
	}
}
