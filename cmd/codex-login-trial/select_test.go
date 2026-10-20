package main

import (
	"testing"

	"omniproxy/internal/application/subscription"
)

func TestAvailableQuotaRequiresCompleteNonExhaustedWindows(t *testing.T) {
	window := func(remaining int) *subscription.QuotaWindow {
		return &subscription.QuotaWindow{RemainingPercent: remaining}
	}
	for _, test := range []struct {
		name      string
		quota     subscription.Quota
		remaining int
		known     bool
	}{
		{"missing-weekly", subscription.Quota{}, 0, false},
		{"missing-five-hour", subscription.Quota{FiveHourEnabled: true, Weekly: window(80)}, 0, false},
		{"weekly-exhausted", subscription.Quota{FiveHourEnabled: true, Weekly: window(0), FiveHour: window(80)}, 0, true},
		{"five-hour-exhausted", subscription.Quota{FiveHourEnabled: true, Weekly: window(90), FiveHour: window(0)}, 0, true},
		{"limiting-window", subscription.Quota{FiveHourEnabled: true, Weekly: window(70), FiveHour: window(40)}, 40, true},
		{"no-five-hour-limit", subscription.Quota{Weekly: window(70), FiveHour: window(0)}, 70, true},
		{"invalid-percentage", subscription.Quota{Weekly: window(101)}, 0, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			remaining, known := availableQuota(test.quota)
			if remaining != test.remaining || known != test.known {
				t.Fatal("incomplete quota or exhausted window was misclassified")
			}
		})
	}
}
