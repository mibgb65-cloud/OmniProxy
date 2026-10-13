package routing

import (
	"context"
	"slices"
	"strings"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
)

func TestProviderOrderPersistsAndDrivesPreviewCatalogAndFallback(t *testing.T) {
	s, db := preferenceFixture(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO providers(id,kind,source,name,base_url,enabled,created_at,updated_at) SELECT 'third_ds',kind,source,'Third DeepSeek',base_url,enabled,created_at,updated_at FROM providers WHERE id='backup_ds';
INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at) VALUES('acct_third','third_ds','Third','unused',1,'2026-03-01T00:00:00Z','now');
INSERT INTO account_models(account_id,model,name,fetched_at,account_version,metadata_json) VALUES('acct_third','deepseek-flash','Third Model','now','unused:','{"contextWindow":333000}');`); err != nil {
		t.Fatal(err)
	}
	order := []string{"backup_ds", "third_ds", "builtin_deepseek"}
	if err := s.SetProviderOrder(ctx, "deepseek-flash", order); err != nil {
		t.Fatal(err)
	}
	s = New(db, provider.NewService(db), account.NewService(db, unreadStore{}))
	if saved, err := s.preferences(ctx); err != nil || !slices.Equal(saved["deepseek-flash"], order) {
		t.Fatal("order not persisted", saved, err)
	}
	if _, err := db.Exec(`UPDATE providers SET enabled=0 WHERE id='backup_ds'`); err != nil {
		t.Fatal(err)
	}
	preview, err := s.Resolve(ctx, "deepseek-flash")
	if err != nil || preview.ProviderID != "third_ds" || preview.ProviderPriority != 2 || preview.PreferenceApplied {
		t.Fatalf("second priority not used: %+v %v", preview, err)
	}
	entries, err := s.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if entry.ID == "deepseek-flash" {
			found = true
			if entry.ProviderID != "third_ds" || entry.Metadata.ContextWindow == nil || *entry.Metadata.ContextWindow != 333000 {
				t.Fatal("catalog capability did not follow second provider")
			}
		}
	}
	if !found {
		t.Fatal("catalog model missing")
	}
	items, err := s.DefaultRoutes(ctx)
	if err != nil || len(items[0].ProviderOrder) != 3 || items[0].ProviderOrder[0].Available || !items[0].ProviderOrder[1].Available {
		t.Fatal("unavailable priority lost", items, err)
	}
	if err := s.SetProviderOrder(ctx, "deepseek-flash", []string{"third_ds", "backup_ds", "builtin_deepseek"}); err != nil {
		t.Fatal("cannot retain unavailable saved priority", err)
	}
	if err := s.SetPreferredProvider(ctx, "deepseek-flash", "builtin_deepseek"); err != nil {
		t.Fatal(err)
	}
	saved, err := s.preferences(ctx)
	if err != nil || !slices.Equal(saved["deepseek-flash"], []string{"builtin_deepseek", "third_ds", "backup_ds"}) {
		t.Fatal("quick preference lost remaining priority", saved, err)
	}
	rule, err := s.Create(ctx, input("deepseek-flash"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetProviderOrder(ctx, "deepseek-flash", []string{"third_ds", "builtin_deepseek"}); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "deepseek-flash")
	if err != nil || !preview.Matched || preview.ProviderID != "builtin_deepseek" || preview.ProviderPriority != 0 {
		t.Fatal("priority escaped explicit scope", err)
	}
	if _, err := s.SetEnabled(ctx, rule.ID, false); err != nil {
		t.Fatal(err)
	}
	if err := s.SetProviderOrder(ctx, "deepseek-flash", nil); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "deepseek-flash")
	if err != nil || preview.ProviderID != "builtin_deepseek" || preview.PreferredProviderID != "" {
		t.Fatal("automatic not restored", err)
	}
}

func TestProviderOrderValidationIsAtomicAndCorruptionFailsClosed(t *testing.T) {
	s, db := preferenceFixture(t)
	ctx := context.Background()
	original := []string{"backup_ds", "builtin_deepseek"}
	if err := s.SetProviderOrder(ctx, "deepseek-flash", original); err != nil {
		t.Fatal(err)
	}
	for _, ids := range [][]string{
		{"backup_ds", "backup_ds"}, {""}, {"missing"}, {"builtin_openai"}, {"bad\nprovider"}, {strings.Repeat("x", 129)}, make([]string, maxProviderOrder+1),
	} {
		if err := s.SetProviderOrder(ctx, "deepseek-flash", ids); err == nil {
			t.Fatal("invalid order accepted")
		}
		saved, err := s.preferences(ctx)
		if err != nil || !slices.Equal(saved["deepseek-flash"], original) {
			t.Fatal("failed save altered priority", err)
		}
	}
	if err := s.SetProviderOrder(ctx, "bad model", original); err == nil {
		t.Fatal("invalid model accepted")
	}
	for _, encoded := range []string{`null`, `[]`, `{}`, `["backup_ds","backup_ds"]`, `["builtin_deepseek"]`, `["backup_ds",42]`, `broken`} {
		if _, err := db.Exec(`UPDATE model_provider_preferences SET provider_order=?`, encoded); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Resolve(ctx, "deepseek-flash"); err == nil {
			t.Fatal("corrupt priority silently ignored", encoded)
		}
	}
	if err := s.SetProviderOrder(ctx, "deepseek-flash", nil); err != nil {
		t.Fatal("corrupt priority cannot be cleared", err)
	}
}
