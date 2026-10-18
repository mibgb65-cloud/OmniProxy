package agentconfig

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
)

func TestConfiguredModelsReadActualManagedMenus(t *testing.T) {
	for _, client := range Clients() {
		t.Run(client, func(t *testing.T) {
			s, c := gatewayClientFixture(t)
			c.entries = []catalog.Entry{{ID: "selected", Name: "Selected", Protocols: []string{"chat_completions", "messages", "responses"}}, {ID: "excluded", Name: "Excluded", Protocols: []string{"chat_completions", "messages", "responses"}}}
			ctx := context.Background()
			before, err := s.ConfiguredModels(ctx, client)
			if err != nil || len(before.Models) != 0 {
				t.Fatal("unmanaged models reported", err)
			}
			path, _ := s.path(client)
			file, err := s.snapshotClient(client, path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Apply(ctx, Input{Client: client, Revision: file.revision, ModelIDs: []string{"selected"}}); err != nil {
				t.Fatal(err)
			}
			// 目录新增尚未刷新到文件时，不能被展示为已经接入。
			c.entries = append(c.entries, catalog.Entry{ID: "new-model", Name: "New", Protocols: []string{"chat_completions", "messages", "responses"}})
			got, err := s.ConfiguredModels(ctx, client)
			expectedName := "Selected"
			if client == "goose" {
				// Goose 自定义 provider 的 ModelInfo 仅提供模型 name（API ID），没有独立展示名称。
				expectedName = "selected"
			}
			if err != nil || len(got.Models) != 1 || got.Models[0].ID != "selected" || got.Models[0].Name != expectedName {
				t.Fatal("actual selection not read", err, got)
			}
			if !isHarness(client) && got.DefaultModel != "selected" {
				t.Fatal("default not read")
			}
			encoded, err := json.Marshal(got)
			if err != nil || strings.Contains(string(encoded), "opk_") || strings.Contains(string(encoded), "apiKey") {
				t.Fatal("credential exposed")
			}
			if _, err := s.Restore(ctx, client); err != nil {
				t.Fatal(err)
			}
			got, err = s.ConfiguredModels(ctx, client)
			if err != nil || len(got.Models) != 0 {
				t.Fatal("restored models remain", err)
			}
		})
	}
}

func TestConfiguredModelsProtectExternalEditsAndCatalogFiles(t *testing.T) {
	s, _ := catalogFixture(t)
	ctx := context.Background()
	if _, err := s.ConfiguredModels(ctx, "../codex"); err == nil {
		t.Fatal("invalid client accepted")
	}
	if _, err := s.Apply(ctx, Input{Client: "codex", Revision: digest(nil, false)}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.catalogPath(), []byte("private-invalid-catalog"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConfiguredModels(ctx, "codex"); err == nil || strings.Contains(err.Error(), "private-invalid") {
		t.Fatal("external catalog accepted", err)
	}
	if _, err := s.Apply(ctx, Input{Client: "claude", Revision: digest(nil, false)}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.paths.Claude, []byte(`{"env":{"ANTHROPIC_AUTH_TOKEN":"private-edited-key"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConfiguredModels(ctx, "claude"); err == nil || err.Error() != "agent: config_changed" {
		t.Fatal("external edit accepted", err)
	}
}

func TestConfiguredModelsDoNotInventUnwrittenModelsForAllSelection(t *testing.T) {
	for _, client := range Clients() {
		t.Run(client, func(t *testing.T) {
			s, source := gatewayClientFixture(t)
			source.entries = []catalog.Entry{{ID: "original", Name: "Original"}}
			ctx := context.Background()
			path, _ := s.path(client)
			before, err := s.snapshotClient(client, path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Apply(ctx, Input{Client: client, Revision: before.revision}); err != nil {
				t.Fatal(err)
			}
			source.entries = append(source.entries, catalog.Entry{ID: "not-written", Name: "Not written"})
			got, err := s.ConfiguredModels(ctx, client)
			if err != nil || len(got.Models) != 1 || got.Models[0].ID != "original" {
				t.Fatal("unwritten model reported", err)
			}
		})
	}
}
