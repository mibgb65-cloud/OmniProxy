package routing

import (
	"context"
	"testing"

	"omniproxy/internal/provider/river"
)

func TestRiverPresetRoutingAliasesAndEligibility(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at)
		VALUES('river-account','builtin_river','River account','protected-ref',1,'2026-10-13T00:00:00Z','2026-10-13T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	preview, err := s.Resolve(ctx, river.DeepSeekFlash)
	if err != nil || !preview.Routable || preview.ProviderID != "builtin_river" || preview.AccountID != "river-account" {
		t.Fatal("preset routing failed", preview, err)
	}
	if _, err := s.Create(ctx, Input{Alias: "research", ProviderID: "builtin_river", Model: river.DeepSeekFlash, AccountID: "river-account", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "research")
	if err != nil || !preview.Routable || !preview.Matched || preview.Model != river.DeepSeekFlash {
		t.Fatal("alias failed", preview, err)
	}
	if err := s.SetProviderOrder(ctx, "deepseek-flash", []string{"builtin_river", "builtin_deepseek"}); err != nil {
		t.Fatal(err)
	}
	entries, err := s.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, entry := range entries {
		if entry.ProviderKind == river.Kind {
			found++
			if entry.Metadata.ContextWindow == nil || *entry.Metadata.ContextWindow != 1048576 ||
				len(entry.Metadata.InputModalities) != 2 || entry.Metadata.InputModalities[0] != "text" || entry.Metadata.InputModalities[1] != "image" ||
				len(entry.Protocols) != 3 {
				t.Fatal("preset metadata lost", entry)
			}
		}
	}
	if found != len(river.Models())+1 {
		t.Fatal("model and alias missing")
	}
	defaults, err := s.DefaultRoutes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found = 0
	for _, item := range defaults {
		if item.Model == "deepseek-flash" && item.Preview.Routable {
			found++
		}
	}
	if found != 1 {
		t.Fatal("default mapping missing")
	}
	for _, change := range []string{`UPDATE accounts SET enabled=0 WHERE id='river-account'`, `UPDATE accounts SET enabled=1,auth_method='subscription_login' WHERE id='river-account'`, `UPDATE providers SET enabled=0 WHERE id='builtin_river'`} {
		if _, err := db.Exec(change); err != nil {
			t.Fatal(err)
		}
		preview, err = s.Resolve(ctx, river.DeepSeekFlash)
		if err != nil || preview.Routable {
			t.Fatal("ineligible account routed", preview, err)
		}
	}
	if _, err := s.Create(ctx, Input{Alias: "wrong", ProviderID: "builtin_deepseek", Model: river.DeepSeekFlash, Enabled: true}); err == nil {
		t.Fatal("model routed to wrong provider")
	}
}
