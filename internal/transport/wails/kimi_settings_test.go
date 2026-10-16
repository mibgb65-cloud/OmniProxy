package wails

import (
	"context"
	"omniproxy/internal/infrastructure/database"
	"os"
	"path/filepath"
	"testing"
)

func TestKimiSettingsBindingsAndFixedCodeConsole(t *testing.T) {
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "kimi.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	s := &AppService{db: db}
	v, err := s.GetKimiSettings()
	if err != nil || v.PreferredBillingMode != "coding" || v.PreferredRegion != "cn" {
		t.Fatal("defaults missing", err)
	}
	if _, err := s.SetKimiPreferredBillingMode("payg"); err != nil {
		t.Fatal(err)
	}
	v, err = s.SetKimiPreferredRegion("global")
	if err != nil || v.PreferredBillingMode != "payg" || v.PreferredRegion != "global" {
		t.Fatal("settings overwrite", err)
	}
	if _, err := s.SetKimiPreferredRegion("http://untrusted.invalid"); err == nil {
		t.Fatal("invalid region accepted")
	}
	if _, err := (&AppService{}).GetKimiSettings(); err == nil {
		t.Fatal("missing database accepted")
	}
}
