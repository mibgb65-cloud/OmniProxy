package account

import (
	"context"
	"errors"
	"testing"

	"omniproxy/internal/provider/mimo"
)

func TestMiMoChannelVersionsAndCredentialBoundaries(t *testing.T) {
	s, store := testService(t)
	ctx := context.Background()
	a, err := s.Create(ctx, CreateInput{ProviderID: "builtin_mimo", Name: "MiMo", APIKey: "tp-placeholder-key", MiMo: &mimo.Options{Region: "cn", BillingMode: "token_plan"}})
	if err != nil || a.MiMo == nil || a.MiMo.Version != 1 {
		t.Fatalf("create=%+v %v", a, err)
	}
	var version string
	if err := s.db.QueryRow(`SELECT version FROM account_catalog_versions WHERE id=?`, a.ID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version)VALUES(?,'mimo-v2.6-pro','Pro','now',?)`, a.ID, version); err != nil {
		t.Fatal(err)
	}
	o := *a.MiMo
	o.Region = "sgp"
	updated, err := s.Update(ctx, UpdateInput{ID: a.ID, Name: a.Name, MiMo: &o})
	if err != nil || updated.Account.MiMo.Region != "sgp" || updated.Account.MiMo.Version != 2 || string(store.secret) != "tp-placeholder-key" {
		t.Fatalf("update=%+v %v", updated, err)
	}
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM account_models WHERE account_id=?`, a.ID).Scan(&count); err != nil || count != 0 {
		t.Fatal("old cluster catalog retained")
	}
	if _, err := s.Update(ctx, UpdateInput{ID: a.ID, Name: a.Name, MiMo: a.MiMo}); !errors.Is(err, ErrConflict) {
		t.Fatal("stale edit accepted")
	}
	if err := s.UseAPIKeyAt(ctx, a.ID, a.UpdatedAt, func([]byte) error { t.Fatal("old snapshot read key"); return nil }); !errors.Is(err, ErrConflict) {
		t.Fatal("old snapshot accepted")
	}
	p := mimo.Options{BillingMode: "payg", Version: 2}
	if _, err := s.Update(ctx, UpdateInput{ID: a.ID, Name: a.Name, MiMo: &p}); err == nil {
		t.Fatal("plan key kept when changing billing channel")
	}
	// 单纯重命名保留通道版本，避免无意义失效。
	renamed, err := s.Update(ctx, UpdateInput{ID: a.ID, Name: "Renamed"})
	if err != nil || renamed.Account.MiMo.Version != 2 {
		t.Fatal("rename changed channel")
	}
	items, err := s.List(ctx)
	if err != nil || len(items) != 1 || items[0].MiMo.Region != "sgp" {
		t.Fatal("list lost channel metadata")
	}
}

func TestMiMoOptionsAndBatchAreValidatedBeforeProtectingKeys(t *testing.T) {
	s, _ := testService(t)
	ctx := context.Background()
	store := &batchStore{values: map[string][]byte{}}
	s.store = store
	for _, input := range []CreateInput{
		{ProviderID: "builtin_mimo", APIKey: "tp-placeholder-key"},
		{ProviderID: "builtin_mimo", APIKey: "sk-placeholder-key", MiMo: &mimo.Options{Region: "ams", BillingMode: "token_plan"}},
		{ProviderID: "builtin_mimo", APIKey: "tp-placeholder-key", MiMo: &mimo.Options{Region: "other", BillingMode: "token_plan"}},
		{ProviderID: "builtin_deepseek", APIKey: "placeholder-key", MiMo: &mimo.Options{BillingMode: "payg"}},
	} {
		if _, err := s.Create(ctx, input); err == nil {
			t.Fatal("invalid channel accepted")
		}
	}
	if store.puts != 0 {
		t.Fatal("invalid request stored credentials")
	}
	if _, err := s.CreateBatch(ctx, BatchCreateInput{ProviderID: "builtin_mimo", MiMo: &mimo.Options{Region: "sgp", BillingMode: "token_plan"}, APIKeys: []string{"tp-placeholder-key", "sk-placeholder-wrong"}}); err == nil || store.puts != 0 {
		t.Fatal("mixed batch stored credentials")
	}
	if _, err := s.db.Exec(`CREATE TRIGGER fail_second_mimo BEFORE INSERT ON mimo_account_options WHEN (SELECT count(*) FROM mimo_account_options)>0 BEGIN SELECT RAISE(ABORT,'test channel error'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateBatch(ctx, BatchCreateInput{ProviderID: "builtin_mimo", MiMo: &mimo.Options{Region: "ams", BillingMode: "token_plan"}, APIKeys: []string{"tp-placeholder-one", "ttp-placeholder-two"}}); err == nil || len(store.values) != 0 {
		t.Fatal("failed batch retained credentials")
	}
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM accounts`).Scan(&count); err != nil || count != 0 {
		t.Fatal("failed batch retained accounts")
	}
}
