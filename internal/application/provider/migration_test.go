package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"omniproxy/internal/infrastructure/database"
)

func TestMigrationPreservesExistingConfigurationsAndAccounts(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "upgrade.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	initialSQL, err := os.ReadFile("../../../migrations/001_initial.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, db, fstest.MapFS{"001_initial.sql": &fstest.MapFile{Data: initialSQL}}); err != nil {
		t.Fatal(err)
	}
	const legacyID = "prov_0123456789abcdef0123456789abcdef"
	const credentialRef = "opaque-existing-ref"
	if _, err := db.Exec(`
		INSERT INTO providers (id, kind, name, base_url, enabled, created_at, updated_at)
		VALUES (?, 'openai', 'Existing endpoint', 'https://existing.example/v1', 0, 'old', 'old')`, legacyID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO accounts (id, provider_id, name, credential_ref, created_at, updated_at)
		VALUES ('account-1', ?, 'Existing account', ?, 'old', 'old')`, legacyID, credentialRef); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	items, err := NewService(db).List(ctx)
	if err != nil || len(items) != 12 {
		t.Fatalf("upgraded List = (%v, %v), want twelve", items, err)
	}
	legacy, err := NewService(db).get(ctx, legacyID)
	if err != nil || legacy.Source != SourceCustom || legacy.Enabled || legacy.Kind != "openai" || legacy.BaseURL != "https://existing.example/v1" {
		t.Fatalf("legacy configuration changed: (%+v, %v)", legacy, err)
	}
	if _, err := NewService(db).Update(ctx, legacyID, UpdateInput{Name: "Renamed legacy", BaseURL: ""}); err != nil {
		t.Fatalf("legacy configuration should remain editable: %v", err)
	}
	var gotRef string
	if err := db.QueryRow(`SELECT credential_ref FROM accounts WHERE provider_id = ?`, legacyID).Scan(&gotRef); err != nil || gotRef != credentialRef {
		t.Fatalf("account reference changed: (%q, %v)", gotRef, err)
	}
	for _, definition := range Catalog() {
		item, err := NewService(db).get(ctx, "builtin_"+definition.Kind)
		if err != nil || item.Source != SourceBuiltIn || !item.Enabled {
			t.Errorf("missing enabled preset %s: (%+v, %v)", definition.Kind, item, err)
		}
	}
	if _, err := NewService(db).SetEnabled(ctx, "builtin_openai", false); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	item, err := NewService(db).get(ctx, "builtin_openai")
	if err != nil || item.Enabled {
		t.Fatalf("reapplying migrations reset the switch: (%+v, %v)", item, err)
	}
}
