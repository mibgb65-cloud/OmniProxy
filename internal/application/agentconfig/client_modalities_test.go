package agentconfig

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
	catalog "omniproxy/internal/core/modelcatalog"
)

// River 已确认图片能力；本地 Agent 客户端必须在各自配置里声明，否则会在本地拦截附图。
// 未确认能力的模型必须保持纯文本，不能因为同目录其他模型支持图片而放宽。
func TestLocalAgentCatalogsPublishOnlyConfirmedImageModalities(t *testing.T) {
	type probe struct {
		client, model, field string
		read                 func(t *testing.T, path, model string) (bool, bool)
	}
	probes := []probe{
		{client: "harness-web", model: "vision-route", field: "pi-ai input", read: readHarnessImage},
		{client: "harness-desktop", model: "vision-route", field: "pi-ai input", read: readHarnessImage},
		{client: "opencode", model: "vision-route", field: "modalities.input", read: readOpenCodeImage},
		{client: "pi", model: "vision-route", field: "input", read: readPiImage},
		{client: "goose", model: "vision-route", field: "supports_vision", read: readGooseVision},
	}
	for _, p := range probes {
		for _, image := range []bool{true, false} {
			name := p.client + map[bool]string{true: "/image", false: "/text-only"}[image]
			t.Run(name, func(t *testing.T) {
				s, source := gatewayClientFixture(t)
				modalities := []string{"text"}
				if image {
					modalities = []string{"text", "image"}
				}
				source.entries = []catalog.Entry{
					{ID: p.model, Name: p.model, Protocols: []string{"chat_completions"}, Metadata: catalog.Metadata{InputModalities: modalities}},
					{ID: "text-model", Name: "text-model", Protocols: []string{"chat_completions"}, Metadata: catalog.Metadata{InputModalities: []string{"text"}}},
				}
				path, err := s.path(p.client)
				if err != nil {
					t.Fatal(err)
				}
				before, err := s.snapshotClient(p.client, path)
				if err != nil {
					t.Fatal(err)
				}
				input := Input{Client: p.client, Revision: before.revision}
				if _, err := s.Apply(context.Background(), input); err != nil {
					t.Fatal(err)
				}
				declared, found := p.read(t, path, p.model)
				if !found {
					t.Fatalf("%s: model missing from %s", p.client, p.field)
				}
				if declared != image {
					t.Fatalf("%s: %s image = %v, want %v", p.client, p.field, declared, image)
				}
				// 同目录的纯文本模型必须始终不声明图片。
				if other, found := p.read(t, path, "text-model"); !found || other {
					t.Fatalf("%s: text model gained image capability", p.client)
				}
			})
		}
	}
}

func readHarnessImage(t *testing.T, path, model string) (bool, bool) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc []map[string]any
	if yaml.Unmarshal(raw, &doc) != nil {
		t.Fatal("invalid harness yaml")
	}
	for _, plugin := range doc {
		config, _ := plugin["config"].(map[string]any)
		providers, _ := config["providers"].(map[string]any)
		route, _ := providers["omniproxy"].(map[string]any)
		models, _ := route["models"].([]any)
		for _, entry := range models {
			m, _ := entry.(map[string]any)
			if m["id"] == model {
				return containsString(m["input"], "image"), true
			}
		}
	}
	return false, false
}

func readOpenCodeImage(t *testing.T, path, model string) (bool, bool) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if json.Unmarshal(raw, &config) != nil {
		t.Fatal("invalid opencode json")
	}
	provider, _ := config["provider"].(map[string]any)
	route, _ := provider["omniproxy"].(map[string]any)
	models, _ := route["models"].(map[string]any)
	entry, ok := models[model].(map[string]any)
	if !ok {
		return false, false
	}
	modalities, _ := entry["modalities"].(map[string]any)
	return containsString(modalities["input"], "image"), true
}

func readPiImage(t *testing.T, path, model string) (bool, bool) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if json.Unmarshal(raw, &config) != nil {
		t.Fatal("invalid pi json")
	}
	providers, _ := config["providers"].(map[string]any)
	route, _ := providers["omniproxy"].(map[string]any)
	models, _ := route["models"].([]any)
	for _, entry := range models {
		m, _ := entry.(map[string]any)
		if m["id"] == model {
			return containsString(m["input"], "image"), true
		}
	}
	return false, false
}

// Goose 的模型目录写在同目录的 custom_providers/omniproxy.json，而非 config.yaml。
func readGooseVision(t *testing.T, path, model string) (bool, bool) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(path), "custom_providers", "omniproxy.json"))
	if err != nil {
		t.Fatal(err)
	}
	var provider map[string]any
	if json.Unmarshal(raw, &provider) != nil {
		t.Fatal("invalid goose provider json")
	}
	models, _ := provider["models"].([]any)
	for _, entry := range models {
		m, _ := entry.(map[string]any)
		if m["name"] == model {
			vision, _ := m["supports_vision"].(bool)
			return vision, true
		}
	}
	return false, false
}

func containsString(value any, want string) bool {
	list, _ := value.([]any)
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}
