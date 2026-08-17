package main

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"omniproxy/internal/config"
	"omniproxy/internal/history"
	"omniproxy/internal/logs"
	"omniproxy/internal/storage"
	"omniproxy/internal/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConfigureCodexSyncsExistingAuthJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codexDir, "auth.json"), []byte(codexAuthJSONForMainTestWithCredentials(t, "coder@example.com", "same-account", "new-access-token")), 0o600); err != nil {
		t.Fatal(err)
	}

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	item, err := manager.Add(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForMainTestWithCredentials(t, "coder@example.com", "same-account", "old-access-token"),
	})
	if err != nil {
		t.Fatal(err)
	}

	app := &appServer{
		cfg: config.Config{
			ProxyPort:       3000,
			ControlPort:     3890,
			UpstreamBaseURL: "https://api.openai.com",
			SwitchThreshold: 15,
			MaxRetries:      1,
		},
		tokens: manager,
		logs:   logs.NewRecorder(10),
	}

	result, err := app.configureCodex()
	if err != nil {
		t.Fatal(err)
	}
	if !result.AuthUpdated || result.ImportedAuth {
		t.Fatalf("expected existing auth to be synced, got %#v", result)
	}

	updated, err := manager.Get(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(updated.TokenValue, "new-access-token") || !strings.Contains(updated.TokenValue, "same-account") {
		t.Fatalf("expected stored auth.json to be refreshed, got %s", updated.TokenValue)
	}
}

func TestConfigureCodexImportsSameEmailDifferentAccountID(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	authValue := codexAuthJSONForMainTestWithCredentials(t, "coder@example.com", "new-account", "new-access-token")
	if err := os.WriteFile(filepath.Join(codexDir, "auth.json"), []byte(authValue), 0o600); err != nil {
		t.Fatal(err)
	}
	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(token.UpsertRequest{Provider: token.ProviderOpenAI, CredentialType: token.CredentialTypeCodexAuthJSON, TokenValue: codexAuthJSONForMainTestWithCredentials(t, "coder@example.com", "old-account", "old-access-token")}); err != nil {
		t.Fatal(err)
	}
	app := &appServer{cfg: config.Config{ProxyPort: 3000, ControlPort: 3890, SwitchThreshold: 15, MaxRetries: 1}, tokens: manager, logs: logs.NewRecorder(10)}

	result, err := app.configureCodex()
	if err != nil {
		t.Fatal(err)
	}
	if !result.ImportedAuth || result.AuthUpdated || result.AuthAlreadyAdded {
		t.Fatalf("expected different account_id to be imported, got %#v", result)
	}
	if items := manager.List(); len(items) != 2 {
		t.Fatalf("expected both Codex accounts to be retained, got %d", len(items))
	}
}

func TestConfigureCodexReportsAlreadyImportedAuthJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	authValue := codexAuthJSONForMainTestWithCredentials(t, "coder@example.com", "same-account", "same-access-token")
	if err := os.WriteFile(filepath.Join(codexDir, "auth.json"), []byte(authValue), 0o600); err != nil {
		t.Fatal(err)
	}

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     authValue,
	}); err != nil {
		t.Fatal(err)
	}

	app := &appServer{
		cfg: config.Config{
			ProxyPort:       3000,
			ControlPort:     3890,
			UpstreamBaseURL: "https://api.openai.com",
			SwitchThreshold: 15,
			MaxRetries:      1,
		},
		tokens: manager,
		logs:   logs.NewRecorder(10),
	}

	result, err := app.configureCodex()
	if err != nil {
		t.Fatal(err)
	}
	if !result.AuthAlreadyAdded || result.AuthUpdated || result.ImportedAuth {
		t.Fatalf("expected auth to be reported as already imported, got %#v", result)
	}
	if !strings.Contains(result.Message, "auth.json 账号已存在") {
		t.Fatalf("expected already-added message, got %q", result.Message)
	}
	if items := manager.List(); len(items) != 1 {
		t.Fatalf("expected no duplicate token to be created, got %d", len(items))
	}
}

