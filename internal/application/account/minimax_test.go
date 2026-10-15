package account

import (
	"context"
	"errors"
	"omniproxy/internal/provider/minimax"
	"testing"
)

func TestMiniMaxOptionsLifecycleAndVersionIsolation(t *testing.T) {
	s, store := testService(t)
	ctx := context.Background()
	o := &minimax.Options{Region: "global", BillingMode: "subscription"}
	a, err := s.Create(ctx, CreateInput{ProviderID: "builtin_minimax", Name: "MiniMax", APIKey: "placeholder-step-key", MiniMax: o})
	if err != nil || a.MiniMax == nil || a.MiniMax.Version != 1 {
		t.Fatalf("create: %+v %v", a, err)
	}
	var version string
	if err := s.db.QueryRow(`SELECT version FROM account_catalog_versions WHERE id=?`, a.ID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version)VALUES(?,'MiniMax-M3','MiniMax','now',?)`, a.ID, version); err != nil {
		t.Fatal(err)
	}
	next := *a.MiniMax
	next.BillingMode = "payg"
	updated, err := s.Update(ctx, UpdateInput{ID: a.ID, Name: "MiniMax renamed", MiniMax: &next})
	if err != nil || updated.Account.MiniMax.Version != 2 {
		t.Fatalf("update: %+v %v", updated, err)
	}
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM account_models WHERE account_id=?`, a.ID).Scan(&count); err != nil || count != 0 {
		t.Fatal("old model cache survived channel switch")
	}
	if _, err := s.Update(ctx, UpdateInput{ID: a.ID, Name: "stale", MiniMax: a.MiniMax}); !errors.Is(err, ErrConflict) {
		t.Fatal("stale options overwrote new billing")
	}
	if err := s.UseAPIKeyAt(ctx, a.ID, a.UpdatedAt, func([]byte) error { t.Fatal("stale target read key"); return nil }); !errors.Is(err, ErrConflict) {
		t.Fatal("stale gateway snapshot accepted")
	}
	if string(store.secret) != "placeholder-step-key" {
		t.Fatal("option change replaced credential")
	}
	unchanged, err := s.Update(ctx, UpdateInput{ID: a.ID, Name: "only name"})
	if err != nil || unchanged.Account.MiniMax.Version != 2 {
		t.Fatal("name change changed channel identity")
	}
}

func TestMiniMaxOptionsValidationAndBatchRollback(t *testing.T) {
	s, _ := testService(t)
	store := &batchStore{values: make(map[string][]byte)}
	s.store = store
	ctx := context.Background()
	for _, input := range []CreateInput{
		{ProviderID: "builtin_minimax", APIKey: "sk-api-placeholder", MiniMax: &minimax.Options{Region: "cn", BillingMode: "subscription"}},
		{ProviderID: "builtin_minimax", APIKey: "sk-cp-placeholder", MiniMax: &minimax.Options{Region: "cn", BillingMode: "payg"}},
		{ProviderID: "builtin_minimax", APIKey: "placeholder-step-key", MiniMax: &minimax.Options{Region: "other", BillingMode: "payg"}},
		{ProviderID: "builtin_openai", APIKey: "placeholder-step-key", MiniMax: &minimax.Options{Region: "cn", BillingMode: "payg"}},
		{ProviderID: "builtin_minimax", APIKey: "placeholder key", MiniMax: &minimax.Options{Region: "cn", BillingMode: "payg"}},
	} {
		if _, err := s.Create(ctx, input); err == nil {
			t.Fatal("invalid MiniMax configuration accepted")
		}
	}
	if store.puts != 0 {
		t.Fatal("invalid configuration reached protected storage")
	}
	if _, err := s.db.Exec(`CREATE TRIGGER reject_second_step BEFORE INSERT ON minimax_account_options WHEN (SELECT COUNT(*) FROM minimax_account_options)>0 BEGIN SELECT RAISE(ABORT,'test options failure'); END`); err != nil {
		t.Fatal(err)
	}
	_, err := s.CreateBatch(ctx, BatchCreateInput{ProviderID: "builtin_minimax", APIKeys: []string{"placeholder-key-1", "placeholder-key-2"}, MiniMax: &minimax.Options{Region: "cn", BillingMode: "subscription"}})
	if err == nil || len(store.values) != 0 {
		t.Fatal("failed option batch retained credentials")
	}
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM accounts`).Scan(&count); err != nil || count != 0 {
		t.Fatal("failed options batch retained accounts")
	}
}
