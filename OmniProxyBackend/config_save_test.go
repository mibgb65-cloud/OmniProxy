package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"omniproxy/internal/config"
	"omniproxy/internal/logs"
	"omniproxy/internal/storage"
	"omniproxy/internal/token"
)

func TestSaveConfigImmediatelyScansCodexWhenAutoActivationIsEnabled(t *testing.T) {
	scanStarted := make(chan struct{}, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/backend-api/wham/usage" {
			return
		}
		select {
		case scanStarted <- struct{}{}:
		default:
		}
		_, _ = w.Write([]byte(`{"plan_type":"plus","rate_limit":{"primary_window":{"used_percent":2,"reset_at":1893456000,"window_minutes":300},"secondary_window":{"used_percent":3,"reset_at":1894060800,"window_minutes":10080}}}`))
	}))
	defer upstream.Close()

	initial := config.Default()
	initial.CodexBaseURL = upstream.URL + "/backend-api/codex"
	initial.CodexUsageEndpoint = upstream.URL + "/backend-api/wham/usage"
	app, _ := newConfigSaveTestApp(t, initial)
	if _, err := app.tokens.Add(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForMainTest(t, "settings-scan@example.com"),
	}); err != nil {
		t.Fatal(err)
	}

	next := initial
	next.CodexAutoActivateUsage = true
	if _, err := app.saveConfig(next); err != nil {
		t.Fatalf("saveConfig: %v", err)
	}
	select {
	case <-scanStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("expected Codex scan to start immediately after enabling setting")
	}

	app.mu.Lock()
	defer app.mu.Unlock()
	if !app.cfg.CodexAutoActivateUsage {
		t.Fatal("expected the enabled Codex activation setting to be applied")
	}
}

func TestSaveConfigRestoresProxyAndConfigWhenProxyRestartFails(t *testing.T) {
	occupied, occupiedPort := listenOnLocalhost(t)
	defer occupied.Close()
	oldListener, oldPort := listenOnLocalhost(t)
	oldListener.Close()

	initial := config.Default()
	initial.ProxyPort = oldPort
	app, store := newConfigSaveTestApp(t, initial)
	if err := app.startProxy(); err != nil {
		t.Fatal(err)
	}
	defer app.stopProxy()
	app.mu.Lock()
	oldProxy := app.proxyServer
	app.mu.Unlock()

	next := initial
	next.ProxyPort = occupiedPort
	if _, err := app.saveConfig(next); err == nil {
		t.Fatal("expected occupied proxy port to reject config")
	}

	app.mu.Lock()
	gotConfig := app.cfg
	proxyPreserved := app.proxyServer == oldProxy
	app.mu.Unlock()
	if gotConfig.ProxyPort != initial.ProxyPort {
		t.Fatalf("expected in-memory config rollback to %d, got %d", initial.ProxyPort, gotConfig.ProxyPort)
	}
	if !proxyPreserved {
		t.Fatal("expected failed port change to leave the old proxy untouched")
	}
	persisted, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if persisted.ProxyPort != initial.ProxyPort {
		t.Fatalf("expected persisted config rollback to %d, got %d", initial.ProxyPort, persisted.ProxyPort)
	}
}

func TestSaveConfigSwitchesProxyAfterNewPortBinds(t *testing.T) {
	oldListener, oldPort := listenOnLocalhost(t)
	oldListener.Close()
	newListener, newPort := listenOnLocalhost(t)
	newListener.Close()

	initial := config.Default()
	initial.ProxyPort = oldPort
	app, store := newConfigSaveTestApp(t, initial)
	if err := app.startProxy(); err != nil {
		t.Fatal(err)
	}
	defer app.stopProxy()
	app.mu.Lock()
	oldProxy := app.proxyServer
	app.mu.Unlock()

	next := initial
	next.ProxyPort = newPort
	if _, err := app.saveConfig(next); err != nil {
		t.Fatal(err)
	}

	app.mu.Lock()
	newProxy := app.proxyServer
	app.mu.Unlock()
	if newProxy == nil || newProxy == oldProxy {
		t.Fatal("expected proxy server to switch after the new listener binds")
	}
	if wantAddr := fmt.Sprintf("127.0.0.1:%d", newPort); newProxy.Addr != wantAddr {
		t.Fatalf("expected proxy address %q, got %q", wantAddr, newProxy.Addr)
	}
	persisted, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if persisted.ProxyPort != newPort {
		t.Fatalf("expected persisted proxy port %d, got %d", newPort, persisted.ProxyPort)
	}
}

