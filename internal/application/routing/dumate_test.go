package routing

import (
	"context"
	"testing"

	"omniproxy/internal/provider/dumate"
)

// DuMate 没有 /models：只有存在已登录账号时才发布两条固定别名。
func TestDuMatePresetModelsAppearOnlyWithAuthenticatedAccount(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	preview, err := s.Resolve(ctx, dumate.ModelText)
	if err != nil || preview.Routable {
		t.Fatalf("model routed without an account: %+v %v", preview, err)
	}
	if entries, err := s.Catalog(ctx); err != nil {
		t.Fatal(err)
	} else {
		for _, entry := range entries {
			if entry.ProviderKind == dumate.Kind {
				t.Fatalf("catalog published DuMate without an account: %+v", entry)
			}
		}
	}
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method,auth_state,auth_account_id)
		VALUES('dumate-account','builtin_dumate','DuMate','protected-ref',1,'2026-11-13T00:00:00Z','2026-11-13T00:00:00Z','dumate_cookie','authenticated','1001')`); err != nil {
		t.Fatal(err)
	}
	for _, model := range dumate.Models() {
		preview, err := s.Resolve(ctx, model)
		if err != nil || !preview.Routable || preview.ProviderID != "builtin_dumate" || preview.AccountID != "dumate-account" {
			t.Fatalf("preset routing failed for %s: %+v %v", model, preview, err)
		}
	}
	entries, err := s.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, entry := range entries {
		if entry.ProviderKind != dumate.Kind {
			continue
		}
		found[entry.ID] = true
		if entry.Metadata.ContextWindow != nil || len(entry.Metadata.InputModalities) != 2 ||
			entry.Metadata.InputModalities[0] != "text" || entry.Metadata.InputModalities[1] != "image" {
			t.Fatalf("unverified capacity or modalities published: %+v", entry.Metadata)
		}
		if len(entry.Protocols) != 3 {
			t.Fatalf("protocol list = %v", entry.Protocols)
		}
	}
	for _, model := range dumate.Models() {
		if !found[model] {
			t.Fatalf("model %s missing from catalog", model)
		}
	}
	defaults, err := s.DefaultRoutes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	listed := map[string]bool{}
	for _, item := range defaults {
		if item.Preview.Routable && item.Preview.ProviderID == "builtin_dumate" {
			listed[item.Model] = true
		}
	}
	for _, model := range dumate.Models() {
		if !listed[model] {
			t.Fatalf("model %s missing from default routes", model)
		}
	}
	for _, change := range []string{
		`UPDATE accounts SET enabled=0 WHERE id='dumate-account'`,
		`UPDATE accounts SET enabled=1,auth_method='api_key' WHERE id='dumate-account'`,
		`UPDATE accounts SET enabled=1,auth_method='dumate_cookie',auth_state='reauth_required' WHERE id='dumate-account'`,
	} {
		if _, err := db.Exec(change); err != nil {
			t.Fatal(err)
		}
		preview, err := s.Resolve(ctx, dumate.ModelText)
		if err != nil || preview.Routable {
			t.Fatalf("ineligible account routed after %q: %+v %v", change, preview, err)
		}
	}
	if _, err := db.Exec(`UPDATE accounts SET auth_state='authenticated' WHERE id='dumate-account'`); err != nil {
		t.Fatal(err)
	}
	if preview, err := s.Resolve(ctx, dumate.ModelText); err != nil || !preview.Routable {
		t.Fatalf("restored session not routable: %+v %v", preview, err)
	}
	if _, err := db.Exec(`UPDATE providers SET enabled=0 WHERE id='builtin_dumate'`); err != nil {
		t.Fatal(err)
	}
	if preview, err := s.Resolve(ctx, dumate.ModelText); err != nil || preview.Routable {
		t.Fatalf("disabled provider still routed: %+v %v", preview, err)
	}
}
