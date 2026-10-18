package activity

import (
	"context"
	"omniproxy/internal/core/proxy"
	"testing"
	"time"
)

func TestAttemptIdempotenceImmutabilityAndRetention(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	e := event(500, "member@example.invalid", "a")
	if err := s.Begin(ctx, e); err != nil {
		t.Fatal(err)
	}
	a := proxy.AttemptEvent{RequestID: e.ID, Index: 1, StartedAt: time.Now().UTC(), Target: e.Target, Model: e.Model, State: "pending"}
	if err := s.RecordAttempt(ctx, a); err != nil {
		t.Fatal(err)
	}
	changed := a
	changed.Target.AccountID = "b"
	if err := s.RecordAttempt(ctx, changed); err != ErrInvalid {
		t.Fatal("attempt retargeted", err)
	}
	a.State, a.StatusCode, a.ErrorCode = "failed", 429, "upstream_rate_limited"
	a.FinishedAt = time.Now().UTC()
	if err := s.RecordAttempt(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordAttempt(ctx, a); err != nil {
		t.Fatal(err)
	}
	b := a
	b.Index = 2
	b.Target.AccountID = "b"
	b.State = "pending"
	b.ErrorCode = ""
	b.StatusCode = 0
	b.FinishedAt = time.Time{}
	if err := s.RecordAttempt(ctx, b); err != nil {
		t.Fatal(err)
	}
	e.AttemptIndex = 2
	e.Target = b.Target
	e.Usage = proxy.UsageSnapshot{Input: 7, Output: 3, InputKnown: true, OutputKnown: true, Complete: true}
	if err := s.Update(ctx, e); err != nil {
		t.Fatal(err)
	}
	b.State, b.StatusCode = "success", 200
	b.FinishedAt = time.Now().UTC()
	b.Usage = e.Usage
	if err := s.RecordAttempt(ctx, b); err != nil {
		t.Fatal(err)
	}
	if err := s.Finish(ctx, e); err != nil {
		t.Fatal(err)
	}
	item, err := s.Request(ctx, e.ID)
	if err != nil || item.AttemptCount != 2 || len(item.Attempts) != 2 || item.Attempts[0].InputKnown || item.Attempts[1].InputTokens != "7" {
		t.Fatal("wrong detail", item, err)
	}
	usage, _ := s.Usage(ctx)
	if usage.AllTime.Requests != 1 || usage.AllTime.TotalTokens != "10" {
		t.Fatal("attempts duplicated totals", usage)
	}
	if _, err = s.db.Exec(`DELETE FROM request_records WHERE id=?`, e.ID); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM request_attempts WHERE request_id=?`, e.ID).Scan(&count); err != nil || count != 0 {
		t.Fatal("attempts orphaned", count, err)
	}
	usage, _ = s.Usage(ctx)
	if usage.AllTime.TotalTokens != "10" {
		t.Fatal("pruning reduced usage")
	}
}

func TestAttemptRecoveryAndHistoricalRequests(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	e := event(1, "member@example.invalid", "a")
	e.StartedAt = time.Now().Add(-time.Hour)
	if err := s.Begin(ctx, e); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordAttempt(ctx, proxy.AttemptEvent{RequestID: e.ID, Index: 1, StartedAt: e.StartedAt, Target: e.Target, Model: e.Model, State: "pending"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	item, err := s.Request(ctx, e.ID)
	if err != nil || item.State != "cancelled" || item.Attempts[0].State != "cancelled" {
		t.Fatal("unfinished attempt not recovered", item, err)
	}
	legacy := event(2, "member@example.invalid", "b")
	record(t, s, legacy)
	item, err = s.Request(ctx, legacy.ID)
	if err != nil || item.AttemptCount != 0 || len(item.Attempts) != 0 {
		t.Fatal("historical attempts invented", item, err)
	}
}
