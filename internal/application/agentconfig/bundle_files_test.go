package agentconfig

import (
	"context"
	"os"
	"strings"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
)

func TestGatewayAgentsProtectSelectionAndExternalChanges(t *testing.T) {
	for _, client := range []string{"kimi-code", "pi", "factory-droid", "goose"} {
		t.Run(client, func(t *testing.T) {
			s, source := gatewayClientFixture(t)
			ctx := context.Background()
			path, _ := s.path(client)
			before, err := s.snapshotClient(client, path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Apply(ctx, Input{Client: client, Revision: before.revision, ModelIDs: []string{"deepseek-flash"}}); err != nil {
				t.Fatal(err)
			}
			source.entries = append(source.entries, catalog.Entry{ID: "excluded-new", Protocols: []string{"chat_completions"}})
			if err := s.RefreshCatalog(ctx); err != nil {
				t.Fatal(err)
			}
			models, err := s.ConfiguredModels(ctx, client)
			if err != nil || len(models.Models) != 1 || models.Models[0].ID != "deepseek-flash" {
				t.Fatal("refresh expanded a selected model list", err)
			}
			changed := path
			if isBundledClient(client) {
				changed = bundlePaths(client, path)[1]
			}
			body, err := os.ReadFile(changed)
			if err != nil || os.WriteFile(changed, append(body, '\n'), 0600) != nil {
				t.Fatal("external change fixture failed", err)
			}
			if _, err := s.ConfiguredModels(ctx, client); err == nil || err.Error() != "agent: config_changed" {
				t.Fatal("external file accepted as owned", err)
			}
			if err := s.RefreshCatalog(ctx); err == nil {
				t.Fatal("refresh overwrote external changes")
			}
			if _, err := s.Restore(ctx, client); err == nil {
				t.Fatal("restore overwrote external changes")
			}
		})
	}
}

func TestBundledClientsRollbackAllFilesOnDatabaseFailure(t *testing.T) {
	for _, client := range []string{"pi", "goose"} {
		for _, existing := range []bool{false, true} {
			t.Run(client+"/"+map[bool]string{false: "new", true: "existing"}[existing], func(t *testing.T) {
				s, _ := gatewayClientFixture(t)
				path, _ := s.path(client)
				if existing {
					writeGatewayFixture(t, path, gatewayAgentOriginal(client))
					if client == "pi" {
						writeGatewayFixture(t, bundlePaths(client, path)[1], `{"theme":"dark"}`)
					}
				}
				before, err := s.snapshotClient(client, path)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := s.db.Exec(`CREATE TRIGGER reject_bundle_version BEFORE UPDATE ON agent_backups BEGIN SELECT RAISE(FAIL,'private-fixture'); END;`); err != nil {
					t.Fatal(err)
				}
				if _, err := s.Apply(context.Background(), Input{Client: client, Revision: before.revision}); err == nil || strings.Contains(err.Error(), "private-fixture") {
					t.Fatal("failed transaction accepted or leaked", err)
				}
				after, err := s.snapshotClient(client, path)
				if err != nil || after.revision != before.revision {
					t.Fatal("transaction failure left partial config files", err)
				}
			})
		}
	}
}

func TestGatewayAgentConflictAndInvalidConfig(t *testing.T) {
	for _, client := range []string{"kimi-code", "pi", "factory-droid", "goose"} {
		t.Run(client, func(t *testing.T) {
			s, _ := gatewayClientFixture(t)
			path, _ := s.path(client)
			conflict := map[string]string{
				"kimi-code":     "[providers.omniproxy]\ntype='openai'\napi_key='private-fixture'\n",
				"pi":            `{"providers":{"omniproxy":{"apiKey":"private-fixture"}}}`,
				"factory-droid": `{"customModels":[{"displayName":"OmniProxy Existing","model":"existing"}]}`,
				"goose":         "providers:\n  omniproxy:\n    model: existing\n",
			}[client]
			writeGatewayFixture(t, path, conflict)
			before, err := s.snapshotClient(client, path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.PreviewPlan(context.Background(), Input{Client: client, Revision: before.revision}); err == nil || err.Error() != "agent: profile_conflict" {
				t.Fatal("unowned provider accepted", err)
			}
			invalid := `{"providers":[],"customModels":{}}`
			if client == "kimi-code" {
				invalid = "providers = ["
			}
			if client == "goose" {
				invalid = "providers: &loop {self: *loop}"
			}
			writeGatewayFixture(t, path, invalid)
			before, err = s.snapshotClient(client, path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.PreviewPlan(context.Background(), Input{Client: client, Revision: before.revision}); err == nil {
				t.Fatal("malformed config accepted")
			}
		})
	}
	for _, body := range []string{`null`, `{}`, `{"files":[{}]}`, `{"files":[{"body":"cHJpdmF0ZQ==","exists":false},{}]}`} {
		if _, err := decodeBundle([]byte(body)); err == nil {
			t.Fatal("malformed bundle accepted")
		}
	}
}
