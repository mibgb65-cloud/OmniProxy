package agentconfig

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
)

func TestClaudeContextOptionsApplyRefreshAndRestore(t *testing.T) {
	for _, client := range []string{"claude", "claude-desktop"} {
		t.Run(client, func(t *testing.T) {
			s, source := catalogFixture(t)
			window := catalog.ClaudeLongContextWindow
			source.entries = []catalog.Entry{{ID: "deepseek-flash", Metadata: catalog.Metadata{ContextWindow: &window}}, {ID: "unknown"}}
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
				source.entries = append(source.entries, catalog.Entry{ID: "deepseek-v4-pro", Metadata: catalog.Metadata{ContextWindow: &window}})
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
	for _, item := range models {
		model := item.(map[string]any)
		seen[model["name"].(string)] = model["labelOverride"].(string)
	}
	if len(models) != count || seen[catalog.DesktopAlias("deepseek-flash")] != "deepseek-flash" || seen[catalog.DesktopAlias("deepseek-flash[1m]")] != "deepseek-flash[1m]" {
		t.Fatal("desktop variants incomplete or mislabeled")
	}
	if selected != "" && models[0].(map[string]any)["name"] != catalog.DesktopAlias(selected) {
		t.Fatal("selected default not first")
	}
}
