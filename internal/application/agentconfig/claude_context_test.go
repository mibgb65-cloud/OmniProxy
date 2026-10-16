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

func TestClaudeContextOptionsApplyRefreshAndRestore(t *testing.T) {
	for _, client := range []string{"claude", "claude-desktop"} {
		t.Run(client, func(t *testing.T) {
			s, source := catalogFixture(t)
			window := catalog.ClaudeLongContextWindow
			source.entries = []catalog.Entry{{ID: "deepseek-flash", ProviderKind: "deepseek", Metadata: catalog.Metadata{ContextWindow: &window}}, {ID: "unknown"}}
			ctx := context.Background()
			input := Input{Client: client, Revision: digest(nil, false), Model: "deepseek-flash[1m]"}
			plan, err := s.PreviewPlan(ctx, input)
			if err != nil || len(plan.Models) != 3 || plan.DefaultModel != input.Model {
				t.Fatal("long context selection unavailable", err)
			}
			if _, err := s.Apply(ctx, input); err != nil {
				t.Fatal(err)
			}
			path, err := s.path(client)
			if err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var config map[string]any
			if err := json.Unmarshal(body, &config); err != nil {
				t.Fatal(err)
			}
			if client == "claude" {
				env := config["env"].(map[string]any)
				for _, key := range []string{"ANTHROPIC_MODEL", "ANTHROPIC_DEFAULT_OPUS_MODEL", "ANTHROPIC_DEFAULT_SONNET_MODEL", "ANTHROPIC_DEFAULT_HAIKU_MODEL"} {
					if env[key] != input.Model {
						t.Fatal("CLI context suffix lost", key)
					}
				}
			} else {
				checkDesktopVariants(t, config, input.Model, 3)
				source.entries = append(source.entries, catalog.Entry{ID: "deepseek-v4-pro", ProviderKind: "deepseek", Metadata: catalog.Metadata{ContextWindow: &window}})
				if err := s.RefreshCatalog(ctx); err != nil {
					t.Fatal(err)
				}
				body, err = os.ReadFile(path)
				if err != nil || json.Unmarshal(body, &config) != nil {
					t.Fatal("cannot read refreshed catalog", err)
				}
				checkDesktopVariants(t, config, "", 5)
			}
			if _, err := s.Restore(ctx, client); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("new config remained after restore")
			}
		})
	}
}

func checkDesktopVariants(t *testing.T, config map[string]any, selected string, count int) {
	t.Helper()
	models := config["inferenceModels"].([]any)
	seen := map[string]string{}
	normalized := map[string]bool{}
	for _, item := range models {
		model := item.(map[string]any)
		seen[model["name"].(string)] = model["labelOverride"].(string)
		name := model["name"].(string)
		base := strings.TrimSuffix(name, catalog.ClaudeLongContextSuffix)
		if normalized[base] {
			t.Fatal("Desktop normalization collapses context options")
		}
		normalized[base] = true
		if strings.HasSuffix(name, catalog.ClaudeLongContextSuffix) {
			if model["supports1m"] != true || model["prefer1m"] != true {
				t.Fatal("Desktop long context flags missing")
			}
		} else if model["supports1m"] != nil || model["prefer1m"] != nil {
			t.Fatal("ordinary model context flags changed")
		}
	}
	if len(models) != count || seen[catalog.DesktopAlias("deepseek-flash")] != "deepseek-flash" || seen[catalog.DesktopAlias("deepseek-flash[1m]")] != "deepseek-flash[1m]" {
		t.Fatal("desktop variants incomplete or mislabeled")
	}
	if selected != "" && models[0].(map[string]any)["name"] != catalog.DesktopAlias(selected) {
		t.Fatal("selected default not first")
	}
}

func TestDeepSeekDefaultPreservesOrdinaryAndLongChoices(t *testing.T) {
	s, source := catalogFixture(t)
	window := catalog.ClaudeLongContextWindow
	source.entries = []catalog.Entry{{ID: "gpt-default", Metadata: catalog.Metadata{IsDefault: true}}, {ID: "deepseek-flash", ProviderKind: "deepseek", Metadata: catalog.Metadata{ContextWindow: &window}}}
	body := []byte(`{"env":{"ANTHROPIC_MODEL":"deepseek-flash"},"permissions":{"allow":[]}}`)
	if err := os.MkdirAll(filepath.Dir(s.paths.Claude), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.paths.Claude, body, 0600); err != nil {
		t.Fatal(err)
	}
	for _, selected := range []string{"", "deepseek-flash", "deepseek-flash[1m]"} {
		plan, err := s.PreviewPlan(context.Background(), Input{Client: "claude", Revision: digest(body, true), Model: selected})
		expected := selected
		if expected == "" {
			expected = "deepseek-flash"
		}
		if err != nil || plan.DefaultModel != expected || len(plan.Models) != 3 {
			t.Fatal("DeepSeek default choice changed", plan.DefaultModel, err)
		}

	}
	plan, err := s.PreviewPlan(context.Background(), Input{Client: "codex", Revision: digest(nil, false), Model: "deepseek-flash"})
	if err != nil || plan.DefaultModel != "deepseek-flash" {
		t.Fatal("Claude preference changed Codex defaults", err)
	}
}
