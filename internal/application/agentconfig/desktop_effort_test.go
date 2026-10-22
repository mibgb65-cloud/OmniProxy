package agentconfig

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
)

// TestDesktopConfigDeclaresConfirmedEffort 固定 Desktop 目录的两处档位声明：
// 每个模型的 maxEffort 只来自已确认枚举，顶层 defaultModelEffort 只来自已确认默认档位。
func TestDesktopConfigDeclaresConfirmedEffort(t *testing.T) {
	s, source := catalogFixture(t)
	source.entries = []catalog.Entry{
		{ID: "gpt-default", Name: "Default GPT", Metadata: catalog.Metadata{IsDefault: true}},
		{
			ID:           "river-effort",
			Name:         "River Effort",
			ProviderKind: "river",
			Metadata:     catalog.Metadata{ReasoningEfforts: []catalog.Effort{{Value: "none"}, {Value: "high"}, {Value: "max"}}, DefaultEffort: "high"},
		},
	}
	ctx := context.Background()
	if _, err := s.Apply(ctx, Input{Client: "claude-desktop", Revision: digest(nil, false), Model: "river-effort"}); err != nil {
		t.Fatal(err)
	}
	config := readDesktopConfig(t, s)
	if config.defaultEffort != "high" {
		t.Fatalf("defaultModelEffort = %q, want upstream default", config.defaultEffort)
	}
	if model := config.byName[catalog.DesktopAlias("river-effort")]; model.maxEffort != "max" {
		t.Fatalf("river maxEffort = %q, want max", model.maxEffort)
	}
	if model := config.byName[catalog.DesktopAlias("gpt-default")]; model.maxEffort != "" {
		t.Fatalf("unconfirmed model declared effort ceiling %q", model.maxEffort)
	}

	// 默认模型改为未确认档位的模型时必须清除旧值，不能继续套用上一个模型的档位。
	applied, err := os.ReadFile(s.paths.ClaudeDesktop)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(ctx, Input{Client: "claude-desktop", Revision: digest(applied, true), Model: "gpt-default"}); err != nil {
		t.Fatal(err)
	}
	if config := readDesktopConfig(t, s); config.defaultEffort != "" {
		t.Fatalf("stale defaultModelEffort kept: %q", config.defaultEffort)
	}
}

type desktopEffortConfig struct {
	defaultEffort string
	byName        map[string]struct{ maxEffort string }
}

func readDesktopConfig(t *testing.T, s *Service) desktopEffortConfig {
	t.Helper()
	body, err := os.ReadFile(s.paths.ClaudeDesktop)
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		DefaultEffort string `json:"defaultModelEffort"`
		Models        []struct {
			Name      string `json:"name"`
			MaxEffort string `json:"maxEffort"`
		} `json:"inferenceModels"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	config := desktopEffortConfig{defaultEffort: raw.DefaultEffort, byName: map[string]struct{ maxEffort string }{}}
	for _, model := range raw.Models {
		config.byName[model.Name] = struct{ maxEffort string }{maxEffort: model.MaxEffort}
	}
	return config
}
