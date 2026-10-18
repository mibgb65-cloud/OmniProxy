//go:build windows

package wails

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"omniproxy/internal/infrastructure/credential"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/platform/profile"
)

func TestLegacyImportBindingsReprotectInIsolatedProfile(t *testing.T) {
	prof := &profile.Profile{Kind: profile.Test, DataDir: t.TempDir()}
	db, err := database.Open(database.Config{Path: prof.DatabasePath()})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	svc := &AppService{prof: prof, db: db}
	if _, err := svc.PreviewLegacyImport(""); err == nil {
		t.Fatal("test profile may not read a real source automatically")
	}
	if _, err := svc.PreviewLegacyImport(prof.DataDir); err == nil {
		t.Fatal("target accepted as source")
	}
	directory := t.TempDir()
	const original = `[{"id":"fake-legacy","name":"测试旧账号","provider":"deepseek","tokenValue":"sk-only-a-fake-import-test","disabled":true}]`
	if err := os.WriteFile(filepath.Join(directory, "tokens.json"), []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	preview, err := svc.PreviewLegacyImport(directory)
	if err != nil || preview.Ready != 1 {
		t.Fatalf("preview=%+v %v", preview, err)
	}
	result, err := svc.ImportLegacyData(directory, preview.Snapshot)
	if err != nil || result.Imported != 1 {
		t.Fatalf("result=%+v %v", result, err)
	}
	var ref string
	if err := db.QueryRow("SELECT credential_ref FROM accounts").Scan(&ref); err != nil {
		t.Fatal(err)
	}
	store, err := credential.New(prof)
	if err != nil {
		t.Fatal(err)
	}
	key, err := store.Get(ref)
	if err != nil || string(key) != "sk-only-a-fake-import-test" {
		t.Fatal("new protected credential unavailable")
	}
	clear(key)
	items, err := svc.ListAccounts()
	if err != nil || len(items) != 1 || items[0].Enabled || items[0].Name != "测试旧账号" {
		t.Fatalf("account metadata=%+v %v", items, err)
	}
	data, err := os.ReadFile(filepath.Join(directory, "tokens.json"))
	if err != nil || string(data) != original {
		t.Fatal("original file changed")
	}
}
