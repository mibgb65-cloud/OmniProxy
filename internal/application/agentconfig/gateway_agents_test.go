package agentconfig

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	catalog "omniproxy/internal/core/modelcatalog"
)

func gatewayAgentOriginal(client string) string {
	switch client {
	case "kimi-code":
		return "telemetry=false\ndefault_permission_mode='manual'\n[providers.existing]\ntype='openai'\nbase_url='https://existing.invalid/v1'\napi_key='private-fixture'\n"
	case "pi":
		return `{"providers":{"existing":{"apiKey":"private-fixture","models":[]}}}`
	case "factory-droid":
		return `{"autonomyMode":"low","customModels":[{"model":"existing","displayName":"Existing Model","apiKey":"private-fixture"}]}`
	default:
		return "# retained\nGOOSE_MODE: approve\nactive_provider: existing\nproviders:\n  existing:\n    enabled: true\n    model: original\nextensions: {}\n"
	}
}

func TestGatewayAgentsCompleteLifecycle(t *testing.T) {
	for _, client := range []string{"kimi-code", "pi", "factory-droid", "goose"} {
		t.Run(client, func(t *testing.T) {
			s, source := gatewayClientFixture(t)
			ctx := context.Background()
			path, _ := s.path(client)
			original := gatewayAgentOriginal(client)
			writeGatewayFixture(t, path, original)
			if client == "pi" {
				writeGatewayFixture(t, bundlePaths(client, path)[1], `{"theme":"dark","defaultProvider":"existing","defaultModel":"original","enabledModels":["existing/*"]}`)
			}
			session := filepath.Join(filepath.Dir(path), "untouched-auth-fixture.json")
			writeGatewayFixture(t, session, "private-auth-fixture")
			before, err := s.snapshotClient(client, path)
			if err != nil {
				t.Fatal(err)
			}
			defer clear(before.body)
			input := Input{Client: client, Revision: before.revision}
			plan, err := s.PreviewPlan(ctx, input)
			if err != nil || len(plan.Models) != 1 || plan.Mode != client+"-model-list" || strings.Contains(plan.Summary, "private") {
				t.Fatal("preview must contain only applicable public model information", err)
			}
			var count int
			if s.db.QueryRow(`SELECT COUNT(*) FROM agent_connections`).Scan(&count) != nil || count != 0 {
				t.Fatal("preview created a credential")
			}
			input.CatalogRevision = plan.CatalogRevision
			if _, err := s.Apply(ctx, input); err != nil {
				t.Fatal(err)
			}
			key, err := s.Key(ctx, client)
			if err != nil || !s.Authorize(ctx, string(key)) {
				t.Fatal("new client key not authorized", err)
			}
			defer clear(key)
			checkGatewayAgentWireConfig(t, client, path, key)
			source.entries = append(source.entries, catalog.Entry{ID: "new-chat", Name: "New", Protocols: []string{"chat_completions"}})
			if err := s.RefreshCatalog(ctx); err != nil {
				t.Fatal(err)
			}
			got, err := s.ConfiguredModels(ctx, client)
			if err != nil || len(got.Models) != 2 || got.DefaultModel != "deepseek-flash" {
				t.Fatal("catalog or default not refreshed correctly", err)
			}
			if _, err := s.Restore(ctx, client); err != nil {
				t.Fatal(err)
			}
			after, err := s.snapshotClient(client, path)
			if err != nil || after.revision != before.revision || s.Authorize(ctx, string(key)) {
				t.Fatal("restore did not preserve original files or disable key", err)
			}
			clear(after.body)
			untouched, err := os.ReadFile(session)
			if err != nil || string(untouched) != "private-auth-fixture" {
				t.Fatal("unrelated auth file changed")
			}
		})
	}
}

