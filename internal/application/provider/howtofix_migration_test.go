package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"omniproxy/internal/infrastructure/database"
)

// TestHowToFixUpgradePreservesExistingStateAndRunsOnce 断言迁移 049 只追加预设，
// 不改动既有服务商开关、账号与凭证引用，并且重复执行不会重复插入。
func TestHowToFixUpgradePreservesExistingStateAndRunsOnce(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "upgrade.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	files, err := os.ReadDir("../../../migrations")
	if err != nil {
		t.Fatal(err)
	}
	before := fstest.MapFS{}
	for _, file := range files {
		if file.Name() == "049_builtin_howtofix.sql" {
			continue
		}
		body, err := os.ReadFile("../../../migrations/" + file.Name())
		if err != nil {
			t.Fatal(err)
		}
		before[file.Name()] = &fstest.MapFile{Data: body}
	}
	if err := database.Migrate(ctx, db, before); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE providers SET enabled=0 WHERE id='builtin_oxionai';
		INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at) VALUES('old','builtin_oxionai','existing','opaque-reference','old','old')`); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	s := NewService(db)
	item, err := s.get(ctx, "builtin_howtofix")
	if err != nil || !item.Enabled || item.Kind != "howtofix" || item.Name != "HowToFix" || item.Source != SourceBuiltIn || item.BaseURL != "" {
		t.Fatal("missing preset", item, err)
	}
	if _, err := s.SetEnabled(ctx, "builtin_howtofix", false); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"builtin_oxionai", "builtin_howtofix"} {
		item, err := s.get(ctx, id)
		if err != nil || item.Enabled {
			t.Fatal("saved switch reset", id, err)
		}
	}
	var ref string
	if err := db.QueryRow(`SELECT credential_ref FROM accounts WHERE id='old'`).Scan(&ref); err != nil || ref != "opaque-reference" {
		t.Fatal("credential reference changed", err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM providers WHERE kind='howtofix'`).Scan(&count); err != nil || count != 1 {
		t.Fatal("migration not idempotent", err)
	}
}
