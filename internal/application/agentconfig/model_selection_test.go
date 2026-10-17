package agentconfig

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
)

func TestModelSelectionValidatesBeforeWriting(t *testing.T) {
	for _, test := range []struct {
		name  string
		ids   []string
		model string
		code  string
	}{
		{"empty", []string{}, "", "invalid_model_selection"},
		{"unknown", []string{"missing"}, "", "invalid_model_selection"},
		{"duplicate", []string{"coding-route", "coding-route"}, "", "invalid_model_selection"},
		{"blank", []string{""}, "", "invalid_model_selection"},
		{"long", []string{strings.Repeat("a", 129)}, "", "invalid_model_selection"},
		{"too many", make([]string, maxSelectedModels+1), "", "invalid_model_selection"},
		{"excluded default", []string{"coding-route"}, "gpt-default", "invalid_model"},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, _ := catalogFixture(t)
			input := Input{Client: "codex", Revision: digest(nil, false), ModelIDs: test.ids, Model: test.model}
			if _, err := s.Apply(context.Background(), input); err == nil || err.Error() != "agent: "+test.code {
				t.Fatal("invalid selection accepted", err)
			}
			if _, err := os.Stat(s.paths.Codex); !os.IsNotExist(err) {
				t.Fatal("invalid selection wrote configuration")
			}
		})
	}
}

func TestModelSelectionPreservesSourceRevisionAndRejectsStaleCatalog(t *testing.T) {
	s, c := catalogFixture(t)
	ctx := context.Background()
	input := Input{Client: "codex", Revision: digest(nil, false)}
	plan, err := s.PreviewPlan(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	input.ModelIDs = []string{"coding-route"}
	input.Model = "coding-route"
	input.CatalogRevision = plan.CatalogRevision
	selected, err := s.PreviewPlan(ctx, input)
	if err != nil || len(selected.Models) != 1 || selected.DefaultModel != "coding-route" {
		t.Fatal("selection cannot use preview snapshot", err)
	}
	c.entries[0].Name = "Updated excluded model"
	if _, err := s.Apply(ctx, input); err == nil || err.Error() != "agent: catalog_changed" {
		t.Fatal("stale full catalog accepted", err)
	}
}

func TestSelectedCatalogSurvivesRefreshAndRestore(t *testing.T) {
	for _, client := range []string{"codex", "claude", "claude-desktop"} {
		t.Run(client, func(t *testing.T) {
			s, c := catalogFixture(t)
			ctx := context.Background()
			input := Input{Client: client, Revision: digest(nil, false), ModelIDs: []string{"coding-route"}}
			plan, err := s.PreviewPlan(ctx, input)
			if err != nil || len(plan.Models) != 1 || plan.DefaultModel != "coding-route" {
				t.Fatal("preview", err)
			}
			input.CatalogRevision = plan.CatalogRevision
			if _, err := s.Apply(ctx, input); err != nil {
				t.Fatal(err)
			}
			c.entries = append(c.entries, catalog.Entry{ID: "new-unselected", Name: "New unselected"})
			c.entries[1].Name = "Updated selected"
			if err := s.RefreshCatalog(ctx); err != nil {
				t.Fatal(err)
			}
			path, _ := s.path(client)
			if client == "codex" {
				path = s.catalogPath()
			}
			body, err := os.ReadFile(path)
			if err != nil || !strings.Contains(string(body), "Updated selected") || strings.Contains(string(body), "new-unselected") || strings.Contains(string(body), "gpt-default") {
				t.Fatal("refresh ignored saved selection", err)
			}
			// 服务重建后仍从数据库加载选择；全部失效不擅自改用其他模型。
			s = New(s.db, s.store, s.paths, s.port, c)
			c.entries = c.entries[:1]
			if err := s.RefreshCatalog(ctx); err == nil || !strings.Contains(err.Error(), "no_models") {
				t.Fatal("missing selection fell back", err)
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != string(body) {
				t.Fatal("failed refresh modified file", err)
			}
			if _, err := s.Restore(ctx, client); err != nil {
				t.Fatal(err)
			}
			var count int
			if err := s.db.QueryRow(`SELECT COUNT(*) FROM agent_backups WHERE client=?`, client).Scan(&count); err != nil || count != 0 {
				t.Fatal("restore retained selection", err)
			}
		})
	}
}

func TestReapplyingAllModelsClearsSavedSelection(t *testing.T) {
	s, c := catalogFixture(t)
	ctx := context.Background()
	if _, err := s.Apply(ctx, Input{Client: "claude", Revision: digest(nil, false), ModelIDs: []string{"coding-route"}}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(s.paths.Claude)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(ctx, Input{Client: "claude", Revision: digest(body, true)}); err != nil {
		t.Fatal(err)
	}
	c.entries = append(c.entries, catalog.Entry{ID: "new-model", Name: "New model"})
	if err := s.RefreshCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	body, err = os.ReadFile(s.paths.Claude)
	if err != nil || !strings.Contains(string(body), "new-model") {
		t.Fatal("all models still restricted", err)
	}
}

func TestSelectedClaudeDefaultFallsBackWithinSelection(t *testing.T) {
	s, c := catalogFixture(t)
	ctx := context.Background()
	input := Input{Client: "claude", Revision: digest(nil, false), ModelIDs: []string{"coding-route", "gpt-default"}, Model: "coding-route"}
	if _, err := s.Apply(ctx, input); err != nil {
		t.Fatal(err)
	}
	c.entries = []catalog.Entry{{ID: "gpt-default"}, {ID: "unselected"}}
	if err := s.RefreshCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(s.paths.Claude)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Env map[string]string `json:"env"`
	}
	if err := json.Unmarshal(body, &config); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"ANTHROPIC_MODEL", "ANTHROPIC_DEFAULT_OPUS_MODEL", "ANTHROPIC_DEFAULT_SONNET_MODEL", "ANTHROPIC_DEFAULT_HAIKU_MODEL"} {
		if config.Env[key] != "gpt-default" {
			t.Fatal("default outside available selection")
		}
	}
}
