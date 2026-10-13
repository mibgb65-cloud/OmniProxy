package wails

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"omniproxy/internal/infrastructure/database"
)

func TestCodexSettingsBindingsPersistWithoutClientAccess(t *testing.T) {
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "settings.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	// 不初始化 Agent、凭据或桌面实例，证明开关只依赖非敏感网关设置。
	service := &AppService{db: db}
	if settings, err := service.GetCodexSettings(); err != nil || !settings.WebSocketEnabled {
		t.Fatal("default settings missing", err)
	}
	if settings, err := service.SetCodexWebSocketEnabled(false); err != nil || settings.WebSocketEnabled {
		t.Fatal("disable failed", err)
	}
	service = &AppService{db: db}
	if _, err := service.SetCodexQuotaAutoActivation(true); err == nil {
		t.Fatal("uninitialized activation worker accepted authorization")
	}
	if settings, err := service.SetCodexQuotaAutoActivation(false); err != nil || settings.AutoQuotaActivation {
		t.Fatal("cannot disable unavailable worker", err)
	}
	if settings, err := service.GetCodexSettings(); err != nil || settings.WebSocketEnabled {
		t.Fatal("state not persisted", err)
	}
	if _, err := (&AppService{}).GetCodexSettings(); err == nil {
		t.Fatal("unavailable database accepted")
	}
}
