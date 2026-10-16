package agentconfig

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/pelletier/go-toml/v2"
	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/provider/river"
)

func TestRiverManagedMenusUpgradeNamesAndPreserveSelectedModel(t *testing.T) {
	for _, client := range []string{"claude", "claude-desktop", "codex"} {
		t.Run(client, func(t *testing.T) {
			s, source := catalogFixture(t)
			window := int64(1048576)
			source.entries = []catalog.Entry{{ID: river.NemotronVL, Name: river.NemotronVL, Metadata: catalog.Metadata{ContextWindow: &window}}}
			if _, err := s.Apply(context.Background(), Input{Client: client, Revision: digest(nil, false), Model: river.NemotronVL}); err != nil {
				t.Fatal(err)
			}
			source.entries = []catalog.Entry{{ID: "other-first", Metadata: catalog.Metadata{IsDefault: true}}, {ID: "NVIDIA-Nemotron-3.5", Name: "NVIDIA-Nemotron-3.5", Metadata: catalog.Metadata{ContextWindow: &window}}}
			if err := s.RefreshCatalog(context.Background()); err != nil {
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
			if client == "codex" {
				if toml.Unmarshal(body, &config) != nil || config["model"] != "NVIDIA-Nemotron-3.5" {
					t.Fatal("Codex default switched to unrelated model")
				}
			} else {
				if json.Unmarshal(body, &config) != nil {
					t.Fatal("invalid client config")
				}
				if client == "claude" {
					if config["env"].(map[string]any)["ANTHROPIC_MODEL"] != "NVIDIA-Nemotron-3.5" {
						t.Fatal("CLI legacy default not upgraded")
					}
					checkClaudePicker(t, path, mustClientEntries(t, s, client))
				} else {
					models := config["inferenceModels"].([]any)
					if models[0].(map[string]any)["labelOverride"] != "NVIDIA-Nemotron-3.5" {
						t.Fatal("Desktop default switched")
					}
				}
			}
			for _, raw := range river.Models() {
				entries, _ := catalog.ClaudeEntries([]catalog.Entry{{ID: river.PublicModel(raw), Name: river.ModelName(raw), Metadata: catalog.Metadata{ContextWindow: &window}}})
				for _, suffix := range []string{"", "[1m]"} {
					for _, old := range []string{raw + suffix, catalog.DesktopAlias(raw + suffix)} {
						if selected := upgradeClaudeDefault(entries, old); selected != river.PublicModel(raw)+suffix {
							t.Fatal("legacy selected variant changed", old, selected)
						}
					}
				}
			}
		})
	}
}

func mustClientEntries(t *testing.T, s *Service, client string) []catalog.Entry {
	t.Helper()
	entries, err := s.clientEntries(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}
