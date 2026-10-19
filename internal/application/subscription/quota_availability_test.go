package subscription

import (
	"context"
	"omniproxy/internal/infrastructure/codex"
	"testing"
	"time"
)

func TestQuotaAvailabilityHandlesBothWindowsAndReset(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	for _, which := range []string{"five-hour", "weekly", "both", "healthy", "missing", "five-disabled"} {
		t.Run(which, func(t *testing.T) {
			s := &Service{}
			quota := Quota{FiveHourEnabled: true, FetchedAt: now.Format(time.RFC3339), FiveHour: &QuotaWindow{RemainingPercent: 80, ResetsAt: now.Add(time.Hour).Format(time.RFC3339)}, Weekly: &QuotaWindow{RemainingPercent: 80, ResetsAt: now.Add(7 * 24 * time.Hour).Format(time.RFC3339)}}
			want := time.Time{}
			if which == "five-hour" || which == "both" {
				quota.FiveHour.RemainingPercent = 0
				want = now.Add(time.Hour)
			}
			if which == "weekly" || which == "both" {
				quota.Weekly.RemainingPercent = 0
				want = now.Add(7 * 24 * time.Hour)
			}
			if which == "missing" {
				quota.FiveHour = nil
				quota.Weekly = nil
			}
			if which == "five-disabled" {
				quota.FiveHourEnabled = false
				quota.FiveHour.RemainingPercent = 0
			}
			s.rememberQuota("account", "version", quota)
			until, blocked := s.QuotaBlockedUntil("account", "version", now)
			if blocked != !want.IsZero() || blocked && !until.Equal(want) {
				t.Fatal("wrong shared quota gate", until, blocked, want)
			}
			if _, blocked = s.QuotaBlockedUntil("account", "new-version", now); blocked {
				t.Fatal("stale credential quota reused")
			}
			if !want.IsZero() {
				if _, blocked = s.QuotaBlockedUntil("account", "version", want); blocked {
					t.Fatal("reset account remained blocked")
				}
			}
		})
	}
}

func TestQuotaAvailabilityCopiesAndBoundsObservations(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	s := &Service{}
	q := Quota{FiveHourEnabled: true, FetchedAt: now.Format(time.RFC3339), FiveHour: &QuotaWindow{RemainingPercent: 0, ResetsAt: now.Add(time.Hour).Format(time.RFC3339)}}
	s.rememberQuota("a", "v", q)
	q.FiveHour.RemainingPercent = 100
	if _, blocked := s.QuotaBlockedUntil("a", "v", now); !blocked {
		t.Fatal("caller mutated cached quota")
	}
	s.rememberQuota("a", "v", q)
	if _, blocked := s.QuotaBlockedUntil("a", "v", now); blocked {
		t.Fatal("fresh usable quota did not recover account")
	}
	q.FiveHour.RemainingPercent = 0
	q.FiveHour.ResetsAt = ""
	s.rememberQuota("a", "v", q)
	if _, blocked := s.QuotaBlockedUntil("a", "v", now); !blocked {
		t.Fatal("fresh exhaustion without reset ignored")
	}
	if _, blocked := s.QuotaBlockedUntil("a", "v", now.Add(2*time.Minute)); blocked {
		t.Fatal("unknown reset blocked indefinitely")
	}
	s.RecordQuotaRejection("a", "v", now.Add(7*24*time.Hour))
	if _, blocked := s.QuotaBlockedUntil("a", "v", now.Add(time.Hour)); !blocked {
		t.Fatal("weekly rejection not shared across models")
	}
	s.RecordQuotaRejection("a", "v", now.Add(100*24*time.Hour))
	until, _ := s.QuotaBlockedUntil("a", "v", now)
	if until.After(now.Add(8 * 24 * time.Hour)) {
		t.Fatal("unbounded quota denial")
	}
	for i := 0; i < maxQuotaObservations+10; i++ {
		s.RecordQuotaRejection(time.Unix(int64(i), 0).String(), "v", now.Add(time.Hour))
	}
	if len(s.quotaObservations) > maxQuotaObservations {
		t.Fatal("unbounded observation cache")
	}
}

func TestQuotaReadFeedsGatewayAvailability(t *testing.T) {
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.authorize("quota@example.test")
	item := awaitState(t, f.manager, view.ID, "completed").Account
	client.mu.Lock()
	client.rateLimits = &codex.RateLimits{AccountID: item.AuthAccountID, Snapshot: quotaSnapshot("plus", 0, 100)}
	client.mu.Unlock()
	if _, err := f.manager.ReadQuota(context.Background(), item.ID); err != nil {
		t.Fatal(err)
	}
	until, blocked := f.manager.QuotaBlockedUntil(item.ID, item.UpdatedAt, time.Now())
	if !blocked || until.IsZero() {
		t.Fatal("verified weekly exhaustion was not observed")
	}
}

func TestQuotaAvailabilityDoesNotRecoverFromMissingOrOlderWindow(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	s := &Service{}
	blocked := Quota{FetchedAt: now.Format(time.RFC3339), Weekly: &QuotaWindow{RemainingPercent: 0, ResetsAt: now.Add(7 * 24 * time.Hour).Format(time.RFC3339)}}
	s.rememberQuota("a", "v", blocked)
	s.rememberQuota("a", "v", Quota{FetchedAt: now.Add(time.Second).Format(time.RFC3339)})
	if _, limited := s.QuotaBlockedUntil("a", "v", now); !limited {
		t.Fatal("missing weekly window cleared exhaustion")
	}
	s.rememberQuota("a", "v", Quota{FetchedAt: now.Add(-time.Second).Format(time.RFC3339), Weekly: &QuotaWindow{RemainingPercent: 100}})
	if _, limited := s.QuotaBlockedUntil("a", "v", now); !limited {
		t.Fatal("older healthy snapshot cleared exhaustion")
	}
	s.rememberQuota("a", "v", Quota{FetchedAt: now.Add(2 * time.Second).Format(time.RFC3339), Weekly: &QuotaWindow{RemainingPercent: 100}})
	if _, limited := s.QuotaBlockedUntil("a", "v", now); limited {
		t.Fatal("fresh weekly recovery ignored")
	}
}
