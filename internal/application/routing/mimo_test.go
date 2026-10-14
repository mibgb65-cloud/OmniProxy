package routing

import (
	"context"
	"testing"

	"omniproxy/internal/provider/mimo"
)

func TestMiMoRoutingCatalogAliasesAndAccountEligibility(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at) VALUES('mimo-account','builtin_mimo','MiMo','protected-ref',1,'2026-10-14T00:00:00Z','2026-10-14T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	for _, model := range mimo.Models() {
		preview, err := s.Resolve(ctx, model)
		if err != nil || !preview.Routable || preview.ProviderID != "builtin_mimo" || preview.AccountID != "mimo-account" {
			t.Fatal("preset not routed", preview, err)
		}
	}
	if _, err := s.Create(ctx, Input{Alias: "mimo-coding", ProviderID: "builtin_mimo", AccountID: "mimo-account", Model: "mimo-v2.6-pro", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	preview, err := s.Resolve(ctx, "mimo-coding")
	if err != nil || !preview.Matched || !preview.Routable || preview.Model != "mimo-v2.6-pro" {
		t.Fatal("alias failed", preview, err)
	}
	entries, err := s.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, entry := range entries {
		if entry.ProviderKind != mimo.Kind {
			continue
		}
		found++
		if len(entry.Protocols) != 3 || entry.Metadata.ContextWindow == nil || *entry.Metadata.ContextWindow != 1000000 || entry.Metadata.ContextVerified {
			t.Fatal("native protocols or metadata lost")
		}
	}
	if found != len(mimo.Models())+1 {
		t.Fatal("models or alias missing", found)
	}
	for _, change := range []string{`UPDATE accounts SET enabled=0 WHERE id='mimo-account'`, `UPDATE accounts SET enabled=1,auth_method='subscription_login' WHERE id='mimo-account'`, `UPDATE providers SET enabled=0 WHERE id='builtin_mimo'`} {
		if _, err := db.Exec(change); err != nil {
			t.Fatal(err)
		}
		preview, err = s.Resolve(ctx, "mimo-v2.6-pro")
		if err != nil || preview.Routable {
			t.Fatal("ineligible MiMo account routed", err)
		}
	}
	if _, err := s.Create(ctx, Input{Alias: "speech", ProviderID: "builtin_mimo", Model: "mimo-v2.5-asr", Enabled: true}); err == nil {
		t.Fatal("speech model accepted in dialogue route")
	}
}

func TestMiMoDiscoveredAccountRestrictsPresetsAndDefaults(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at) VALUES('mimo-account','builtin_mimo','MiMo','protected-ref',1,'2026-10-14T00:00:00Z','2026-10-14T00:00:00Z');
		INSERT INTO account_models(account_id,model,name,fetched_at,account_version,metadata_json) VALUES('mimo-account','mimo-v2.6-flash','Flash','now','protected-ref:','{}')`); err != nil {
		t.Fatal(err)
	}
	preview, err := s.Resolve(ctx, "mimo-v2.6-pro")
	if err != nil || preview.Routable {
		t.Fatal("preset overrode account discovery")
	}
	preview, err = s.Resolve(ctx, "mimo-v2.6-flash")
	if err != nil || !preview.Routable {
		t.Fatal("discovered model not routed")
	}
	defaults, err := s.DefaultRoutes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range defaults {
		if item.Model == "mimo-v2.6-pro" {
			t.Fatal("removed model returned as preset")
		}
	}
	if _, err := db.Exec(`UPDATE accounts SET credential_ref='rotated-ref' WHERE id='mimo-account'`); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "mimo-v2.6-pro")
	if err != nil || !preview.Routable {
		t.Fatal("rotated account cannot use explicit preset")
	}
}
