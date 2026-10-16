package wails

import (
	"context"
	"omniproxy/internal/infrastructure/database"
	"os"
	"path/filepath"
	"testing"
)

func TestStepFunSettingsBindingsPersistWithoutCredentials(t *testing.T) {
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "stepfun-settings.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	s := &AppService{db: db}
	if settings, err := s.GetStepFunSettings(); err != nil || settings.PreferredBillingMode != "step_plan" {
		t.Fatal(settings, err)
	}
	if settings, err := s.SetStepFunPreferredBillingMode("payg"); err != nil || settings.PreferredBillingMode != "payg" {
		t.Fatal(settings, err)
	}
	if settings, err := (&AppService{db: db}).GetStepFunSettings(); err != nil || settings.PreferredBillingMode != "payg" {
		t.Fatal(settings, err)
	}
	if _, err := s.SetStepFunPreferredBillingMode("api_key"); err == nil {
		t.Fatal("invalid mode accepted")
	}
	if _, err := (&AppService{}).GetStepFunSettings(); err == nil {
		t.Fatal("missing database accepted")
	}
}
