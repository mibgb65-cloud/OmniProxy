package agentconfig

import (
	"context"
	"os"
	"slices"
	"strings"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
)

func googleAgentEntries() []catalog.Entry {
	entries := []catalog.Entry{}
	for _, id := range []string{"gemini-test", "claude-opus-4-6-thinking", "gpt-oss-120b-medium"} {
		entries = append(entries, catalog.Entry{ID: id, Name: id, Protocols: []string{"chat_completions", "responses", "messages"}, ProviderKind: "google"})
	}
	return entries
}

func TestGoogleModelsRefreshAllManagedAgentsWithoutExpandingExplicitSelections(t *testing.T) {
	for _, client := range Clients() {
		for _, limited := range []bool{false, true} {
			name := client + "/all"
			if limited {
				name = client + "/selected"
			}
			t.Run(name, func(t *testing.T) {
				s, source := gatewayClientFixture(t)
				source.entries = []catalog.Entry{{ID: "existing", Name: "Existing", Protocols: []string{"chat_completions", "responses", "messages"}}}
				ctx := context.Background()
				path, err := s.path(client)
				if err != nil {
					t.Fatal(err)
				}
				file, err := s.snapshotClient(client, path)
				if err != nil {
					t.Fatal(err)
				}
				input := Input{Client: client, Revision: file.revision, Model: "existing"}
				if limited {
					input.ModelIDs = []string{"existing"}
				}
				if _, err := s.Apply(ctx, input); err != nil {
					t.Fatal(err)
				}
				source.entries = append(source.entries, googleAgentEntries()...)
				if err := s.RefreshCatalog(ctx); err != nil {
					t.Fatal(err)
				}
				got, err := s.ConfiguredModels(ctx, client)
				if err != nil {
					t.Fatal(err)
				}
				expected := 4
				if limited {
					expected = 1
				}
				if len(got.Models) != expected {
					t.Fatalf("实际菜单数量错误: got=%d want=%d", len(got.Models), expected)
				}
				if !isHarness(client) && got.DefaultModel != "existing" {
					t.Fatal("自动同步改变了仍可用的默认模型")
				}
				for _, model := range googleAgentEntries() {
					found := slices.ContainsFunc(got.Models, func(item ModelSummary) bool { return item.ID == model.ID })
					if found == limited {
						t.Fatal("未更新实际目录，或擅自扩张用户限选")
					}
				}
			})
		}
	}
}

func TestGoogleAutoRefreshDoesNotTakeOverOrOverwriteAgentFiles(t *testing.T) {
	for _, client := range Clients() {
		t.Run(client, func(t *testing.T) {
			s, source := gatewayClientFixture(t)
			source.entries = googleAgentEntries()
			ctx := context.Background()
			path, err := s.path(client)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.RefreshCatalog(ctx); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("自动接管了尚未确认的客户端")
			}
			file, err := s.snapshotClient(client, path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Apply(ctx, Input{Client: client, Revision: file.revision}); err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			external := append(body, '\n')
			if err := os.WriteFile(path, external, 0600); err != nil {
				t.Fatal(err)
			}
			if err := s.RefreshCatalog(ctx); err == nil || !strings.Contains(err.Error(), "config_changed") {
				t.Fatal("外部配置变更未被保护", err)
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != string(external) {
				t.Fatal("覆盖了外部修改的配置", err)
			}
		})
	}
}
