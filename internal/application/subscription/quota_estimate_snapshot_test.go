package subscription

import "testing"

func TestEstimateCannotUseAnInferredResetWithANewPercentage(t *testing.T) {
	previous := Quota{FetchedAt: "2026-10-20T09:00:00Z",
		Weekly: &QuotaWindow{RemainingPercent: 80, ResetsAt: "2026-10-23T09:00:00Z", ObservedAt: "2026-10-20T09:00:00Z"}}
	next := Quota{FetchedAt: "2026-10-20T10:00:00Z",
		Weekly: &QuotaWindow{RemainingPercent: 70, ObservedAt: "2026-10-20T10:00:00Z"}}
	merged := mergeQuotaSnapshot(previous, next)
	if !merged.FromCache || merged.Weekly.ResetsAt == "" || merged.Weekly.ObservedAt != "" || merged.Weekly.RemainingPercent != 70 {
		t.Fatal("display-only reset inference became usable cost evidence")
	}
	missing := mergeQuotaSnapshot(previous, Quota{FetchedAt: "2026-10-20T10:00:00Z"})
	if missing.Weekly.ObservedAt != previous.Weekly.ObservedAt {
		t.Fatal("retained old percentage acquired a new observation time")
	}
}
