package account

import (
	"context"
	"errors"
	"omniproxy/internal/provider/kimi"
	"testing"
)

func TestKimiBillingChangeInvalidatesSnapshotAndPreservesProtectedCredential(t *testing.T) {
	s, store := testService(t)
	ctx := context.Background()
	a, err := s.Create(ctx, CreateInput{ProviderID: "builtin_kimi", Name: "ordinary", APIKey: "placeholder-kimi-key"})
	if err != nil {
		t.Fatal(err)
	}
	var version string
	if err := s.db.QueryRow(`SELECT version FROM account_catalog_versions WHERE id=?`, a.ID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version)VALUES(?,'kimi-k3','K3','now',?)`, a.ID, version); err != nil {
		t.Fatal(err)
	}
	o := *a.Kimi
	o.BillingMode = "coding"
	u, err := s.Update(ctx, UpdateInput{ID: a.ID, Name: a.Name, Kimi: &o})
	if err != nil || u.Account.Kimi.Mode() != "coding" || u.Account.Kimi.Version != a.Kimi.Version+1 {
		t.Fatal("billing change missing", err)
	}
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM account_models WHERE account_id=?`, a.ID).Scan(&count); err != nil || count != 0 {
		t.Fatal("API catalog reused for subscription")
	}
	if string(store.secret) != "placeholder-kimi-key" {
		t.Fatal("billing edit replaced credential")
	}
	if err := s.UseAPIKeyAt(ctx, a.ID, a.UpdatedAt, func([]byte) error { t.Fatal("stale route read key"); return nil }); !errors.Is(err, ErrConflict) {
		t.Fatal("stale route accepted")
	}
	if _, err := s.Update(ctx, UpdateInput{ID: a.ID, Name: a.Name, Kimi: a.Kimi}); !errors.Is(err, ErrConflict) {
		t.Fatal("stale billing edit accepted")
	}
	if _, err := s.Create(ctx, CreateInput{ProviderID: "builtin_kimi", APIKey: "placeholder", Kimi: &kimi.Options{Region: "cn", BillingMode: "invalid"}}); err == nil {
		t.Fatal("invalid billing accepted")
	}
	batch, err := s.CreateBatch(ctx, BatchCreateInput{ProviderID: "builtin_kimi", APIKeys: []string{"placeholder-one", "placeholder-two"}, Kimi: &kimi.Options{Region: "global", BillingMode: "coding"}})
	if err != nil || len(batch) != 2 {
		t.Fatal("subscription batch failed", err)
	}
	for _, item := range batch {
		if item.Kimi.Mode() != "coding" || item.Kimi.Region != "global" {
			t.Fatal("batch pool mixed")
		}
	}
}
