package routing

import (
	"context"
	"testing"

	"omniproxy/internal/provider/atria"
)

func TestAtriaPresetRoutingAliasesAndEligibility(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at)
		VALUES('atria-account','builtin_atria','Atria account','protected-ref',1,'2026-10-13T00:00:00Z','2026-10-13T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	preview, err := s.Resolve(ctx, atria.ModelID)
	if err != nil || !preview.Routable || preview.ProviderID != "builtin_atria" || preview.AccountID != "atria-account" {
		t.Fatal("preset routing failed", preview, err)
	}
	if _, err := s.Create(ctx, Input{Alias: "research", ProviderID: "builtin_atria", Model: atria.ModelID, AccountID: "atria-account", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "research")
	if err != nil || !preview.Routable || !preview.Matched || preview.Model != atria.ModelID {
		t.Fatal("alias failed", preview, err)
	}
	entries, err := s.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, entry := range entries {
		if entry.ProviderKind == atria.Kind {
			found++
			if entry.Metadata.ContextWindow == nil || *entry.Metadata.ContextWindow != 256000 || len(entry.Metadata.InputModalities) != 1 || entry.Metadata.InputModalities[0] != "text" || len(entry.Protocols) != 3 {
				t.Fatal("preset metadata lost", entry)
			}
		}
	}
	if found != 2 {
		t.Fatal("model and alias missing")
	}
	defaults, err := s.DefaultRoutes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found = 0
	for _, item := range defaults {
		if item.Model == atria.ModelID && item.Preview.Routable {
			found++
		}
	}
	if found != 1 {
		t.Fatal("default mapping missing")
	}
	for _, change := range []string{`UPDATE accounts SET enabled=0 WHERE id='atria-account'`, `UPDATE accounts SET enabled=1,auth_method='subscription_login' WHERE id='atria-account'`, `UPDATE providers SET enabled=0 WHERE id='builtin_atria'`} {
		if _, err := db.Exec(change); err != nil {
			t.Fatal(err)
		}
		preview, err = s.Resolve(ctx, atria.ModelID)
		if err != nil || preview.Routable {
			t.Fatal("ineligible account routed", preview, err)
		}
	}
	if _, err := s.Create(ctx, Input{Alias: "wrong", ProviderID: "builtin_deepseek", Model: atria.ModelID, Enabled: true}); err == nil {
		t.Fatal("model routed to wrong provider")
	}
}
