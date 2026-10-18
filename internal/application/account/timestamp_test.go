package account

import (
	"errors"
	"testing"
	"time"
)

func TestTimestampAfterRepeatedOrRolledBackClock(t *testing.T) {
	base := time.Date(2026, 10, 18, 12, 0, 0, 123456700, time.UTC)
	for _, tc := range []struct {
		name, previous string
		current        time.Time
		want           time.Time
	}{
		{"same-clock", base.Format(time.RFC3339Nano), base, base.Add(time.Nanosecond)},
		{"rollback", base.Format(time.RFC3339Nano), base.Add(-24 * time.Hour), base.Add(time.Nanosecond)},
		{"normal", base.Format(time.RFC3339Nano), base.Add(time.Second), base.Add(time.Second)},
		{"timezone", base.In(time.FixedZone("offset", 8*3600)).Format(time.RFC3339Nano), base, base.Add(time.Nanosecond)},
		{"legacy", "legacy-version", base, base},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := timestampAfter(tc.previous, tc.current)
			if err != nil || got != tc.want.Format(time.RFC3339Nano) {
				t.Fatal("incorrect timestamp boundary", got, err)
			}
			if _, err := time.Parse(time.RFC3339Nano, got); err != nil {
				t.Fatal("timestamp cannot round trip", err)
			}
		})
	}
	previous := base.Format(time.RFC3339Nano)
	for i := 1; i <= 100; i++ {
		next, err := timestampAfter(previous, base)
		if err != nil || next != base.Add(time.Duration(i)*time.Nanosecond).Format(time.RFC3339Nano) {
			t.Fatal("fixed clock reused a timestamp", err)
		}
		previous = next
	}
}

func TestTimestampAfterRejectsUnrepresentableTime(t *testing.T) {
	for _, tc := range []struct {
		previous string
		current  time.Time
	}{
		{"9999-12-31T23:59:59.999999999Z", time.Now()},
		{"", time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"", time.Date(-1, 1, 1, 0, 0, 0, 0, time.UTC)},
	} {
		if value, err := timestampAfter(tc.previous, tc.current); value != "" || !errors.Is(err, ErrInvalidInput) {
			t.Fatal("invalid time was accepted", err)
		}
	}
}
