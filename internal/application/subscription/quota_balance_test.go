package subscription

import (
	"testing"
	"time"
)

func TestQuotaBalanceWeightUsesOnlyFreshCompleteMatchingObservations(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	for _, test := range []struct {
		name    string
		quota   Quota
		version string
		want    int
		fresh   bool
	}{
		{"weekly", Quota{FetchedAt: now.Format(time.RFC3339), Weekly: &QuotaWindow{RemainingPercent: 80}}, "v1", 80, true},
		{"limiting-window", Quota{FetchedAt: now.Format(time.RFC3339), Weekly: &QuotaWindow{RemainingPercent: 80}, FiveHourEnabled: true, FiveHour: &QuotaWindow{RemainingPercent: 20}}, "v1", 20, true},
		{"missing-window", Quota{FetchedAt: now.Format(time.RFC3339), Weekly: &QuotaWindow{RemainingPercent: 80}, FiveHourEnabled: true}, "v1", 0, false},
		{"stale", Quota{FetchedAt: now.Add(-2 * time.Minute).Format(time.RFC3339), Weekly: &QuotaWindow{RemainingPercent: 80}}, "v1", 0, false},
		{"changed-version", Quota{FetchedAt: now.Format(time.RFC3339), Weekly: &QuotaWindow{RemainingPercent: 80}}, "v2", 0, false},
		{"reset", Quota{FetchedAt: now.Format(time.RFC3339), Weekly: &QuotaWindow{RemainingPercent: 80, ResetsAt: now.Add(-time.Second).Format(time.RFC3339)}}, "v1", 0, false},
		{"zero", Quota{FetchedAt: now.Format(time.RFC3339), Weekly: &QuotaWindow{RemainingPercent: 0}}, "v1", 0, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := &Service{}
			s.rememberQuota("a", "v1", test.quota)
			weight, fresh := s.QuotaBalanceWeight("a", test.version, now)
			if weight != test.want || fresh != test.fresh {
				t.Fatalf("got weight=%d fresh=%v", weight, fresh)
			}
		})
	}
}
