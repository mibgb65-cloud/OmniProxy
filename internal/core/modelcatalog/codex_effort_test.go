package modelcatalog

import (
	"encoding/json"
	"testing"
)

func TestCodexEffortsMapUpstreamVocabulary(t *testing.T) {
	cases := []struct {
		name     string
		efforts  []Effort
		expected []string
	}{
		{"river-deepseek", []Effort{{Value: "none"}, {Value: "high"}, {Value: "max"}}, []string{"minimal", "high", "xhigh"}},
		{"river-glm", []Effort{{Value: "low"}, {Value: "high"}, {Value: "max"}}, []string{"low", "high", "xhigh"}},
		{"minimax", []Effort{{Value: "low"}, {Value: "medium"}, {Value: "high"}, {Value: "xhigh"}, {Value: "max"}}, []string{"low", "medium", "high", "xhigh"}},
		{"official", []Effort{{Value: "minimal"}, {Value: "low"}, {Value: "medium"}, {Value: "high"}, {Value: "xhigh"}}, []string{"minimal", "low", "medium", "high", "xhigh"}},
		{"unknown-only", []Effort{{Value: "ultra"}}, nil},
		{"collision", []Effort{{Value: "none"}, {Value: "minimal"}}, []string{"minimal"}},
	}
	for _, tc := range cases {
		result := CodexEfforts(tc.efforts)
		if len(result) != len(tc.expected) {
			t.Fatalf("%s: levels = %v, want %v", tc.name, result, tc.expected)
		}
		for index, level := range tc.expected {
			if result[index].Value != level {
				t.Fatalf("%s: level[%d] = %q, want %q", tc.name, index, result[index].Value, level)
			}
		}
	}
}

func TestCodexUpstreamEffortRestoresUpstreamValue(t *testing.T) {
	efforts := []Effort{{Value: "none"}, {Value: "high"}, {Value: "max"}}
	for level, want := range map[string]string{"minimal": "none", "high": "high", "xhigh": "max"} {
		got, ok := CodexUpstreamEffort(efforts, level)
		if !ok || got != want {
			t.Fatalf("level %q = (%q,%t), want %q", level, got, ok, want)
		}
	}
	for _, level := range []string{"", "low", "medium", "none", "max"} {
		if _, ok := CodexUpstreamEffort(efforts, level); ok {
			t.Fatalf("unconfirmed level %q accepted", level)
		}
	}
}

// Codex 只发布自己认识的档位，选择器才不会只剩一个值。
func TestCodexJSONPublishesMappedLevelsAndDefault(t *testing.T) {
	data, err := CodexJSON([]Entry{{ID: "river-1017", Name: "V4.1 Flash 1017", Metadata: Metadata{
		ReasoningEfforts: []Effort{{Value: "none", Description: "关闭"}, {Value: "high"}, {Value: "max"}},
		DefaultEffort:    "high",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Models []struct {
			DefaultReasoningLevel    *string `json:"default_reasoning_level"`
			SupportedReasoningLevels []struct {
				Effort string `json:"effort"`
			} `json:"supported_reasoning_levels"`
		} `json:"models"`
	}
	if json.Unmarshal(data, &doc) != nil || len(doc.Models) != 1 {
		t.Fatalf("invalid catalog: %s", data)
	}
	model := doc.Models[0]
	if len(model.SupportedReasoningLevels) != 3 {
		t.Fatalf("levels = %v, want three mapped levels", model.SupportedReasoningLevels)
	}
	if model.DefaultReasoningLevel == nil || *model.DefaultReasoningLevel != "high" {
		t.Fatalf("default level = %v, want high", model.DefaultReasoningLevel)
	}
}
