package wails

import (
	"context"
	"omniproxy/internal/infrastructure/database"
	"os"
	"path/filepath"
	"testing"
)

func TestMiMoSettingsBindingsPersistWithoutCredentials(t *testing.T) {
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "mimo-settings.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	s := &AppService{db: db}
	if settings, err := s.GetMiMoSettings(); err != nil || settings.PreferredBillingMode != "token_plan" {
		t.Fatal(settings, err)
	}
	if settings, err := s.SetMiMoPreferredBillingMode("payg"); err != nil || settings.PreferredBillingMode != "payg" {
		t.Fatal(settings, err)
	}
	if settings, err := (&AppService{db: db}).GetMiMoSettings(); err != nil || settings.PreferredBillingMode != "payg" {
		t.Fatal(settings, err)
	}
	if _, err := s.SetMiMoPreferredBillingMode("api_key"); err == nil {
		t.Fatal("invalid mode accepted")
	}
	if _, err := (&AppService{}).GetMiMoSettings(); err == nil {
		t.Fatal("missing database accepted")
	}
}