func TestConfigureCodexUsesAPIKeyOnlyAuthAsFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codexDir, "auth.json"), []byte(`{"OPENAI_API_KEY":"sk-local-placeholder"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	app := &appServer{
		cfg: config.Config{
			ProxyPort:       3000,
			ControlPort:     3890,
			UpstreamBaseURL: "https://api.openai.com",
			SwitchThreshold: 15,
			MaxRetries:      1,
		},
		tokens: manager,
		logs:   logs.NewRecorder(10),
	}

	result, err := app.configureCodex()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Message, "已自动切换为 API 登录模式") {
		t.Fatalf("expected API login fallback message, got %q", result.Message)
	}
	if items := manager.List(); len(items) != 0 {
		t.Fatalf("expected no Codex token to be imported, got %d", len(items))
	}
	configContent, err := os.ReadFile(filepath.Join(codexDir, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(configContent), `forced_login_method = "api"`) {
		t.Fatalf("expected API login config, got:\n%s", configContent)
	}
	authContent, err := os.ReadFile(filepath.Join(codexDir, "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(authContent), "sk-local-placeholder") {
		t.Fatalf("expected existing API key login to be preserved, got %s", authContent)
	}
}

func TestMigrateLegacyRequestHistoryAssignsIDsForZeroIDEntries(t *testing.T) {
	dataDir := t.TempDir()
	legacyPath := filepath.Join(dataDir, "request_history.json")
	now := time.Now()
	entries := []history.Entry{
		{
			Time:     now.Add(-time.Minute),
			Level:    "info",
			Provider: "openai",
			Status:   http.StatusOK,
			Message:  "first legacy entry",
		},
		{
			Time:     now,
			Level:    "warn",
			Provider: "openai",
			Status:   http.StatusTooManyRequests,
			Message:  "second legacy entry",
		},
	}
	raw, err := json.Marshal(entries)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	store, err := history.NewSQLiteStore(filepath.Join(dataDir, "request_history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if err := migrateLegacyRequestHistory(store, legacyPath); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.List(history.Filter{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected both legacy entries to be imported, got %#v", loaded)
	}
	if loaded[0].ID == 0 || loaded[1].ID == 0 || loaded[0].ID == loaded[1].ID {
		t.Fatalf("expected generated unique IDs, got %#v", loaded)
	}
}

func TestProxyConfigChanged(t *testing.T) {
	base := config.Default()

	if proxyConfigChanged(base, base) {
		t.Fatal("same config should not require proxy restart")
	}

	thresholdOnly := base
	thresholdOnly.SwitchThreshold = 30
	if proxyConfigChanged(base, thresholdOnly) {
		t.Fatal("threshold-only change should not require proxy restart")
	}

	next := base
	next.UpstreamBaseURL = "https://example.com"
	if !proxyConfigChanged(base, next) {
		t.Fatal("upstream change should require proxy restart")
	}

	next = base
	next.KimiBaseURL = "https://example.com/coding"
	if !proxyConfigChanged(base, next) {
		t.Fatal("kimi base url change should require proxy restart")
	}

	next = base
	next.SchedulingMode = config.SchedulingModeBalanced
	if !proxyConfigChanged(base, next) {
		t.Fatal("scheduling mode change should require proxy restart")
	}

	next = base
	next.WebSocketMode = config.WebSocketModeDisabled
	if !proxyConfigChanged(base, next) {
		t.Fatal("websocket mode change should require proxy restart")
	}
}

func TestValidateConfiguredPortsRejectsInvalidPorts(t *testing.T) {
	cfg := config.Default()
	cfg.ControlPort = cfg.ProxyPort
	if err := validateConfiguredPorts(cfg); err == nil {
		t.Fatal("expected matching proxy and control ports to fail")
	}

	cfg = config.Default()
	cfg.ProxyPort = 70000
	if err := validateConfiguredPorts(cfg); err == nil {
		t.Fatal("expected out-of-range proxy port to fail")
	}
}

func TestStartProxyReturnsErrorWhenPortOccupied(t *testing.T) {
	listener, port := listenOnLocalhost(t)
	defer listener.Close()

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.ProxyPort = port
	app := &appServer{
		cfg:    cfg,
		tokens: manager,
		logs:   logs.NewRecorder(10),
	}

	err = app.startProxy()
	if err == nil {
		t.Fatal("expected occupied proxy port to fail")
	}
	if !strings.Contains(err.Error(), "start proxy listener") {
		t.Fatalf("expected listener error, got %v", err)
	}
	if app.proxyServer != nil {
		t.Fatal("proxy server should not be marked running after listen failure")
	}
}

func TestStartControlReturnsErrorWhenPortOccupied(t *testing.T) {
	listener, port := listenOnLocalhost(t)
	defer listener.Close()

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.ControlPort = port
	app := &appServer{
		cfg:    cfg,
		tokens: manager,
		logs:   logs.NewRecorder(10),
	}

	err = app.startControl()
	if err == nil {
		t.Fatal("expected occupied control API port to fail")
	}
	if !strings.Contains(err.Error(), "start control API listener") {
		t.Fatalf("expected listener error, got %v", err)
	}
	if app.control != nil {
		t.Fatal("control server should not be marked running after listen failure")
	}
}

func TestSaveConfigRejectsInvalidURLBeforePersisting(t *testing.T) {
	dataDir := t.TempDir()
	store := config.NewStore(filepath.Join(dataDir, "config.json"))
	initial := config.Default()
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

	bad := initial
	bad.OpenAIBaseURL = "://bad-url"
	if _, err := app.saveConfig(bad); err == nil {
		t.Fatal("expected invalid url to be rejected")
	}

	reloaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.OpenAIBaseURL != initial.OpenAIBaseURL {
		t.Fatalf("invalid url should not be persisted, got %q", reloaded.OpenAIBaseURL)
	}
	if app.cfg.OpenAIBaseURL != initial.OpenAIBaseURL {
		t.Fatalf("invalid url should not update runtime config, got %q", app.cfg.OpenAIBaseURL)
	}
}

func TestSaveConfigRestartsControlAPIWhenPortChanges(t *testing.T) {
	firstListener, firstPort := listenOnLocalhost(t)
	secondListener, secondPort := listenOnLocalhost(t)
	proxyListener, proxyPort := listenOnLocalhost(t)
	firstListener.Close()
	secondListener.Close()
	proxyListener.Close()

	dataDir := t.TempDir()
	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(dataDir, "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.ProxyPort = proxyPort
	cfg.ControlPort = firstPort
	app := &appServer{
		cfg:          cfg,
		configStore:  config.NewStore(filepath.Join(dataDir, "config.json")),
		tokens:       manager,
		logs:         logs.NewRecorder(10),
		controlToken: "test-control-token",
	}
	if err := app.startControl(); err != nil {
		t.Fatal(err)
	}
	defer app.stopControl()

	next := cfg
	next.ControlPort = secondPort
	if _, err := app.saveConfig(next); err != nil {
		t.Fatal(err)
	}

	app.mu.Lock()
	addr := ""
	if app.control != nil {
		addr = app.control.Addr
	}
	app.mu.Unlock()
	if !strings.HasSuffix(addr, intToString(secondPort)) {
		t.Fatalf("expected control API to move to port %d, got %q", secondPort, addr)
	}

	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:"+intToString(secondPort)+"/api/logs", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(controlTokenHeader, "test-control-token")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected new control API port to respond, got %d", res.StatusCode)
	}
}

func TestChangeDataDirectoryMigratesFilesAndSavesBootstrap(t *testing.T) {
	home := t.TempDir()
	appData := t.TempDir()
	t.Setenv("OMNIPROXY_DATA_DIR", "")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", appData)
	t.Setenv("XDG_CONFIG_HOME", appData)

	currentDir := filepath.Join(t.TempDir(), "current")
	nextDir := filepath.Join(t.TempDir(), "next")
	if err := os.MkdirAll(currentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(currentDir, "config.json"), []byte(`{"proxyPort":3000}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(currentDir, "tokens.json"), []byte(`[]`), 0o600); err != nil {
		t.Fatal(err)
	}

	app := &appServer{
		dataDir: currentDir,
		cfg:     config.Default(),
		logs:    logs.NewRecorder(10),
	}
	result, err := app.changeDataDirectory(nextDir, true)
	if err != nil {
		t.Fatal(err)
	}
	if !result.RestartRequired {
		t.Fatal("expected data directory change to require restart")
	}
	for _, name := range []string{"config.json", "tokens.json"} {
		if _, err := os.Stat(filepath.Join(nextDir, name)); err != nil {
			t.Fatalf("expected migrated %s: %v", name, err)
		}
	}
	bootstrap, err := os.ReadFile(config.BootstrapPath())
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		DataDir string `json:"dataDir"`
	}
	if err := json.Unmarshal(bootstrap, &saved); err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(filepath.Clean(saved.DataDir), filepath.Clean(nextDir)) {
		t.Fatalf("expected bootstrap to point at next data dir, got %s", string(bootstrap))
	}
}

