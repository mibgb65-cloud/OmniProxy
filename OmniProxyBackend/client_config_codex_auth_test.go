package main

import (
	"encoding/base64"
	"encoding/json"
	"omniproxy/internal/config"
	"omniproxy/internal/logs"
	"omniproxy/internal/storage"
	"omniproxy/internal/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigureCodexWritesPoolAccountForChatGPTLogin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	var storedAuth map[string]any
	if err := json.Unmarshal([]byte(codexAuthJSONForMainTestWithCredentials(t, "pool@example.com", "pool-account", "pool-access-token")), &storedAuth); err != nil {
		t.Fatal(err)
	}
	storedAuth["OPENAI_API_KEY"] = "sk-api-login-must-not-be-used"
	authBytes, err := json.Marshal(storedAuth)
	if err != nil {
		t.Fatal(err)
	}
	authValue := string(authBytes)
	if _, err := manager.Add(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     authValue,
	}); err != nil {
		t.Fatal(err)
	}

	app := &appServer{cfg: config.Config{ProxyPort: 3000}, tokens: manager, logs: logs.NewRecorder(10)}
	result, err := app.configureCodex()
	if err != nil {
		t.Fatal(err)
	}
	if !result.AuthUpdated || !strings.Contains(result.Message, "实际请求仍由 OmniProxy 后端 auth 池调度") {
		t.Fatalf("expected pool account setup result, got %#v", result)
	}

	content, err := os.ReadFile(filepath.Join(home, ".codex", "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		t.Fatal(err)
	}
	if _, exists := payload["OPENAI_API_KEY"]; exists {
		t.Fatalf("API key login must not be written: %#v", payload)
	}
	if !strings.Contains(string(content), "pool-account") || !strings.Contains(string(content), "pool-access-token") {
		t.Fatalf("expected selected account auth.json to be written, got %s", content)
	}
}

func TestConfigureCodexCreatesAPIKeyFallbackWithoutAccount(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	app := &appServer{cfg: config.Config{ProxyPort: 3000}, tokens: manager, logs: logs.NewRecorder(10)}
	result, err := app.configureCodex()
	if err != nil {
		t.Fatal(err)
	}
	if !result.AuthUpdated || !strings.Contains(result.Message, "API 登录模式") {
		t.Fatalf("expected API fallback result, got %#v", result)
	}

	authContent, err := os.ReadFile(filepath.Join(home, ".codex", "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	var authPayload map[string]any
	if err := json.Unmarshal(authContent, &authPayload); err != nil {
		t.Fatal(err)
	}
	if authPayload["OPENAI_API_KEY"] != codexLocalAPIKey {
		t.Fatalf("expected local API fallback key, got %#v", authPayload)
	}
	configContent, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(configContent), `forced_login_method = "api"`) {
		t.Fatalf("expected API login method, got:\n%s", configContent)
	}
}

func TestConfigureCodexCreatesAPIKeyFallbackWhenAccountPlanTypeIsMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	idTokenPayload, err := json.Marshal(map[string]any{
		"https://api.openai.com/profile": map[string]string{"email": "pool@example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	authBytes, err := json.Marshal(map[string]any{
		"tokens": map[string]string{
			"access_token": "pool-access-token",
			"account_id":   "pool-account",
			"id_token":     "header." + base64.RawURLEncoding.EncodeToString(idTokenPayload) + ".signature",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     string(authBytes),
	}); err != nil {
		t.Fatal(err)
	}

	app := &appServer{cfg: config.Config{ProxyPort: 3000}, tokens: manager, logs: logs.NewRecorder(10)}
	result, err := app.configureCodex()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Message, "API 登录模式") {
		t.Fatalf("expected API fallback result, got %#v", result)
	}

	authContent, err := os.ReadFile(filepath.Join(home, ".codex", "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	var authPayload map[string]any
	if err := json.Unmarshal(authContent, &authPayload); err != nil {
		t.Fatal(err)
	}
	if authPayload["OPENAI_API_KEY"] != codexLocalAPIKey {
		t.Fatalf("expected local API fallback key, got %#v", authPayload)
	}
}

func TestSelectCodexClientAuthPrefersImportedAccount(t *testing.T) {
	preferred := token.Token{
		ID:             "preferred",
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForMainTestWithCredentials(t, "preferred@example.com", "preferred-account", "preferred-access-token"),
		Status:         token.StatusActive,
	}
	other := token.Token{
		ID:             "other",
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForMainTestWithCredentials(t, "other@example.com", "other-account", "other-access-token"),
		Status:         token.StatusActive,
	}

	selected, ok := selectCodexClientAuth([]token.Token{other, preferred}, preferred.ID)
	if !ok || selected.ID != preferred.ID {
		t.Fatalf("expected imported account to be used for client login, got %#v", selected)
	}
}
