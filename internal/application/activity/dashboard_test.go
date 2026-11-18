package activity

import (
	"context"
	"omniproxy/internal/core/proxy"
	"testing"
	"time"
)

func dashboardRange(now time.Time) DashboardQuery {
	start := now.UTC().Truncate(24 * time.Hour).Unix()
	return DashboardQuery{Boundaries: []int64{start, start + 86400}}
}

func TestDashboardCohortResultsTokensAndIdempotency(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	now := time.Now()
	for i, state := range []string{"success", "failed", "cancelled"} {
		e := event(i+1, "a@example.invalid", "a")
		e.StartedAt = now.UTC().Truncate(time.Hour).Add(time.Duration(i+1) * time.Second)
		e.State = state
		e.LatencyMS = int64((i + 1) * 1000)
		e.Usage = proxy.UsageSnapshot{Input: 10, Output: 5, InputKnown: true, OutputKnown: true, Complete: true}
		record(t, s, e)
		if err := s.Begin(ctx, e); err != nil {
			t.Fatal(err)
		}
		if err := s.Update(ctx, e); err != nil {
			t.Fatal(err)
		}
		if err := s.Finish(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	pending := event(4, "a@example.invalid", "a")
	pending.StartedAt = now.UTC().Truncate(time.Hour).Add(4 * time.Second)
	if err := s.Begin(ctx, pending); err != nil {
		t.Fatal(err)
	}
	info, err := s.Dashboard(ctx, dashboardRange(now))
	if err != nil {
		t.Fatal(err)
	}
	got := info.Summary
	if got.Requests != 4 || got.Success != 1 || got.Failed != 1 || got.Cancelled != 1 || got.Pending != 1 || got.TotalTokens != "45" || got.CompleteUsage != 3 || got.SuccessRate == nil || *got.SuccessRate != 50 || got.AverageLatencyMS == nil || *got.AverageLatencyMS != 1500 {
		t.Fatalf("incorrect summary: %+v", got)
	}
	if len(info.Recent) != 4 || len(info.Points) != 1 || info.HistoryIncomplete {
		t.Fatal("incorrect snapshot coverage")
	}
	select {
	case <-s.Updates():
	default:
		t.Fatal("missing committed update notice")
	}
}

func TestDashboardSurvivesRetentionAndResourceDeletion(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	now := time.Now()
	for i := 1; i <= 105; i++ {
		e := event(i, "a@example.invalid", "a")
		e.StartedAt = now.UTC().Truncate(time.Hour).Add(time.Duration(i) * time.Millisecond)
		e.LatencyMS = 10
		e.Usage = proxy.UsageSnapshot{Input: 1, Output: 2, Complete: true, InputKnown: true, OutputKnown: true}
		record(t, s, e)
	}
	if _, err := s.SetRetention(ctx, 100); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`DELETE FROM accounts`); err != nil {
		t.Fatal(err)
	}
	info, err := s.Dashboard(ctx, dashboardRange(now))
	if err != nil {
		t.Fatal(err)
	}
	if info.Summary.Requests != 105 || info.Summary.Success != 105 || info.Summary.TotalTokens != "315" || len(info.Recent) != 8 {
		t.Fatal("retention changed dashboard totals", info.Summary)
	}
}

func TestDashboardFailureRollsBackExistingAccounting(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	e := event(1, "a@example.invalid", "a")
	if err := s.Begin(ctx, e); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`CREATE TRIGGER reject_dashboard BEFORE UPDATE ON dashboard_buckets BEGIN SELECT RAISE(ABORT,'blocked'); END`); err != nil {
		t.Fatal(err)
	}
	e.Usage = proxy.UsageSnapshot{Input: 10, Output: 5, InputKnown: true, OutputKnown: true, Complete: true}
	if err := s.Update(ctx, e); err == nil {
		t.Fatal("failed rollup was ignored")
	}
	item, err := s.Request(ctx, e.ID)
	if err != nil || item.InputTokens != "0" || item.OutputTokens != "0" {
		t.Fatal("partial request committed", item, err)
	}
	usage, err := s.Usage(ctx)
	if err != nil || usage.AllTime.TotalTokens != "0" {
		t.Fatal("partial usage committed", usage, err)
	}
}

