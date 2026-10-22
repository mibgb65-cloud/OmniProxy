package modelcatalog

import "testing"

func TestDesktopEffortOnlyUsesConfirmedLevels(t *testing.T) {
	for _, test := range []struct {
		name     string
		metadata Metadata
		ceiling  string
		start    string
	}{
		{"river-defaults", Metadata{ReasoningEfforts: []Effort{{Value: "none"}, {Value: "high"}, {Value: "max"}}, DefaultEffort: "high"}, "max", "high"},
		{"max-default", Metadata{ReasoningEfforts: []Effort{{Value: "high"}, {Value: "max"}}, DefaultEffort: "max"}, "max", "max"},
		{"unknown-effort", Metadata{}, "", ""},
		{"off-only", Metadata{ReasoningEfforts: []Effort{{Value: "none"}}, DefaultEffort: "none"}, "", ""},
		{"unaccepted-default", Metadata{ReasoningEfforts: []Effort{{Value: "none"}, {Value: "medium"}}, DefaultEffort: "none"}, "medium", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			entry := Entry{ID: "model", Metadata: test.metadata}
			if got := DesktopMaxEffort(entry); got != test.ceiling {
				t.Fatalf("maxEffort = %q, want %q", got, test.ceiling)
			}
			if got := DesktopDefaultEffort(entry); got != test.start {
				t.Fatalf("defaultModelEffort = %q, want %q", got, test.start)
			}
		})
	}
}

func TestDesktopModelsDeclareConfirmedCeilingOnly(t *testing.T) {
	window := ClaudeLongContextWindow
	entries := []Entry{
		{ID: "deepseek-flash", Name: "DeepSeek", Metadata: Metadata{ContextWindow: &window, ReasoningEfforts: []Effort{{Value: "high"}, {Value: "max"}}, DefaultEffort: "high"}},
		{ID: "claude-sonnet-4-6", Name: "Sonnet"},
	}
	models := DesktopModels(entries, "deepseek-flash")
	if len(models) != 2 || models[0].MaxEffort != "max" {
		t.Fatalf("confirmed ceiling missing or moved: %+v", models)
	}
	// 未确认档位的模型保持未知，不能借用其他模型的能力。
	if models[1].MaxEffort != "" || models[1].Name != DesktopAlias("claude-sonnet-4-6") {
		t.Fatalf("unconfirmed model gained effort capability: %+v", models[1])
	}
}
