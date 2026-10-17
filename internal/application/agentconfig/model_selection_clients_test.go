package agentconfig

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	catalog "omniproxy/internal/core/modelcatalog"
)

func TestChatClientsKeepSelectedCatalogOnRefresh(t *testing.T) {
	for _, client := range []string{"opencode", "kilocode", "mimo-code", "qwen-code", "harness-web", "harness-desktop"} {
		t.Run(client, func(t *testing.T) {
			s, c := gatewayClientFixture(t)
			ctx := context.Background()
			c.entries = []catalog.Entry{{ID: "selected", Name: "Selected", Protocols: []string{"chat_completions"}}, {ID: "excluded", Protocols: []string{"chat_completions"}}}
			path, _ := s.path(client)
			before, err := s.snapshotClient(client, path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Apply(ctx, Input{Client: client, Revision: before.revision, ModelIDs: []string{"selected"}}); err != nil {
				t.Fatal(err)
			}
			c.entries = append(c.entries, catalog.Entry{ID: "new-model", Protocols: []string{"chat_completions"}})
			if err := s.RefreshCatalog(ctx); err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(path)
			if err != nil || !strings.Contains(string(body), "selected") || strings.Contains(string(body), "excluded") || strings.Contains(string(body), "new-model") {
				t.Fatal("selection not preserved", err)
			}
			if _, err := s.Restore(ctx, client); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCodexSelectionSwitchesRemoteCatalogToLocal(t *testing.T) {
	s, _ := catalogFixture(t)
	ctx := context.Background()
	s.SetCodexVersion("codex-cli 0.157.1")
	writeGatewayFixture(t, s.paths.Codex, "model_provider='existing'\n")
	body, err := os.ReadFile(s.paths.Codex)
	if err != nil {
		t.Fatal(err)
	}
	input := Input{Client: "codex", Revision: digest(body, true)}
	plan, err := s.PreviewPlan(ctx, input)
	if err != nil || plan.Mode != "remote-catalog" {
		t.Fatal("remote preview", err)
	}
	if _, err := s.Apply(ctx, input); err != nil {
		t.Fatal(err)
	}
	body, err = os.ReadFile(s.paths.Codex)
	if err != nil {
		t.Fatal(err)
	}
	input.Revision = digest(body, true)
	input.ModelIDs = []string{"coding-route"}
	input.CatalogRevision = plan.CatalogRevision
	if _, err := s.Apply(ctx, input); err != nil {
		t.Fatal(err)
	}
	body, err = os.ReadFile(s.paths.Codex)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if toml.Unmarshal(body, &config) != nil {
		t.Fatal("invalid TOML")
	}
	if config["model_catalog_json"] != s.catalogPath() || config["features"].(map[string]any)["api_key_model_discovery"] != false || strings.Contains(string(body), "model_catalog_url") {
		t.Fatal("remote discovery overrides selection")
	}
}

func TestFailedSelectionCommitPreservesPreviousSelection(t *testing.T) {
	s, _ := catalogFixture(t)
	ctx := context.Background()
	if _, err := s.Apply(ctx, Input{Client: "claude", Revision: digest(nil, false), ModelIDs: []string{"coding-route"}}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(s.paths.Claude)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`CREATE TRIGGER reject_selection_hash BEFORE UPDATE OF applied_hash ON agent_backups BEGIN SELECT RAISE(FAIL,'fixture'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(ctx, Input{Client: "claude", Revision: digest(body, true), ModelIDs: []string{"gpt-default"}}); err == nil {
		t.Fatal("failed transaction accepted")
	}
	after, err := os.ReadFile(s.paths.Claude)
	if err != nil || string(after) != string(body) {
		t.Fatal("file not rolled back", err)
	}
	var ids string
	if err := s.db.QueryRow(`SELECT model_ids FROM agent_backups WHERE client='claude'`).Scan(&ids); err != nil || ids != `["coding-route"]` {
		t.Fatal("selection not rolled back", err)
	}
}
