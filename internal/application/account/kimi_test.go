package account

import (
	"context"
	"errors"
	"omniproxy/internal/provider/kimi"
	"testing"
)

func TestKimiPlatformChangesInvalidateModelsAndRejectStaleEdit(t *testing.T) {
	s, store := testService(t)
	ctx := context.Background()
	a, err := s.Create(ctx, CreateInput{ProviderID: "builtin_kimi", Name: "Kimi", APIKey: "placeholder-kimi-key"})
	if err != nil || a.Kimi == nil || a.Kimi.Region != "cn" || a.Kimi.Version != 1 {
		t.Fatalf("create=%+v %v", a, err)
	}
	var version string
	if err := s.db.QueryRow(`SELECT version FROM account_catalog_versions WHERE id=?`, a.ID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version)VALUES(?,'kimi-k3','Kimi','now',?)`, a.ID, version); err != nil {
		t.Fatal(err)
	}
	options := *a.Kimi
	options.Region = "global"
	updated, err := s.Update(ctx, UpdateInput{ID: a.ID, Name: a.Name, Kimi: &options})
	if err != nil || updated.Account.Kimi.Version != 2 {
		t.Fatalf("update=%+v %v", updated, err)
	}
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM account_models WHERE account_id=?`, a.ID).Scan(&count); err != nil || count != 0 {
		t.Fatal("old platform cache survived")
	}
	if _, err := s.Update(ctx, UpdateInput{ID: a.ID, Name: "stale", Kimi: a.Kimi}); !errors.Is(err, ErrConflict) {
		t.Fatal("stale platform edit accepted")
	}
	if string(store.secret) != "placeholder-kimi-key" {
		t.Fatal("platform edit replaced key")
	}
	if err := s.UseAPIKeyAt(ctx, a.ID, a.UpdatedAt, func([]byte) error { t.Fatal("stale platform read credential"); return nil }); !errors.Is(err, ErrConflict) {
		t.Fatal("stale account snapshot accepted")
	}
}

func TestKimiOptionsValidateAndBatchRollsBackProtectedKeys(t *testing.T) {
	s, _ := testService(t)
	store := &batchStore{values: map[string][]byte{}}
	s.store = store
	ctx := context.Background()
	for _, input := range []CreateInput{{ProviderID: "builtin_kimi", APIKey: "placeholder", Kimi: &kimi.Options{Region: "other"}}, {ProviderID: "builtin_deepseek", APIKey: "placeholder", Kimi: &kimi.Options{Region: "cn"}}, {ProviderID: "builtin_kimi", APIKey: "placeholder key", Kimi: &kimi.Options{Region: "cn"}}} {
		if _, err := s.Create(ctx, input); err == nil {
			t.Fatal("invalid platform or key accepted")
		}
	}
	if store.puts != 0 {
		t.Fatal("invalid options reached credential store")
	}
	if _, err := s.db.Exec(`CREATE TRIGGER fail_second_kimi BEFORE INSERT ON kimi_account_options WHEN (SELECT count(*) FROM kimi_account_options)>0 BEGIN SELECT RAISE(ABORT,'test platform error'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateBatch(ctx, BatchCreateInput{ProviderID: "builtin_kimi", APIKeys: []string{"placeholder-kimi-one", "placeholder-kimi-two"}, Kimi: &kimi.Options{Region: "global"}}); err == nil || len(store.values) != 0 {
		t.Fatal("failed batch retained protected keys")
	}
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM accounts`).Scan(&count); err != nil || count != 0 {
		t.Fatal("failed platform batch retained accounts")
	}
}
