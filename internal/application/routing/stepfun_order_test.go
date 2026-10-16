package routing

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

func stepOrderFixture(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	s, db := fixture(t)
	_, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at) VALUES
 ('step-api','builtin_stepfun','API','ref-api',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z'),
 ('step-plan','builtin_stepfun','Plan CN','ref-plan',1,'2026-01-02T00:00:00Z','2026-01-02T00:00:00Z'),
 ('step-plan-global','builtin_stepfun','Plan Global','ref-global',1,'2026-01-03T00:00:00Z','2026-01-03T00:00:00Z');
 INSERT INTO stepfun_account_options(account_id,region,billing_mode) VALUES
 ('step-api','global','payg'),('step-plan','cn','step_plan'),('step-plan-global','global','step_plan')`)
	if err != nil {
		t.Fatal(err)
	}
	return s, db
}

func TestStepFunDefaultOrderCapabilitiesAndExplicitRoutes(t *testing.T) {
	s, db := stepOrderFixture(t)
	ctx := context.Background()
	assertAccount := func(model, want string) {
		t.Helper()
		p, err := s.Resolve(ctx, model)
		if err != nil || !p.Routable || p.AccountID != want {
			t.Fatalf("%s: %+v %v", model, p, err)
		}
	}
	assertAccount("step-5-preview", "step-plan")
	if _, err := s.SetStepFunPreferredBillingMode(ctx, "payg"); err != nil {
		t.Fatal(err)
	}
	assertAccount("step-5-preview", "step-api")
	assertAccount("step-router-v1", "step-plan")
	assertAccount("stepfun-cn-plan/step-5-preview", "step-plan")
	assertAccount("stepfun-global-plan/step-5-preview", "step-plan-global")
	if _, err := s.Create(ctx, Input{Alias: "fixed-plan", ProviderID: "builtin_stepfun", AccountID: "step-plan", Model: "step-5-preview", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	assertAccount("fixed-plan", "step-plan")
	if _, err := db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version,metadata_json) SELECT a.id,'step-3.7-flash','Flash','now',v.version,'{}' FROM accounts a JOIN account_catalog_versions v ON a.id=v.id WHERE a.id='step-api'`); err != nil {
		t.Fatal(err)
	}
	assertAccount("step-5-preview", "step-plan")
	assertAccount("step-3.7-flash", "step-api")
	if _, err := db.Exec(`UPDATE accounts SET enabled=0 WHERE id='step-api'`); err != nil {
		t.Fatal(err)
	}
	assertAccount("step-3.7-flash", "step-plan")
	if _, err := db.Exec(`UPDATE accounts SET enabled=0 WHERE id='step-plan'`); err != nil {
		t.Fatal(err)
	}
	assertAccount("step-5-preview", "step-plan-global")
	p, err := s.Resolve(ctx, "fixed-plan")
	if err != nil || p.Routable {
		t.Fatal("fixed disabled account fell back", p, err)
	}
	p, err = s.Resolve(ctx, "step-router-v1")
	if err != nil || p.Routable {
		t.Fatal("global plan advertised CN-only router", p, err)
	}
}

func TestStepFunCatalogDeduplicatesKeepsMetadataAndManualAliases(t *testing.T) {
	s, _ := stepOrderFixture(t)
	ctx := context.Background()
	if _, err := s.Create(ctx, Input{Alias: "manual-api", ProviderID: "builtin_stepfun", AccountID: "step-api", Model: "step-5-preview", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"step_plan", "payg"} {
		if _, err := s.SetStepFunPreferredBillingMode(ctx, mode); err != nil {
			t.Fatal(err)
		}
		entries, err := s.Catalog(ctx)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for _, entry := range entries {
			if entry.ProviderKind != "stepfun" {
				continue
			}
			if seen[entry.ID] || strings.Contains(entry.ID, "/") {
				t.Fatal("duplicate channel model", entry.ID)
			}
			seen[entry.ID] = true
			if entry.TargetModel == "step-5-preview" && (entry.Metadata.ContextWindow == nil || *entry.Metadata.ContextWindow != 1000000 || len(entry.Protocols) != 3) {
				t.Fatal("metadata lost", entry.ID)
			}
			if entry.TargetModel == "step-router-v1" && len(entry.Protocols) != 2 {
				t.Fatal("router invents Responses support")
			}
		}
		if len(seen) != 6 || !seen["manual-api"] || !seen["step-router-v1"] {
			t.Fatal("supported models missing", seen)
		}
		defaults, err := s.DefaultRoutes(ctx)
		if err != nil {
			t.Fatal(err)
		}
		want := "step-plan"
		if mode == "payg" {
			want = "step-api"
		}
		found := false
		for _, item := range defaults {
			if item.Model == "step-5-preview" {
				found = true
				if item.Preview.AccountID != want || len(item.Providers) != 1 {
					t.Fatal("preview disagrees", item)
				}
			}
		}
		if !found {
			t.Fatal("default model missing")
		}
	}
}
