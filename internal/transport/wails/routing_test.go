package wails

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/platform/profile"
)

func TestRoutingBindingsWithoutCredentialAccess(t *testing.T) {
	prof, err := profile.New(profile.Test)
	if err != nil {
		t.Fatal(err)
	}
	prof.DataDir = t.TempDir()
	db, err := database.Open(database.Config{Path: filepath.Join(prof.DataDir, "routing.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at) VALUES('acct_one','builtin_deepseek','Metadata only','unread',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	s := NewAppService(prof, nil, db, nil, nil)
	defaults, err := s.ListDefaultRoutes()
	if err != nil || len(defaults) != 2 || !defaults[0].Preview.Routable {
		t.Fatalf("default binding: %+v %v", defaults, err)
	}
	if err := s.SetModelPreferredProvider("deepseek-flash", "builtin_deepseek"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetModelProviderOrder("deepseek-flash", []string{"builtin_deepseek"}); err != nil {
		t.Fatal(err)
	}
	defaults, err = s.ListDefaultRoutes()
	if err != nil || len(defaults[0].ProviderOrder) != 1 || defaults[0].ProviderOrder[0].ID != "builtin_deepseek" {
		t.Fatal("priority binding not saved", err)
	}
	if err := s.SetModelProviderOrder("deepseek-flash", []string{"builtin_deepseek", "builtin_deepseek"}); err == nil {
		t.Fatal("binding accepted duplicate priority")
	}
	if err := s.SetModelPreferredProvider("deepseek-flash", ""); err != nil {
		t.Fatal(err)
	}
	input := routes.Input{Alias: "coding-fast", ProviderID: "builtin_deepseek", Model: "deepseek-flash", AccountID: "acct_one", Enabled: true}
	rule, err := s.CreateRoute(input)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := s.PreviewRoute("coding-fast")
	if err != nil || !preview.Routable || preview.AccountID != "acct_one" {
		t.Fatalf("preview: %+v %v", preview, err)
	}
	if _, err := os.Stat(filepath.Join(prof.DataDir, "credentials")); !os.IsNotExist(err) {
		t.Fatal("preview accessed protected storage")
	}
	input.Model = "deepseek-v4-pro"
	if _, err := s.UpdateRoute(rule.ID, input); err != nil {
		t.Fatal(err)
	}
	preview, err = s.PreviewRoute("coding-fast")
	if err != nil || preview.Model != "deepseek-v4-pro" {
		t.Fatal("updated target not visible")
	}
	if _, err := s.SetRouteEnabled(rule.ID, false); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteRoute(rule.ID); err != nil {
		t.Fatal(err)
	}
	items, err := s.ListRoutes()
	if err != nil || len(items) != 0 {
		t.Fatal("deleted rule remains")
	}
}
