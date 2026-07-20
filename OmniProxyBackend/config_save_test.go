package main

import (
	"path/filepath"
	"testing"

	"omniproxy/internal/config"
	"omniproxy/internal/logs"
	"omniproxy/internal/storage"
	"omniproxy/internal/token"
)

func TestSaveConfigRestoresProxyAndConfigWhenProxyRestartFails(t *testing.T) {
	occupied, occupiedPort := listenOnLocalhost(t)
	defer occupied.Close()
	oldListener, oldPort := listenOnLocalhost(t)
	oldListener.Close()

	dataDir := t.TempDir()
	initial := config.Default()
	initial.ProxyPort = oldPort
	store := config.NewStore(filepath.Join(dataDir, "config.json"))
	if err := store.Save(initial); err != nil {
		t.Fatal(err)
	}
	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(dataDir, "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	app := &appServer{
		cfg:         initial,
		configStore: store,
		tokens:      manager,
		logs:        logs.NewRecorder(10),
	}
	if err := app.startProxy(); err != nil {
		t.Fatal(err)
	}
	defer app.stopProxy()

	next := initial
	next.ProxyPort = occupiedPort
	if _, err := app.saveConfig(next); err == nil {
		t.Fatal("expected occupied proxy port to reject config")
	}

	app.mu.Lock()
	gotConfig := app.cfg
	proxyRunning := app.proxyServer != nil
	app.mu.Unlock()
	if gotConfig.ProxyPort != initial.ProxyPort {
		t.Fatalf("expected in-memory config rollback to %d, got %d", initial.ProxyPort, gotConfig.ProxyPort)
	}
	if !proxyRunning {
		t.Fatal("expected old proxy to be restored after failed restart")
	}
	persisted, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if persisted.ProxyPort != initial.ProxyPort {
		t.Fatalf("expected persisted config rollback to %d, got %d", initial.ProxyPort, persisted.ProxyPort)
	}
}
