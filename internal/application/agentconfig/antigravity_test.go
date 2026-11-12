package agentconfig

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
)

func TestAntigravityDesktopGatewayLifecycle(t *testing.T) {
	s := fixture(t)
	root := t.TempDir()
	s.paths.Antigravity = filepath.Join(root, ".gemini", "config", "config.json")
	s.paths.AntigravityLegacy = filepath.Join(root, ".antigravity", "settings.json")
	s.catalog = &changingCatalog{entries: []catalog.Entry{
		{ID: "gemini-3.8-flash", Name: "Gemini 3.8 Flash", ProviderKind: "google", Protocols: []string{"chat_completions"}},
		{ID: "claude-opus-4-6-thinking", Name: "Claude", ProviderKind: "google", Protocols: []string{"chat_completions"}},
		{ID: "gpt-default", Name: "GPT", ProviderKind: "openai", Protocols: []string{"chat_completions"}},
	}}
	primary := []byte(`{"gateway":{"enabled":false,"url":"https://old.invalid","customHeaders":{"X-Test":"1"}},"models":{"defaultModelId":"old","list":[{"modelId":"old"}],"other":true},"theme":"dark"}`)
	legacy := []byte(`{"models":{"list":[]},"extensions":{}}`)
	for path, body := range map[string][]byte{s.paths.Antigravity: primary, s.paths.AntigravityLegacy: legacy} {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	before, err := s.snapshotClient("antigravity", s.paths.Antigravity)
	if err != nil {
		t.Fatal(err)
	}
	input := Input{Client: "antigravity", Revision: before.revision}
	plan, err := s.PreviewPlan(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Models) != 1 || plan.Models[0].ID != "gemini-3.8-flash" || plan.DefaultModel != "gemini-3.8-flash" || plan.Mode != "antigravity-model-list" {
		t.Fatalf("catalog or default incorrect: %+v", plan)
	}
	input.CatalogRevision = plan.CatalogRevision
	if _, err := s.Apply(ctx, input); err != nil {
		t.Fatal(err)
	}
	key, err := s.Key(ctx, "antigravity")
	if err != nil || !s.Authorize(ctx, string(key)) {
		t.Fatal("antigravity key not issued", err)
	}
	defer clear(key)
	for _, path := range []string{s.paths.Antigravity, s.paths.AntigravityLegacy} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var config map[string]any
		if json.Unmarshal(body, &config) != nil {
			t.Fatal("invalid JSON written")
		}
		gateway := config["gateway"].(map[string]any)
		if gateway["enabled"] != true || gateway["url"] != "http://127.0.0.1:3001" || gateway["apiKey"] != string(key) || gateway["wireProtocol"] != "openai" {
			t.Fatalf("gateway block incorrect: %v", gateway)
		}
		models := config["models"].(map[string]any)
		if models["defaultModelId"] != "gemini-3.8-flash" || models["listSource"] != "http://127.0.0.1:3001/v1/models?format=antigravity" {
			t.Fatalf("models block incorrect: %v", models)
		}
		if _, exists := models["list"]; exists {
			t.Fatal("inline model list still hides the remote catalog")
		}
	}
	var primaryConfig map[string]any
	body, _ := os.ReadFile(s.paths.Antigravity)
	_ = json.Unmarshal(body, &primaryConfig)
	if primaryConfig["theme"] != "dark" || primaryConfig["gateway"].(map[string]any)["customHeaders"].(map[string]any)["X-Test"] != "1" {
		t.Fatal("unrelated user settings lost")
	}
	if got, err := s.ConfiguredModels(ctx, "antigravity"); err != nil || got.DefaultModel != "gemini-3.8-flash" {
		t.Fatal("configured model not reported", err)
	}
	if _, err := s.Restore(ctx, "antigravity"); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string][]byte{s.paths.Antigravity: primary, s.paths.AntigravityLegacy: legacy} {
		restored, _ := os.ReadFile(path)
		if string(restored) != string(want) {
			t.Fatalf("%s not restored exactly: %s", path, restored)
		}
	}
	if s.Authorize(ctx, string(key)) {
		t.Fatal("restored antigravity key still enabled")
	}
}

func TestAntigravityRequiresExistingSettings(t *testing.T) {
	s := fixture(t)
	root := t.TempDir()
	s.paths.Antigravity = filepath.Join(root, "config.json")
	s.paths.AntigravityLegacy = filepath.Join(root, "settings.json")
	if _, err := s.PreviewPlan(context.Background(), Input{Client: "antigravity", Revision: digest(nil, false)}); err == nil || !strings.Contains(err.Error(), "client_not_found") {
		t.Fatal("missing desktop settings accepted", err)
	}
}
