package main

import (
	"omniproxy/internal/clientconfig"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteGeminiConfigPreservesExistingSettings(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(envPath, []byte("OTHER=keep\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte(`{"mcpServers":{"demo":{}},"security":{"auth":{"old":"keep"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := clientconfig.WriteGeminiEnv(envPath, "http://127.0.0.1:3000/gemini", "omniproxy-local", "gemini-3-pro-preview"); err != nil {
		t.Fatal(err)
	}
	if err := clientconfig.WriteGeminiSettings(settingsPath, "gemini-api-key"); err != nil {
		t.Fatal(err)
	}

	env, err := clientconfig.ReadEnvFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if env["OTHER"] != "keep" || env["GOOGLE_GEMINI_BASE_URL"] != "http://127.0.0.1:3000/gemini" || env["GEMINI_MODEL"] != "gemini-3-pro-preview" {
		t.Fatalf("unexpected gemini env: %#v", env)
	}
	settings, err := clientconfig.ReadJSONObject(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if settings["mcpServers"] == nil {
		t.Fatalf("expected existing settings to be preserved: %#v", settings)
	}
	security := settings["security"].(map[string]any)
	auth := security["auth"].(map[string]any)
	if auth["old"] != "keep" || auth["selectedType"] != "gemini-api-key" {
		t.Fatalf("unexpected auth settings: %#v", auth)
	}
	if _, err := os.Stat(envPath + ".omniproxy.bak"); err != nil {
		t.Fatalf("expected env backup: %v", err)
	}
	if _, err := os.Stat(settingsPath + ".omniproxy.bak"); err != nil {
		t.Fatalf("expected settings backup: %v", err)
	}
}

func TestWriteDeepSeekTUIConfigAddsOmniProxyDeepSeekProvider(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`provider = "openai"
default_text_model = "old-model"

[memory]
enabled = true

[providers.deepseek]
api_key = "deepseek-key"
other = "keep"

[providers.openai]
base_url = "https://api.openai.com/v1"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := clientconfig.WriteDeepSeekTUIConfig(path, "http://127.0.0.1:3000/opencode-router/v1", "omniproxy-local", "deepseek-v4-pro"); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, expected := range []string{
		`provider = "omniproxy"`,
		`default_text_model = "deepseek-v4-pro"`,
		`[providers.omniproxy]`,
		`api_key = "omniproxy-local"`,
		`base_url = "http://127.0.0.1:3000/opencode-router/v1"`,
		`model = "deepseek-v4-pro"`,
		`http_headers = { "X-OmniProxy-Client" = "DeepSeek-TUI" }`,
		`[providers.deepseek]`,
		`other = "keep"`,
		`[memory]`,
		`[providers.openai]`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in DeepSeek-TUI config:\n%s", expected, body)
		}
	}
	if strings.Count(body, `base_url = "http://127.0.0.1:3000/opencode-router/v1"`) != 1 {
		t.Fatalf("expected duplicate provider base_url to be removed:\n%s", body)
	}
	if _, err := os.Stat(path + ".omniproxy.bak"); err != nil {
		t.Fatalf("expected deepseek-tui backup: %v", err)
	}
}

func TestWriteDeepSeekTUIConfigKeepsExistingProviderHeaders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`[providers.deepseek]
http_headers = { "X-Existing" = "keep" }
`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := clientconfig.WriteDeepSeekTUIConfig(path, "http://127.0.0.1:3000/opencode-router/v1", "omniproxy-local", "deepseek-v4-pro"); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if !strings.Contains(body, `http_headers = { "X-Existing" = "keep" }`) {
		t.Fatalf("expected existing provider headers to be preserved:\n%s", body)
	}
	if strings.Count(body, `X-OmniProxy-Client`) != 1 {
		t.Fatalf("expected OmniProxy provider headers to be added once:\n%s", body)
	}
}

func TestWriteOpenCodeConfigAddsOmniProxyProviders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opencode.json")
	if err := os.WriteFile(path, []byte(`{"provider":{"existing":{"npm":"@ai-sdk/openai-compatible","models":{}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	openRouterModels := map[string]any{
		"openai/gpt-test": map[string]any{"name": "GPT Test"},
	}
	if err := writeOpenCodeConfig(path, "http://127.0.0.1:3000/opencode-router/v1", "http://127.0.0.1:3000/gemini", "http://127.0.0.1:3000/openrouter/v1", "http://127.0.0.1:3000/tokenrouter/v1", "http://127.0.0.1:3000/zo/v1", "http://127.0.0.1:3000/custom/v1", openRouterModels); err != nil {
		t.Fatal(err)
	}

	data, err := clientconfig.ReadJSONObject(path)
	if err != nil {
		t.Fatal(err)
	}
	providers := data["provider"].(map[string]any)
	for _, id := range []string{"existing", opencodeOmniProviderID} {
		if providers[id] == nil {
			t.Fatalf("expected provider %s in %#v", id, providers)
		}
	}
	for _, id := range []string{opencodeGeminiProviderID, opencodeOpenRouterProviderID, opencodeTokenRouterProviderID, opencodeZoProviderID, opencodeCustomProviderID} {
		if providers[id] != nil {
			t.Fatalf("expected OpenCode config to remove old OmniProxy provider %s: %#v", id, providers)
		}
	}
	routerProvider := providers[opencodeOmniProviderID].(map[string]any)
	options := routerProvider["options"].(map[string]any)
	if options["baseURL"] != "http://127.0.0.1:3000/opencode-router/v1" {
		t.Fatalf("unexpected router baseURL: %#v", options)
	}
	routerModels := routerProvider["models"].(map[string]any)
	if routerModels["gpt-6-astra"] == nil || routerModels["gpt-5.6-sol"] == nil || routerModels["gpt-5.6-terra"] == nil || routerModels["gpt-5.6-luna"] == nil || routerModels["openai/gpt-test"] == nil || routerModels["auto:balance"] == nil || routerModels["custom-model"] == nil {
		t.Fatalf("expected gateway models in %#v", routerModels)
	}
	if _, err := os.Stat(path + ".omniproxy.bak"); err != nil {
		t.Fatalf("expected opencode backup: %v", err)
	}
}

func TestWritePiModelsConfigAddsOmniProxyProviders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "models.json")
	if err := os.WriteFile(path, []byte(`{"providers":{"existing":{"api":"openai-completions","models":[]},"omniproxy-anthropic":{"api":"anthropic-messages"},"omniproxy-gemini":{"api":"google-generative-ai"},"omniproxy-openrouter":{"api":"openai-completions"},"omniproxy-custom":{"api":"openai-completions"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	openRouterModels := []map[string]any{
		{"id": "openai/gpt-test", "name": "GPT Test"},
	}
	if err := writePiModelsConfig(
		path,
		"http://127.0.0.1:3000/pi-router/v1",
		"http://127.0.0.1:3000/zo/v1",
		openRouterModels,
	); err != nil {
		t.Fatal(err)
	}

	data, err := clientconfig.ReadJSONObject(path)
	if err != nil {
		t.Fatal(err)
	}
	providers := data["providers"].(map[string]any)
	for _, id := range []string{"existing", piOmniProviderID} {
		if providers[id] == nil {
			t.Fatalf("expected provider %s in %#v", id, providers)
		}
	}
	for _, id := range []string{piAnthropicProviderID, piGeminiProviderID, piOpenRouterProviderID, piZoProviderID, piCustomProviderID} {
		if providers[id] != nil {
			t.Fatalf("expected Pi config to remove old OmniProxy provider %s: %#v", id, providers)
		}
	}
	routerProvider := providers[piOmniProviderID].(map[string]any)
	if routerProvider["api"] != "openai-completions" || routerProvider["baseUrl"] != "http://127.0.0.1:3000/pi-router/v1" {
		t.Fatalf("unexpected Pi router provider: %#v", routerProvider)
	}
	routerModels := routerProvider["models"].([]any)
	if len(routerModels) == 0 {
		t.Fatalf("expected Pi router models: %#v", routerProvider)
	}
	for _, id := range []string{"gpt-6-astra", "gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"} {
		if _, ok := piTestFindModel(routerModels, id); !ok {
			t.Fatalf("expected Pi router models to include %s: %#v", id, routerModels)
		}
	}
	routerCompat := routerProvider["compat"].(map[string]any)
	if routerCompat["supportsReasoningEffort"] != true {
		t.Fatalf("expected Pi router provider to support reasoning effort: %#v", routerCompat)
	}
	mimoModel, ok := piTestFindModel(routerModels, "mimo-v2.5-pro")
	if !ok {
		t.Fatalf("expected Pi router models to include MiMo Pro model: %#v", routerModels)
	}
	if mimoModel["reasoning"] != true {
		t.Fatalf("expected MiMo Pi model to be marked reasoning-capable: %#v", mimoModel)
	}
	mimoCompat := mimoModel["compat"].(map[string]any)
	if mimoCompat["supportsReasoningEffort"] != true {
		t.Fatalf("expected MiMo Pi model to support reasoning effort: %#v", mimoCompat)
	}
	if _, ok := piTestFindModel(routerModels, "mimo-v2.5-pro[1m]"); ok {
		t.Fatalf("Pi OpenAI-compatible router should not advertise MiMo 1M model: %#v", routerModels)
	}
	if _, ok := piTestFindModel(routerModels, "custom-model"); !ok {
		t.Fatalf("expected Pi router models to include custom gateway model: %#v", routerModels)
	}
	if _, ok := piTestFindModel(routerModels, "auto:balance"); !ok {
		t.Fatalf("expected Pi router models to include TokenRouter auto model: %#v", routerModels)
	}
	openRouterModel, ok := piTestFindModel(routerModels, "openai/gpt-test")
	if !ok {
		t.Fatalf("expected Pi router models to include OpenRouter models, got %#v", routerModels)
	}
	firstModel := openRouterModel
	if firstModel["id"] != "openai/gpt-test" {
		t.Fatalf("unexpected Pi openrouter model: %#v", firstModel)
	}
	if _, err := os.Stat(path + ".omniproxy.bak"); err != nil {
		t.Fatalf("expected Pi backup: %v", err)
	}
}

func piTestFindModel(models []any, id string) (map[string]any, bool) {
	for _, item := range models {
		model, ok := item.(map[string]any)
		if ok && model["id"] == id {
			return model, true
		}
	}
	return nil, false
}
