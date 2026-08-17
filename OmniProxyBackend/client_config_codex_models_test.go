package main

import (
	"errors"
	"omniproxy/internal/config"
	"omniproxy/internal/logs"
	"omniproxy/internal/storage"
	"omniproxy/internal/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigureCodexWritesSelectedModelProfiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	legacyConfig := strings.Join([]string{
		`model_provider = "deepseek"`,
		`model_catalog_json = "~/.codex/models.json"`,
		``,
		`[model_providers.OpenAI]`,
		`base_url = "http://127.0.0.1:3000/codex/v1"`,
		``,
		`[model_providers.deepseek]`,
		`base_url = "https://api.deepseek.com"`,
		`experimental_bearer_token = "sk-fake-deepseek"`,
	}, "\n")
	if err := os.WriteFile(filepath.Join(codexDir, "config.toml"), []byte(legacyConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForMainTest(t, "coder@example.com"),
	}); err != nil {
		t.Fatal(err)
	}
	app := &appServer{
		cfg:    config.Config{ProxyPort: 3000},
		tokens: manager,
		logs:   logs.NewRecorder(10),
	}

	result, err := app.configureCodex(codexConfigureRequest{
		Models: []string{" gpt-5.6-sol ", "gpt-5.6-terra", "gpt-5.6-luna", "deepseek-v4-pro[1m]", "ignored"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Model != "gpt-5.6-sol" || len(result.Models) != maxCodexModels {
		t.Fatalf("expected four selected Codex models with gpt-5.6-sol primary, got %#v", result)
	}
	if result.Models[3] != "deepseek-v4-pro" {
		t.Fatalf("expected Codex 1M suffix to normalize for upstream routing, got %#v", result.Models)
	}

	content, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, expected := range []string{
		`model = "gpt-5.6-sol"`,
		`review_model = "gpt-5.6-sol"`,
		`model_context_window = 1050000`,
		`model_provider = "openai"`,
		`openai_base_url = "http://127.0.0.1:3000/codex/v1"`,
		`forced_login_method = "chatgpt"`,
		`cli_auth_credentials_store = "file"`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected config to contain %q, got:\n%s", expected, text)
		}
	}
	if strings.Contains(text, "[model_providers.OpenAI]") {
		t.Fatalf("Codex gateway config must keep the built-in OpenAI provider so official and gateway conversations share history:\n%s", text)
	}
	if strings.Contains(text, "model_catalog_json") || strings.Contains(text, "[model_providers.deepseek]") || strings.Contains(text, "sk-fake-deepseek") {
		t.Fatalf("Codex gateway config must remove the DeepSeek-only catalog and direct provider:\n%s", text)
	}

	profilePath := filepath.Join(home, ".codex", "omniproxy-gpt-5-6-luna.config.toml")
	profile, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(profile), `model = "gpt-5.6-luna"`) ||
		!strings.Contains(string(profile), `review_model = "gpt-5.6-luna"`) ||
		!strings.Contains(string(profile), `model_provider = "openai"`) ||
		!strings.Contains(string(profile), `model_reasoning_effort = "xhigh"`) ||
		!strings.Contains(string(profile), `model_context_window = 400000`) ||
		!strings.Contains(string(profile), `model_auto_compact_token_limit = 360000`) {
		t.Fatalf("unexpected profile content:\n%s", string(profile))
	}
	deepSeekProfilePath := filepath.Join(home, ".codex", "omniproxy-deepseek-v4-pro.config.toml")
	deepSeekProfile, err := os.ReadFile(deepSeekProfilePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`model_provider = "omniproxy_deepseek"`,
		`model_reasoning_effort = "high"`,
		`model_context_window = 1048576`,
		`model_auto_compact_token_limit = 996147`,
	} {
		if !strings.Contains(string(deepSeekProfile), expected) {
			t.Fatalf("expected DeepSeek profile to contain %q, got:\n%s", expected, string(deepSeekProfile))
		}
	}
	for _, expected := range []string{
		`[model_providers.omniproxy_deepseek]`,
		`base_url = "http://127.0.0.1:3000/codex/v1"`,
		`wire_api = "responses"`,
		`requires_openai_auth = true`,
		`supports_websockets = false`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected config to contain %q, got:\n%s", expected, text)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".codex", "omniproxy-ignored.config.toml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected fifth Codex model profile to be skipped, got %v", err)
	}
}

func TestConfigureCodexUsesDeepSeekProviderForDeepSeekDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	app := &appServer{
		cfg:    config.Config{ProxyPort: 3000},
		tokens: manager,
		logs:   logs.NewRecorder(10),
	}

	if _, err := app.configureCodex(codexConfigureRequest{Models: []string{"deepseek-v4-flash", "gpt-5.6-sol"}}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, expected := range []string{
		`model = "deepseek-v4-flash"`,
		`model_provider = "omniproxy_deepseek"`,
		`model_reasoning_effort = "high"`,
		`model_context_window = 1048576`,
		`model_auto_compact_token_limit = 996147`,
		`supports_websockets = false`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected DeepSeek default config to contain %q, got:\n%s", expected, text)
		}
	}
}