func listenOnLocalhost(t *testing.T) (net.Listener, int) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		t.Fatalf("expected TCP listener, got %T", listener.Addr())
	}
	return listener, addr.Port
}

func codexAuthJSONForMainTest(t *testing.T, email string) string {
	return codexAuthJSONForMainTestWithCredentials(t, email, "account-123", "codex-access-token")
}

func codexAuthJSONForMainTestWithCredentials(t *testing.T, email string, accountID string, accessToken string) string {
	t.Helper()

	authJSON, err := json.Marshal(map[string]any{
		"tokens": map[string]string{
			"access_token": accessToken,
			"account_id":   accountID,
			"id_token":     codexIDTokenForMainTest(t, email),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(authJSON)
}

func codexAuthJSONForMainTestWithRefreshToken(t *testing.T, email string, accountID string, accessToken string, refreshToken string) string {
	t.Helper()

	authJSON, err := json.Marshal(map[string]any{
		"tokens": map[string]string{
			"access_token":  accessToken,
			"account_id":    accountID,
			"id_token":      codexIDTokenForMainTest(t, email),
			"refresh_token": refreshToken,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(authJSON)
}

func codexIDTokenForMainTest(t *testing.T, email string) string {
	t.Helper()

	payload, err := json.Marshal(map[string]any{
		"https://api.openai.com/profile": map[string]string{"email": email},
		"https://api.openai.com/auth":    map[string]string{"chatgpt_plan_type": "plus"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestWriteCodexOmniProxyConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	initial := strings.Join([]string{
		`model = "gpt-5.5"`,
		`model_provider = "openai"`,
		`model_catalog_json = "~/.codex/models.json"`,
		`chatgpt_base_url = "http://127.0.0.1:3000/backend-api/"`,
		`disable_response_storage = true`,
		``,
		`[projects.'E:\go\OmniProxy']`,
		`trust_level = "trusted"`,
		``,
		`[model_providers.omniproxy]`,
		`base_url = "http://old.example/v1"`,
		``,
		`[model_providers.deepseek]`,
		`experimental_bearer_token = "sk-fake-deepseek"`,
	}, "\n")
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := writeCodexOmniProxyConfig(path, "http://127.0.0.1:3000/backend-api/codex"); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, expected := range []string{
		`model_provider = "openai"`,
		`openai_base_url = "http://127.0.0.1:3000/backend-api/codex"`,
		`[projects.'E:\go\OmniProxy']`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected config to contain %q, got:\n%s", expected, text)
		}
	}
	if strings.Contains(text, "old.example") {
		t.Fatalf("old omniproxy section was not removed:\n%s", text)
	}
	if strings.Contains(text, "[model_providers.omniproxy]") {
		t.Fatalf("legacy omniproxy section was not removed:\n%s", text)
	}
	if strings.Contains(text, "[model_providers.openai]") {
		t.Fatalf("reserved openai provider section was not removed:\n%s", text)
	}
	if strings.Contains(text, "chatgpt_base_url") {
		t.Fatalf("chatgpt_base_url should not be used for the model proxy:\n%s", text)
	}
	if strings.Contains(text, "disable_response_storage") {
		t.Fatalf("disable_response_storage should not be kept for Codex history:\n%s", text)
	}
	if strings.Contains(text, "model_catalog_json") || strings.Contains(text, "[model_providers.deepseek]") || strings.Contains(text, "sk-fake-deepseek") {
		t.Fatalf("DeepSeek-only catalog and provider should be removed:\n%s", text)
	}
	if _, err := os.Stat(path + ".omniproxy.bak"); err != nil {
		t.Fatalf("expected backup file: %v", err)
	}
}

func TestWriteCodexOmniProxyConfigKeepsOriginalBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	first := `model_provider = "openai"` + "\n"
	if err := os.WriteFile(path, []byte(first), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeCodexOmniProxyConfig(path, "http://127.0.0.1:3000/backend-api/codex"); err != nil {
		t.Fatal(err)
	}
	if err := writeCodexOmniProxyConfig(path, "http://127.0.0.1:3001/backend-api/codex"); err != nil {
		t.Fatal(err)
	}

	backup, err := os.ReadFile(path + ".omniproxy.bak")
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != first {
		t.Fatalf("backup should keep original config, got:\n%s", string(backup))
	}
}
