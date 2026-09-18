package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"omniproxy/internal/config"
	"omniproxy/internal/logs"
	"omniproxy/internal/storage"
	"omniproxy/internal/token"
)

// withTempCodexHome points HOME/USERPROFILE at a directory whose .codex folder
// holds the given profile files, so loadCodexModelProfiles reads them.
func withTempCodexHome(t *testing.T, profiles map[string]string) string {
	t.Helper()
	home := t.TempDir()
	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range profiles {
		if err := os.WriteFile(filepath.Join(codexDir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return codexDir
}

func TestServiceInjectsManagedModelsIntoCodexCatalog(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(codexCatalogFixture(t))
	}))
	defer upstream.Close()

	withTempCodexHome(t, map[string]string{
		"omniproxy-atria-dawn-preview.config.toml": "model = \"Atria-Dawn-Preview\"\nmodel_context_window = 256000\n",
	})

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(token.UpsertRequest{
		Name:           "coder@example.com",
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForServiceTest(t, "coder@example.com"),
	}); err != nil {
		t.Fatal(err)
	}

	service, err := NewService(config.Config{
		ProxyPort:       3000,
		ControlPort:     3890,
		CodexBaseURL:    upstream.URL + "/backend-api/codex",
		SwitchThreshold: 15,
		MaxRetries:      0,
	}, manager, logs.NewRecorder(10))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/codex/v1/models?client_version=0.155.0", nil)
	res := httptest.NewRecorder()
	service.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	body := res.Body.Bytes()
	if !strings.Contains(string(body), "Atria-Dawn-Preview") {
		t.Fatal("client did not receive the injected model")
	}
	if got := res.Header().Get("Content-Length"); got != strconv.Itoa(len(body)) {
		t.Fatalf("Content-Length = %q, want %d (body length)", got, len(body))
	}

	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("client body is not valid JSON: %v", err)
	}
	models := doc["models"].([]any)
	if len(models) != 2 {
		t.Fatalf("models = %d, want 2", len(models))
	}
	entry := models[1].(map[string]any)
	if entry["slug"] != "Atria-Dawn-Preview" || entry["display_name"] != "Atria Dawn Preview" {
		t.Fatalf("injected entry = %v", entry)
	}
	if entry["context_window"] != float64(256000) {
		t.Errorf("context_window = %v", entry["context_window"])
	}
}

func TestServiceLeavesCodexCatalogAloneWithoutProfiles(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(codexCatalogFixture(t))
	}))
	defer upstream.Close()

	withTempCodexHome(t, nil)

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(token.UpsertRequest{
		Name:           "coder@example.com",
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForServiceTest(t, "coder@example.com"),
	}); err != nil {
		t.Fatal(err)
	}

	service, err := NewService(config.Config{
		ProxyPort:       3000,
		ControlPort:     3890,
		CodexBaseURL:    upstream.URL + "/backend-api/codex",
		SwitchThreshold: 15,
		MaxRetries:      0,
	}, manager, logs.NewRecorder(10))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/codex/v1/models", nil)
	res := httptest.NewRecorder()
	service.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	if strings.Contains(res.Body.String(), "Atria") {
		t.Fatal("catalog was modified even though no managed profiles exist")
	}
}
