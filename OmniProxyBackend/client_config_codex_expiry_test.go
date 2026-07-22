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
	"time"
)

func TestConfigureCodexSkipsExpiredLocalAuthJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	expiredAccessToken := codexAccessTokenForMainTest(t, time.Now().Add(-time.Hour))
	expiredAuth := codexAuthJSONForMainTestWithCredentials(t, "coder@example.com", "same-account", expiredAccessToken)
	if err := os.WriteFile(filepath.Join(codexDir, "auth.json"), []byte(expiredAuth), 0o600); err != nil {
		t.Fatal(err)
	}

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	freshAccessToken := codexAccessTokenForMainTest(t, time.Now().Add(time.Hour))
	stored, err := manager.Add(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForMainTestWithCredentials(t, "coder@example.com", "same-account", freshAccessToken),
	})
	if err != nil {
		t.Fatal(err)
	}

	app := &appServer{cfg: config.Config{ProxyPort: 3000}, tokens: manager, logs: logs.NewRecorder(10)}
	result, err := app.configureCodex()
	if err != nil {
		t.Fatal(err)
	}
	if result.ImportedAuth || result.AuthAlreadyAdded || !result.AuthUpdated {
		t.Fatalf("expected expired local auth to be skipped in favor of the stored account, got %#v", result)
	}
	if items := manager.List(); len(items) != 1 {
		t.Fatalf("expected expired local auth not to be imported, got %d accounts", len(items))
	}
	unchanged, err := manager.Get(stored.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(unchanged.TokenValue, freshAccessToken) || strings.Contains(unchanged.TokenValue, expiredAccessToken) {
		t.Fatalf("expected stored valid auth to remain unchanged, got %s", unchanged.TokenValue)
	}
	authContent, err := os.ReadFile(filepath.Join(codexDir, "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(authContent), freshAccessToken) || strings.Contains(string(authContent), expiredAccessToken) {
		t.Fatalf("expected valid pool auth to be written locally, got %s", authContent)
	}
}

func TestConfigureCodexSkipsExpiredPreferredPoolAuth(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	expiredAccessToken := codexAccessTokenForMainTest(t, time.Now().Add(-time.Hour))
	expired, err := manager.Add(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForMainTestWithCredentials(t, "expired@example.com", "expired-account", expiredAccessToken),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.SetSelected(expired.ID, true); err != nil {
		t.Fatal(err)
	}
	freshAccessToken := codexAccessTokenForMainTest(t, time.Now().Add(time.Hour))
	if _, err := manager.Add(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForMainTestWithCredentials(t, "fresh@example.com", "fresh-account", freshAccessToken),
	}); err != nil {
		t.Fatal(err)
	}

	app := &appServer{cfg: config.Config{ProxyPort: 3000}, tokens: manager, logs: logs.NewRecorder(10)}
	result, err := app.configureCodex()
	if err != nil {
		t.Fatal(err)
	}
	if !result.AuthUpdated {
		t.Fatalf("expected a valid pool auth to be written locally, got %#v", result)
	}
	expiredAfter, err := manager.Get(expired.ID)
	if err != nil {
		t.Fatal(err)
	}
	if expiredAfter.Status != token.StatusInvalid {
		t.Fatalf("expected expired preferred auth to be marked invalid, got %#v", expiredAfter)
	}
	authContent, err := os.ReadFile(filepath.Join(home, ".codex", "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(authContent), freshAccessToken) || strings.Contains(string(authContent), expiredAccessToken) {
		t.Fatalf("expected fresh pool auth to be written locally, got %s", authContent)
	}
}

func codexAccessTokenForMainTest(t *testing.T, expiresAt time.Time) string {
	t.Helper()

	payload, err := json.Marshal(map[string]any{"exp": expiresAt.Unix()})
	if err != nil {
		t.Fatal(err)
	}
	return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}
