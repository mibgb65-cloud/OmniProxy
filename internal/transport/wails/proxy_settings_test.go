package wails

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"omniproxy/internal/application/proxysettings"
	"omniproxy/internal/infrastructure/database"
)

func TestProxySettingsBindingsPersistWithoutSecrets(t *testing.T) {
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "proxy.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	service := &AppService{db: db}

	defaults, err := service.GetProxySettings()
	if err != nil {
		t.Fatalf("defaults: %v", err)
	}
	if defaults.Enabled || defaults.Scheme != "auto" || defaults.Port != 10808 || defaults.HasPassword {
		t.Fatalf("unexpected defaults: %#v", defaults)
	}

	saved, err := service.SaveProxySettings(proxysettings.SaveInput{Enabled: true, Scheme: "auto", Host: "127.0.0.1", Port: 10809, NoProxy: "localhost"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if !saved.Enabled || saved.Endpoint != "auto://127.0.0.1:10809" {
		t.Fatalf("saved = %#v", saved)
	}

	item, err := service.SetProviderProxyMode("builtin_openai", "proxy")
	if err != nil {
		t.Fatalf("set provider proxy mode: %v", err)
	}
	if item.ProxyMode != "proxy" {
		t.Fatalf("proxy mode = %q", item.ProxyMode)
	}
	if _, err := service.SetProviderProxyMode("builtin_openai", "always"); err == nil {
		t.Fatal("invalid mode accepted")
	}

	result, err := service.TestProxySettings(proxysettings.SaveInput{Scheme: "auto", Host: "127.0.0.1", Port: 1})
	if err != nil {
		t.Fatalf("test: %v", err)
	}
	if result.Reachable || result.ErrorCode != "proxy_unavailable" {
		t.Fatalf("result = %#v", result)
	}

	if _, err := (&AppService{}).GetProxySettings(); err == nil {
		t.Fatal("unavailable database accepted")
	}
}
