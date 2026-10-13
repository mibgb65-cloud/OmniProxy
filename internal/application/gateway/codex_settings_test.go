package gateway

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"omniproxy/internal/infrastructure/database"
)

func TestCodexWebSocketSettingsDefaultPersistenceAndFailure(t *testing.T) {
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "settings.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	service := NewCodexSettingsService(db)
	if settings, err := service.Read(ctx); err != nil || !settings.WebSocketEnabled {
		t.Fatal("default changed", err)
	}
	for _, enabled := range []bool{false, true, false} {
		if settings, err := service.SetWebSocketEnabled(ctx, enabled); err != nil || settings.WebSocketEnabled != enabled {
			t.Fatal("save failed", err)
		}
		service = NewCodexSettingsService(db)
		if settings, err := service.Read(ctx); err != nil || settings.WebSocketEnabled != enabled {
			t.Fatal("setting not persisted", err)
		}
	}
	if _, err := db.Exec(`DELETE FROM codex_settings`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Read(ctx); err == nil {
		t.Fatal("missing record silently defaulted")
	}
	if _, err := service.SetWebSocketEnabled(ctx, true); err == nil {
		t.Fatal("missing record reported save success")
	}
	if _, err := db.Exec(`DROP TABLE codex_settings`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.WebSocketAllowed(ctx); err == nil {
		t.Fatal("database failure accepted")
	}
}

func TestCodexActivationSettingPreservesWebSocket(t *testing.T) {
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "settings.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	service := NewCodexSettingsService(db)
	original, err := service.Read(ctx)
	if err != nil || original.AutoQuotaActivation || !original.WebSocketEnabled {
		t.Fatal("invalid defaults", err)
	}
	if settings, err := service.SetAutoQuotaActivation(ctx, true); err != nil || !settings.AutoQuotaActivation || !settings.WebSocketEnabled {
		t.Fatal("activation changed websocket", err)
	}
	if settings, err := service.SetWebSocketEnabled(ctx, false); err != nil || !settings.AutoQuotaActivation || settings.WebSocketEnabled {
		t.Fatal("websocket disabled activation", err)
	}
	service = NewCodexSettingsService(db)
	if settings, err := service.SetAutoQuotaActivation(ctx, false); err != nil || settings.AutoQuotaActivation || settings.WebSocketEnabled {
		t.Fatal("independent settings not saved", err)
	}
}
