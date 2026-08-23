package main

import (
	"testing"
	"time"

	"omniproxy/internal/token"
)

func TestCodexWeeklyQuotaWindowResetDetectsWeeklyCounterChanges(t *testing.T) {
	consumedAt := time.Unix(100, 0)
	tests := []struct {
		name   string
		before token.UsageInfo
		after  token.UsageInfo
		want   bool
	}{
		{
			name:   "weekly percentage dropped",
			before: token.UsageInfo{SecondaryUsedPercentExact: 42.5, SecondaryResetAt: 200},
			after:  token.UsageInfo{SecondaryUsedPercentExact: 1.5, SecondaryResetAt: 200},
			want:   true,
		},
		{
			name:   "weekly reset time changed",
			before: token.UsageInfo{SecondaryUsedPercent: 0, SecondaryResetAt: 200},
			after:  token.UsageInfo{SecondaryUsedPercent: 0, SecondaryResetAt: 300},
			want:   true,
		},
		{
			name: "only five hour window reset",
			before: token.UsageInfo{
				PrimaryUsedPercent: 80, SecondaryUsedPercent: 30, SecondaryResetAt: 200,
			},
			after: token.UsageInfo{
				PrimaryUsedPercent: 0, SecondaryUsedPercent: 30, SecondaryResetAt: 200,
			},
			want: false,
		},
		{
			name:   "expired weekly window rolled naturally",
			before: token.UsageInfo{SecondaryUsedPercent: 80, SecondaryResetAt: 90},
			after:  token.UsageInfo{SecondaryUsedPercent: 0, SecondaryResetAt: 300},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := codexWeeklyQuotaWindowReset(tt.before, tt.after, consumedAt); got != tt.want {
				t.Fatalf("codexWeeklyQuotaWindowReset() = %v, want %v", got, tt.want)
			}
		})
	}
}