func checkGatewayAgentWireConfig(t *testing.T, client, path string, key []byte) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer clear(body)
	if !strings.Contains(string(body), "existing") || client != "goose" && !strings.Contains(string(body), "private-fixture") {
		t.Fatal("unrelated provider or settings lost")
	}
	var config map[string]any
	switch client {
	case "kimi-code":
		if toml.Unmarshal(body, &config) != nil || config["default_permission_mode"] != "manual" || config["telemetry"] != false || config["default_model"] != "omniproxy/deepseek-flash" {
			t.Fatal("Kimi defaults or permission settings incorrect")
		}
		provider := config["providers"].(map[string]any)["omniproxy"].(map[string]any)
		if provider["type"] != "openai" || provider["api_key"] != string(key) || provider["base_url"] != "http://127.0.0.1:3001/v1" {
			t.Fatal("Kimi protocol or auth incorrect")
		}
	case "factory-droid":
		if json.Unmarshal(body, &config) != nil || config["model"] != "custom:OmniProxy-Flash-1" || config["autonomyMode"] != "low" {
			t.Fatal("Droid default must use actual custom model reference")
		}
		models := config["customModels"].([]any)
		if models[0].(map[string]any)["model"] != "existing" || models[1].(map[string]any)["provider"] != "generic-chat-completion-api" {
			t.Fatal("Droid existing model order or protocol changed")
		}
	case "pi":
		if json.Unmarshal(body, &config) != nil {
			t.Fatal("Pi invalid model JSON")
		}
		provider := config["providers"].(map[string]any)["omniproxy"].(map[string]any)
		if provider["api"] != "openai-completions" || provider["authHeader"] != true || provider["apiKey"] != string(key) {
			t.Fatal("Pi transport or authorization missing")
		}
		settings, err := os.ReadFile(bundlePaths(client, path)[1])
		if err != nil || !strings.Contains(string(settings), "existing/*") || !strings.Contains(string(settings), "omniproxy/deepseek-flash") {
			t.Fatal("Pi scoped models hide the configured default", err)
		}
	case "goose":
		providerBody, err := os.ReadFile(bundlePaths(client, path)[1])
		if err != nil || json.Unmarshal(providerBody, &config) != nil {
			t.Fatal("Goose invalid provider file", err)
		}
		defer clear(providerBody)
		if config["base_url"] != "http://127.0.0.1:3001/v1/chat/completions" || config["dynamic_models"] != false || config["headers"].(map[string]any)["Authorization"] != "Bearer "+string(key) || strings.Contains(string(body), string(key)) {
			t.Fatal("Goose URL, selection boundary or isolated auth incorrect")
		}
	}
}

func TestLegacyPythonKimiUsesChatProtocol(t *testing.T) {
	s, _ := gatewayClientFixture(t)
	s.paths.KimiCodeLegacy = true
	before, err := s.snapshotClient("kimi-code", s.paths.KimiCode)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(context.Background(), Input{Client: "kimi-code", Revision: before.revision}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(s.paths.KimiCode)
	var config map[string]any
	if err != nil || toml.Unmarshal(body, &config) != nil {
		t.Fatal("legacy Kimi config invalid", err)
	}
	provider := config["providers"].(map[string]any)["omniproxy"].(map[string]any)
	model := config["models"].(map[string]any)["omniproxy/deepseek-flash"].(map[string]any)
	if provider["type"] != "openai_legacy" || model["capabilities"] != nil {
		t.Fatal("legacy Kimi must not use Responses or new capability enums")
	}
}

func TestGatewayAgentPreviewRejectsChangedConfigurationContext(t *testing.T) {
	for _, client := range []string{"kimi-code", "pi"} {
		t.Run(client, func(t *testing.T) {
			s, _ := gatewayClientFixture(t)
			path, _ := s.path(client)
			before, err := s.snapshotClient(client, path)
			if err != nil {
				t.Fatal(err)
			}
			input := Input{Client: client, Revision: before.revision}
			plan, err := s.PreviewPlan(context.Background(), input)
			if err != nil {
				t.Fatal(err)
			}
			input.CatalogRevision = plan.CatalogRevision
			if client == "kimi-code" {
				s.paths.KimiCodeLegacy = true
			} else {
				s.paths.Pi = filepath.Join(t.TempDir(), "models.json")
			}
			if _, err := s.Apply(context.Background(), input); err == nil || err.Error() != "agent: catalog_changed" {
				t.Fatal("stale preview accepted a changed configuration context", err)
			}
		})
	}
}