func TestDashboardRecoveryAndNoSample(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	empty, err := s.Dashboard(ctx, dashboardRange(time.Now()))
	if err != nil {
		t.Fatal(err)
	}
	if empty.Summary.SuccessRate != nil || empty.Summary.AverageLatencyMS != nil || empty.Summary.Requests != 0 || empty.Summary.TotalTokens != "0" {
		t.Fatal("empty sample fabricated values")
	}
	e := event(1, "a@example.invalid", "a")
	e.StartedAt = time.Now().Add(-time.Hour)
	if err := s.Begin(ctx, e); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := s.Recover(ctx); err != nil {
			t.Fatal(err)
		}
	}
	query := DashboardQuery{Boundaries: []int64{dashboardBucket(e.StartedAt), dashboardBucket(e.StartedAt) + 7200}}
	info, err := s.Dashboard(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	if info.Summary.Cancelled != 1 || info.Summary.Pending != 0 || info.Summary.SuccessRate != nil {
		t.Fatal("recovery double counted", info.Summary)
	}
}

func TestDashboardValidatesAndUsesQuarterHourTimezoneBoundaries(t *testing.T) {
	now := time.Now()
	base := dashboardBucket(now)
	for _, boundaries := range [][]int64{nil, {base}, {base, base}, {base, base - 900}, {base + 1, base + 3600}, {base - 40*86400, base}, {base, base + 100000}, {base, base + 3600, base + 900}} {
		if validateDashboard(DashboardQuery{Boundaries: boundaries}, now) == nil {
			t.Fatalf("accepted invalid range %v", boundaries)
		}
	}
	s := fixture(t)
	ctx := context.Background()
	start := base - 7200
	query := DashboardQuery{Boundaries: []int64{start + 1800, start + 5400, start + 9000}}
	for i, offset := range []int64{900, 1800, 5399, 5400} {
		e := event(i+1, "a@example.invalid", "a")
		e.StartedAt = time.Unix(start+offset, 0)
		record(t, s, e)
	}
	info, err := s.Dashboard(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	if info.Summary.Requests != 3 || info.Points[0].Summary.Requests != 2 || info.Points[1].Summary.Requests != 1 {
		t.Fatal("local half-hour boundary mismatch", info.Points)
	}
}

// 模型占比必须跟随所选时段：只统计被该时段覆盖的本地日期，前一日的用量不得混入。
func TestDashboardModelShareFollowsSelectedPeriod(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	local := time.Now().In(time.Local)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
	query := DashboardQuery{Boundaries: []int64{start.Unix(), start.AddDate(0, 0, 1).Unix()}}
	for i, row := range []struct {
		at     time.Time
		model  string
		input  int64
		output int64
	}{
		{start.Add(9 * time.Hour), "big", 100, 50},
		{start.Add(10 * time.Hour), "big", 100, 50},
		{start.Add(11 * time.Hour), "small", 10, 5},
		{start.AddDate(0, 0, -1).Add(9 * time.Hour), "stale", 500, 0},
	} {
		e := event(i+1, "a@example.invalid", "a")
		e.StartedAt = row.at
		e.FinishedAt = row.at.Add(time.Second)
		e.Model = row.model
		e.Usage = proxy.UsageSnapshot{Input: row.input, Output: row.output, InputKnown: true, OutputKnown: true, Complete: true}
		record(t, s, e)
	}
	info, err := s.Dashboard(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Models) != 2 {
		t.Fatalf("models must stay inside the selected period: %+v", info.Models)
	}
	if info.Models[0].Model != "big" || info.Models[0].TotalTokens != "300" || info.Models[0].InputTokens != "200" || info.Models[0].Requests != 2 {
		t.Fatalf("top model = %+v", info.Models[0])
	}
	if info.Models[1].Model != "small" || info.Models[1].TotalTokens != "15" {
		t.Fatalf("second model = %+v", info.Models[1])
	}
	if info.Summary.TotalTokens != "315" || info.Summary.Requests != 3 {
		t.Fatalf("period summary disagrees with model shares: %+v", info.Summary)
	}
}