func TestSaveConfigControlRestartFailureDoesNotRestartProxy(t *testing.T) {
	proxyListener, proxyPort := listenOnLocalhost(t)
	proxyListener.Close()
	controlListener, controlPort := listenOnLocalhost(t)
	controlListener.Close()
	occupiedControl, occupiedControlPort := listenOnLocalhost(t)
	defer occupiedControl.Close()

	initial := config.Default()
	initial.ProxyPort = proxyPort
	initial.ControlPort = controlPort
	app, store := newConfigSaveTestApp(t, initial)
	if err := app.startProxy(); err != nil {
		t.Fatal(err)
	}
	defer app.stopProxy()
	if err := app.startControl(); err != nil {
		t.Fatal(err)
	}
	defer app.stopControl()

	app.mu.Lock()
	oldProxy := app.proxyServer
	oldControl := app.control
	app.mu.Unlock()

	next := initial
	next.ControlPort = occupiedControlPort
	if _, err := app.saveConfig(next); err == nil {
		t.Fatal("expected occupied control port to reject config")
	}

	app.mu.Lock()
	gotConfig := app.cfg
	proxyPreserved := app.proxyServer == oldProxy
	controlPreserved := app.control == oldControl
	app.mu.Unlock()
	if gotConfig.ControlPort != initial.ControlPort {
		t.Fatalf("expected in-memory control port rollback to %d, got %d", initial.ControlPort, gotConfig.ControlPort)
	}
	if !proxyPreserved {
		t.Fatal("expected control restart failure to leave the proxy untouched")
	}
	if !controlPreserved {
		t.Fatal("expected control restart failure to leave the old control server untouched")
	}
	persisted, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if persisted.ControlPort != initial.ControlPort {
		t.Fatalf("expected persisted control port rollback to %d, got %d", initial.ControlPort, persisted.ControlPort)
	}
}

func TestSaveConfigRestoresProxyAfterControlRestartFails(t *testing.T) {
	oldProxyListener, oldProxyPort := listenOnLocalhost(t)
	oldProxyListener.Close()
	newProxyListener, newProxyPort := listenOnLocalhost(t)
	newProxyListener.Close()
	controlListener, controlPort := listenOnLocalhost(t)
	controlListener.Close()
	occupiedControl, occupiedControlPort := listenOnLocalhost(t)
	defer occupiedControl.Close()

	initial := config.Default()
	initial.ProxyPort = oldProxyPort
	initial.ControlPort = controlPort
	app, store := newConfigSaveTestApp(t, initial)
	if err := app.startProxy(); err != nil {
		t.Fatal(err)
	}
	defer app.stopProxy()
	if err := app.startControl(); err != nil {
		t.Fatal(err)
	}
	defer app.stopControl()
	app.mu.Lock()
	oldProxy := app.proxyServer
	app.mu.Unlock()

	next := initial
	next.ProxyPort = newProxyPort
	next.ControlPort = occupiedControlPort
	if _, err := app.saveConfig(next); err == nil {
		t.Fatal("expected occupied control port to reject config")
	}

	app.mu.Lock()
	gotConfig := app.cfg
	proxyAddr := ""
	if app.proxyServer != nil {
		proxyAddr = app.proxyServer.Addr
	}
	proxyPreserved := app.proxyServer == oldProxy
	app.mu.Unlock()
	if gotConfig.ProxyPort != oldProxyPort || gotConfig.ControlPort != controlPort {
		t.Fatalf("expected complete config rollback, got proxy=%d control=%d", gotConfig.ProxyPort, gotConfig.ControlPort)
	}
	if wantAddr := fmt.Sprintf("127.0.0.1:%d", oldProxyPort); proxyAddr != wantAddr {
		t.Fatalf("expected proxy rollback to %q, got %q", wantAddr, proxyAddr)
	}
	if !proxyPreserved {
		t.Fatal("expected unavailable control port to leave the old proxy untouched")
	}
	persisted, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if persisted.ProxyPort != oldProxyPort || persisted.ControlPort != controlPort {
		t.Fatalf("expected persisted config rollback, got proxy=%d control=%d", persisted.ProxyPort, persisted.ControlPort)
	}
}

func newConfigSaveTestApp(t *testing.T, initial config.Config) (*appServer, *config.Store) {
	t.Helper()
	dataDir := t.TempDir()
	store := config.NewStore(filepath.Join(dataDir, "config.json"))
	if err := store.Save(initial); err != nil {
		t.Fatal(err)
	}
	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(dataDir, "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	return &appServer{
		cfg:         initial,
		configStore: store,
		tokens:      manager,
		logs:        logs.NewRecorder(10),
	}, store
}
