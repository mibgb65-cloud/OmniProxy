package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/platform/profile"
)

func TestLegacyStatisticsSurviveFullStartupMigration(t *testing.T) {
	t.Setenv("OMNIPROXY_CONFIG_DIR", t.TempDir())
	t.Setenv("OMNIPROXY_DATA_DIR", "")
	if err := profile.SaveConfig(profile.Test, profile.Config{UpdateChannel: "stable", CloseBehavior: "tray"}); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	p := &profile.Profile{Kind: profile.Test, DataDir: root, LogDir: filepath.Join(root, "logs")}
	old, err := database.Open(database.Config{Path: p.DatabasePath()})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = old.Close() }()
	if _, err := old.Exec(`CREATE TABLE request_records(client,client_model,provider_model,success);
		INSERT INTO request_records VALUES('historical','old-model','old-model',1)`); err != nil {
		t.Fatal(err)
	}
	files, err := embeddedMigrations()
	if err != nil {
		t.Fatal(err)
	}
	// 复现 Beta.1 真实失败：前七个迁移已落盘，008 与历史表冲突。
	partial := fstest.MapFS{}
	names, err := os.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range names {
		if entry.Name() < "008" {
			data, err := os.ReadFile(filepath.Join("migrations", entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			partial[entry.Name()] = &fstest.MapFile{Data: data}
		}
	}
	if err := database.Migrate(context.Background(), old, partial); err != nil {
		t.Fatal(err)
	}
	if err := resolveDatabaseProfile(p); err != nil {
		t.Fatal(err)
	}
	if p.DataDir != filepath.Join(root, "v2") || p.LogDir != filepath.Join(root, "v2", "logs") {
		t.Fatalf("unexpected isolated profile: %+v", p)
	}
	fresh, err := database.Open(database.Config{Path: p.DatabasePath()})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = fresh.Close() }()
	if err := database.Migrate(context.Background(), fresh, files); err != nil {
		t.Fatal(err)
	}
	var history, versions int
	if err := old.QueryRow(`SELECT COUNT(*) FROM request_records WHERE client='historical'`).Scan(&history); err != nil || history != 1 {
		t.Fatalf("historical row changed: %d, %v", history, err)
	}
	if err := old.QueryRow(`SELECT COUNT(*) FROM schema_version`).Scan(&versions); err != nil || versions != 7 {
		t.Fatalf("historical migration state changed: %d, %v", versions, err)
	}
	cfg, err := profile.LoadConfig(profile.Test)
	if err != nil || cfg.DataDir != p.DataDir || cfg.PreviousDataDir != "" || cfg.UpdateChannel != "stable" || cfg.CloseBehavior != "tray" {
		t.Fatalf("isolated config not persisted: %+v, %v", cfg, err)
	}
	// 重启直接沿用已保存的新版库，不依赖旧统计库继续存在。
	t.Setenv("OMNIPROXY_PROFILE", "test")
	restarted, err := profile.Detect()
	if err != nil || restarted.DataDir != p.DataDir {
		t.Fatalf("restart profile: %+v, %v", restarted, err)
	}
	if err := resolveDatabaseProfile(restarted); err != nil {
		t.Fatal(err)
	}
}
