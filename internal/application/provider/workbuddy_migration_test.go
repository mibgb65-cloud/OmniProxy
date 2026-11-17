package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"omniproxy/internal/infrastructure/database"
)

// TestWorkBuddyUpgradePreservesExistingStateAndRunsOnce 断言迁移 050/051/052 只追加
// WorkBuddy（国际版／国内版）预设与额度快照表，不改动既有服务商开关、账号与凭证引用，
// 并且重复执行不会重复插入。
func TestWorkBuddyUpgradePreservesExistingStateAndRunsOnce(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "upgrade.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	files, err := os.ReadDir("../../../migrations")
	if err != nil {
		t.Fatal(err)
	}
	before := fstest.MapFS{}
	for _, file := range files {
		if file.Name() == "050_builtin_workbuddy_ai.sql" || file.Name() == "051_workbuddy_quota_snapshots.sql" || file.Name() == "052_builtin_workbuddy.sql" {
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
	if _, err := db.Exec(`UPDATE providers SET enabled=0 WHERE id='builtin_dumate';
		INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at) VALUES('old','builtin_dumate','existing','opaque-reference','old','old')`); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	s := NewService(db)
	for _, want := range []struct{ id, kind, name string }{
		{"builtin_workbuddy_ai", "workbuddy_ai", "WorkBuddy AI"},
		{"builtin_workbuddy", "workbuddy", "WorkBuddy (China)"},
	} {
		item, err := s.get(ctx, want.id)
		if err != nil || !item.Enabled || item.Kind != want.kind || item.Name != want.name || item.Source != SourceBuiltIn || item.BaseURL != "" {
			t.Fatal("missing preset", want.id, item, err)
		}
	}
	var table int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='workbuddy_quota_snapshots'`).Scan(&table); err != nil || table != 1 {
		t.Fatal("missing quota snapshot table", table, err)
	}
	for _, id := range []string{"builtin_workbuddy_ai", "builtin_workbuddy"} {
		if _, err := s.SetEnabled(ctx, id, false); err != nil {
			t.Fatal(err)
		}
	}
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"builtin_dumate", "builtin_workbuddy_ai", "builtin_workbuddy"} {
		item, err := s.get(ctx, id)
		if err != nil || item.Enabled {
			t.Fatal("saved switch reset", id, err)
		}
	}
	var ref string
	if err := db.QueryRow(`SELECT credential_ref FROM accounts WHERE id='old'`).Scan(&ref); err != nil || ref != "opaque-reference" {
		t.Fatal("credential reference changed", err)
	}
	for kind := range map[string]bool{"workbuddy_ai": true, "workbuddy": true} {
		var count int
		if err := db.QueryRow(`SELECT count(*) FROM providers WHERE kind=?`, kind).Scan(&count); err != nil || count != 1 {
			t.Fatal("migration not idempotent", kind, count, err)
		}
	}
}
