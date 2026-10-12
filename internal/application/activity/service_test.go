package activity

import (
	"context"
	"fmt"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func fixture(t *testing.T) *Service {
	t.Helper()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "activity.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err = database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	return New(db)
}
func event(id int, email, account string) proxy.RequestEvent {
	return proxy.RequestEvent{ID: fmt.Sprintf("%032x", id), StartedAt: time.Now().UTC().Add(time.Duration(id) * time.Second), FinishedAt: time.Now().UTC(), Protocol: "responses", Transport: "http", Model: "gpt-test", InputModel: "alias", Target: proxy.TargetSnapshot{ProviderID: "builtin_openai", ProviderKind: "openai", ProviderName: "OpenAI", AccountID: account, AccountName: "Member", AuthMethod: "subscription_login", Email: email, WorkspaceID: "shared-team"}, State: "success", StatusCode: 200}
}
func record(t *testing.T, s *Service, e proxy.RequestEvent) {
	t.Helper()
	ctx := context.Background()
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
func TestTeamMembersDeletionAndRecreation(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	s.db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at)VALUES('a','builtin_openai','A','unused',1,'now','now')`)
	a := event(1, "a@example.invalid", "a")
	a.Usage = proxy.UsageSnapshot{Input: 100, Output: 50, InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, a)
	if err := s.Update(ctx, a); err != nil {
		t.Fatal(err)
	} // 完成后的重复快照不能再次累计。
	b := event(2, "b@example.invalid", "b")
	b.Usage = proxy.UsageSnapshot{Input: 200, Output: 25, InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, b)
	info, err := s.Usage(ctx)
	if err != nil || info.AllTime.TotalTokens != "375" || len(info.Accounts) != 2 {
		t.Fatal("Team members merged", info, err)
	}
	if _, err = s.db.Exec(`DELETE FROM accounts WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	original, err := s.Request(ctx, a.ID)
	if err != nil || !original.AccountDeleted || original.InputTokens != "100" {
		t.Fatal("deletion lost snapshot", err)
	}
	info, err = s.Usage(ctx)
	if err != nil || info.AllTime.TotalTokens != "375" {
		t.Fatal("deletion changed totals")
	}
	again := event(3, "A@EXAMPLE.INVALID", "new-local-id")
	again.Usage = proxy.UsageSnapshot{Input: 7, Output: 3, InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, again)
	info, _ = s.Usage(ctx)
	if info.AllTime.TotalTokens != "385" || len(info.Accounts) != 2 {
		t.Fatal("same identity recreation split history")
	}
}
func TestConcurrentCumulativeSnapshotsAndUnknownZero(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	e := event(1, "a@example.invalid", "a")
	if err := s.Begin(ctx, e); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 1; i <= 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			snapshot := e
			snapshot.Usage = proxy.UsageSnapshot{Input: int64(n), Output: int64(n * 2), InputKnown: true, OutputKnown: true, Complete: true}
			if err := s.Update(ctx, snapshot); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	if err := s.Finish(ctx, e); err != nil {
		t.Fatal(err)
	}
	unknown := event(2, "a@example.invalid", "a")
	record(t, s, unknown)
	zero := event(3, "a@example.invalid", "a")
	zero.Usage = proxy.UsageSnapshot{InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, zero)
	info, _ := s.Usage(ctx)
	if info.AllTime.TotalTokens != "60" || info.AllTime.Requests != 3 || info.AllTime.CompleteUsage != 2 {
		t.Fatal("double counting", info.AllTime)
	}
	item, _ := s.Request(ctx, unknown.ID)
	if item.InputKnown || item.OutputKnown {
		t.Fatal("unknown changed to zero")
	}
}
func TestRetentionKeepsLifetimeAndDailyTotals(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	defaults, err := s.Retention(ctx)
	if err != nil || defaults.MaxRecords != 10000 || defaults.LogCapacity != 5000 {
		t.Fatal("wrong defaults")
	}
	if _, err = s.SetRetention(ctx, 99); err == nil {
		t.Fatal("invalid setting accepted")
	}
	for i := 1; i <= 105; i++ {
		e := event(i, "a@example.invalid", "a")
		e.Usage = proxy.UsageSnapshot{Input: 1, Output: 1, InputKnown: true, OutputKnown: true, Complete: true}
		record(t, s, e)
	}
	retained, err := s.SetRetention(ctx, 100)
	if err != nil || retained.StoredRecords != 100 {
		t.Fatal("not evicted", err)
	}
	info, err := s.Usage(ctx)
	if err != nil || info.AllTime.TotalTokens != "210" || info.AllTime.Requests != 105 || len(info.Days) != 1 || info.Days[0].Summary.TotalTokens != "210" {
		t.Fatal("eviction changed totals", err)
	}
	if _, err = s.Request(ctx, event(1, "a@example.invalid", "a").ID); err == nil {
		t.Fatal("oldest detail retained")
	}
	page, err := s.Requests(ctx, RequestQuery{Search: "' OR 1=1 --", Limit: 50})
	if err != nil || page.Total != 0 {
		t.Fatal("search injection")
	}
}

func TestRecoveryAndActiveRequestsRetention(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	if _, err := s.SetRetention(ctx, 100); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 100; i++ {
		record(t, s, event(i, "a@example.invalid", "a"))
	}
	pending := event(200, "a@example.invalid", "a")
	if err := s.Begin(ctx, pending); err != nil {
		t.Fatal(err)
	}
	settings, _ := s.Retention(ctx)
	if settings.StoredRecords != 100 {
		t.Fatal("active requests exceeded retention limit")
	}
	old := event(201, "a@example.invalid", "a")
	old.StartedAt = time.Now().Add(-time.Hour)
	if err := s.Begin(ctx, old); err != nil {
		t.Fatal(err)
	}
	if err := s.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	recovered, err := s.Request(ctx, old.ID)
	if err != nil || recovered.State != "cancelled" || recovered.ErrorCode != "process_interrupted" {
		t.Fatal("stale request not recovered", err)
	}
	current, _ := s.Request(ctx, pending.ID)
	if current.State != "pending" {
		t.Fatal("active request overwritten")
	}
}
