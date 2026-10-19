package gateway

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"omniproxy/internal/infrastructure/database"
)

func TestRotationMigrationPreservesFlagsAndPersistsValidatedModes(t *testing.T) {
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "rotation.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	old := fstest.MapFS{}
	files, err := os.ReadDir("../../../migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sql") || file.Name() >= "035" {
			continue
		}
		data, err := os.ReadFile(filepath.Join("../../../migrations", file.Name()))
		if err != nil {
			t.Fatal(err)
		}
		old[file.Name()] = &fstest.MapFile{Data: data}
	}
	ctx := context.Background()
	if err := database.Migrate(ctx, db, old); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE codex_settings SET websocket_enabled=0,auto_quota_activation=1 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	service := NewCodexSettingsService(db)
	settings, err := service.Read(ctx)
	if err != nil || settings.AccountRotationMode != "queue" || settings.WebSocketEnabled || !settings.AutoQuotaActivation {
		t.Fatal("migration changed previous settings", settings, err)
	}
	for _, mode := range []string{"balanced", "queue", "balanced"} {
		settings, err = service.SetAccountRotationMode(ctx, mode)
		if err != nil || settings.AccountRotationMode != mode || settings.WebSocketEnabled || !settings.AutoQuotaActivation {
			t.Fatal("mode changed unrelated flags")
		}
		service = NewCodexSettingsService(db)
		if saved, err := service.AccountRotationMode(ctx); err != nil || saved != mode {
			t.Fatal("mode not persisted")
		}
	}
	for _, value := range []string{"", "random", "BALANCED", "balanced\n", strings.Repeat("a", 1024)} {
		if _, err := service.SetAccountRotationMode(ctx, value); err == nil {
			t.Fatal("invalid rotation mode accepted")
		}
	}
	if _, err := service.SetWebSocketEnabled(ctx, true); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetAutoQuotaActivation(ctx, false); err != nil {
		t.Fatal(err)
	}
	if settings, err := service.Read(ctx); err != nil || settings.AccountRotationMode != "balanced" {
		t.Fatal("other settings reset rotation mode")
	}
	if _, err := db.Exec(`UPDATE codex_settings SET account_rotation_mode='invalid'`); err == nil {
		t.Fatal("database accepted invalid rotation mode")
	}
	if _, err := db.Exec(`DELETE FROM codex_settings`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetAccountRotationMode(ctx, "queue"); err == nil {
		t.Fatal("missing record reported success")
	}
}
