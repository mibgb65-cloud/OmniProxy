package activity

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"
)

func TestFirstTokenSamplesAndUnknownsHaveSeparateSemantics(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	start := time.Now().Add(-time.Minute)
	for i, tc := range []struct {
		state   string
		stream  bool
		first   *int64
		latency int64
	}{
		{"success", true, intPointer(120), 1000}, {"success", true, intPointer(0), 800},
		{"failed", true, intPointer(500), 900}, {"cancelled", true, intPointer(400), 700},
		{"success", false, nil, 1200}, {"success", true, nil, 1000},
	} {
		e := event(i+1, "member@example.invalid", "a")
		e.StartedAt = start
		e.State, e.Stream, e.FirstTokenMS, e.LatencyMS = tc.state, tc.stream, tc.first, tc.latency
		record(t, s, e)
		item, err := s.Request(ctx, e.ID)
		if err != nil || (item.FirstTokenMS == nil) != (tc.first == nil) {
			t.Fatal("unknown timing became a number", item, err)
		}
		if tc.first != nil && *item.FirstTokenMS != *tc.first {
			t.Fatal("timing changed")
		}
	}
	oldPending := event(30, "member@example.invalid", "a")
	oldPending.StartedAt = start.Add(-3 * time.Hour)
	if err := s.Begin(ctx, oldPending); err != nil {
		t.Fatal(err)
	}
	info, err := s.Dashboard(ctx, DashboardQuery{Boundaries: []int64{dashboardBucket(start), dashboardBucket(start) + 7200}})
	if err != nil || info.Summary.FirstTokenSamples != 2 || info.Summary.AverageFirstTokenMS == nil || *info.Summary.AverageFirstTokenMS != 60 || *info.Summary.AverageLatencyMS != 980 {
		t.Fatal("wrong sample population", info, err)
	}
	if info.PendingNow == nil || *info.PendingNow != 1 || info.Summary.Pending != 0 {
		t.Fatal("current in-flight count incorrectly scoped to range", info)
	}
	if len(info.ProviderDistribution) != 1 || info.ProviderDistribution[0].Requests != 6 {
		t.Fatal("provider cumulative distribution missing", info.ProviderDistribution)
	}
}

func TestFirstTokenAggregateSurvivesRetentionAndDuplicateFinish(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	start := time.Now().Add(-time.Minute)
	if _, err := s.SetRetention(ctx, 100); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 102; i++ {
		e := event(i, "member@example.invalid", "a")
		e.StartedAt = start
		e.Stream, e.FirstTokenMS, e.LatencyMS = true, intPointer(50), 100
		record(t, s, e)
		if err := s.Finish(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	info, err := s.Dashboard(ctx, DashboardQuery{Boundaries: []int64{dashboardBucket(start), dashboardBucket(start) + 7200}})
	if err != nil || info.Summary.FirstTokenSamples != 102 || *info.Summary.AverageFirstTokenMS != 50 || info.ProviderDistribution[0].Requests != 102 {
		t.Fatal("retention changed aggregate", info, err)
	}
	var count int
	if s.db.QueryRow(`SELECT COUNT(*) FROM request_records`).Scan(&count) != nil || count != 100 {
		t.Fatal("fixture was not pruned")
	}
}

func TestInvalidFirstTokenAndOverflowDoNotPartiallyCommit(t *testing.T) {
	for _, value := range []int64{-1, 101} {
		s := fixture(t)
		e := event(1, "member@example.invalid", "a")
		e.Stream, e.FirstTokenMS, e.LatencyMS = true, intPointer(value), 100
		if err := s.Finish(context.Background(), e); !errors.Is(err, ErrInvalid) {
			t.Fatal("out of range timing accepted", err)
		}
	}
	s := fixture(t)
	ctx := context.Background()
	e := event(2, "member@example.invalid", "a")
	e.StartedAt = time.Now()
	e.Stream, e.FirstTokenMS, e.LatencyMS = true, intPointer(1), 100
	if err := s.Begin(ctx, e); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(ctx, e); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE dashboard_buckets SET first_token_ms=?`, int64(math.MaxInt64)); err != nil {
		t.Fatal(err)
	}
	if err := s.Finish(ctx, e); !errors.Is(err, ErrUnavailable) {
		t.Fatal("overflow accepted", err)
	}
	item, err := s.Request(ctx, e.ID)
	if err != nil || item.State != "pending" || item.FirstTokenMS != nil {
		t.Fatal("failed aggregate partially committed", item, err)
	}
}

func intPointer(value int64) *int64 { return &value }
