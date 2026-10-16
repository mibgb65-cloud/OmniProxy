package routing

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

func mimoOrderFixture(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	s, db := fixture(t)
	_, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at) VALUES
 ('mimo-payg','builtin_mimo','API','ref-payg',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z'),
 ('mimo-plan','builtin_mimo','Plan','ref-plan',1,'2026-01-02T00:00:00Z','2026-01-02T00:00:00Z'),
 ('mimo-plan-new','builtin_mimo','Plan SGP','ref-sgp',1,'2026-01-03T00:00:00Z','2026-01-03T00:00:00Z');
 INSERT INTO mimo_account_options(account_id,region,billing_mode) VALUES
 ('mimo-payg','','payg'),('mimo-plan','cn','token_plan'),('mimo-plan-new','sgp','token_plan')`)
	if err != nil {
		t.Fatal(err)
	}
	return s, db
}

func TestMiMoBillingOrderSelectsCapabilitiesAndKeepsExplicitRoutes(t *testing.T) {
	s, db := mimoOrderFixture(t)
	ctx := context.Background()
	assertAccount := func(model, want string) {
		t.Helper()
		preview, err := s.Resolve(ctx, model)
		if err != nil || !preview.Routable || preview.AccountID != want {
			t.Fatalf("%s: %+v %v", model, preview, err)
		}
	}
	assertAccount("mimo-v2.6-pro", "mimo-plan")
	assertAccount("mimo-v2.6-pro-ultraspeed", "mimo-payg")
	if _, err := s.SetMiMoPreferredBillingMode(ctx, "payg"); err != nil {
		t.Fatal(err)
	}
	assertAccount("mimo-v2.6-pro", "mimo-payg")
	assertAccount("mimo-cn-plan/mimo-v2.6-pro", "mimo-plan")
	assertAccount("mimo-sgp-plan/mimo-v2.6-pro", "mimo-plan-new")
	if _, err := s.Create(ctx, Input{Alias: "fixed-plan", ProviderID: "builtin_mimo", AccountID: "mimo-plan", Model: "mimo-v2.6-pro", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	assertAccount("fixed-plan", "mimo-plan")
	if _, err := db.Exec(`UPDATE accounts SET enabled=0 WHERE id='mimo-payg'`); err != nil {
		t.Fatal(err)
	}
	assertAccount("mimo-v2.6-pro", "mimo-plan")
	if _, err := s.SetMiMoPreferredBillingMode(ctx, "token_plan"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE accounts SET enabled=1 WHERE id='mimo-payg';
 INSERT INTO account_models(account_id,model,name,fetched_at,account_version,metadata_json) SELECT a.id,'mimo-v2.6-flash','Flash','now',v.version,'{}' FROM accounts a JOIN account_catalog_versions v ON a.id=v.id WHERE a.id IN ('mimo-plan','mimo-plan-new')`); err != nil {
		t.Fatal(err)
	}
	assertAccount("mimo-v2.6-pro", "mimo-payg")
	assertAccount("mimo-v2.6-flash", "mimo-plan")
	if _, err := db.Exec(`UPDATE accounts SET enabled=0 WHERE id='mimo-plan'`); err != nil {
		t.Fatal(err)
	}
	assertAccount("mimo-v2.6-flash", "mimo-plan-new")
	preview, err := s.Resolve(ctx, "fixed-plan")
	if err != nil || preview.Routable {
		t.Fatal("fixed disabled account fell back", preview, err)
	}
}

func TestMiMoDirectoryDeduplicatesAndDefaultPreviewAgrees(t *testing.T) {
	s, _ := mimoOrderFixture(t)
	ctx := context.Background()
	if _, err := s.Create(ctx, Input{Alias: "manual-api", ProviderID: "builtin_mimo", AccountID: "mimo-payg", Model: "mimo-v2.6-pro", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"token_plan", "payg"} {
		if _, err := s.SetMiMoPreferredBillingMode(ctx, mode); err != nil {
			t.Fatal(err)
		}
		entries, err := s.Catalog(ctx)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for _, entry := range entries {
			if entry.ProviderKind != "mimo" {
				continue
			}
			if seen[entry.ID] || strings.Contains(entry.ID, "/") {
				t.Fatal("generated duplicate channel model", entry.ID)
			}
			seen[entry.ID] = true
			if entry.Metadata.ContextWindow == nil || *entry.Metadata.ContextWindow != 1000000 || len(entry.Protocols) != 3 {
				t.Fatal("model metadata lost", entry.ID)
			}
		}
		if len(seen) != 4 || !seen["manual-api"] {
			t.Fatal("plain models or explicit alias missing", seen)
		}
		defaults, err := s.DefaultRoutes(ctx)
		if err != nil {
			t.Fatal(err)
		}
		want := "mimo-plan"
		if mode == "payg" {
			want = "mimo-payg"
		}
		found := false
		for _, item := range defaults {
			if item.Model == "mimo-v2.6-pro" {
				found = true
				if item.Preview.AccountID != want || len(item.Providers) != 1 {
					t.Fatal("default preview disagrees", item)
				}
			}
		}
		if !found {
			t.Fatal("default model missing")
		}
	}
}
