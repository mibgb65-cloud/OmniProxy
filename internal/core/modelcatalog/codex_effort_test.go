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
		{"river-deepseek", []Effort{{Value: "none"}, {Value: "high"}, {Value: "max"}}, []string{"low", "high", "xhigh"}},
		{"river-glm", []Effort{{Value: "low"}, {Value: "high"}, {Value: "max"}}, []string{"low", "high", "xhigh"}},
		{"minimax", []Effort{{Value: "low"}, {Value: "medium"}, {Value: "high"}, {Value: "xhigh"}, {Value: "max"}}, []string{"low", "medium", "high", "xhigh"}},
		{"official", []Effort{{Value: "minimal"}, {Value: "low"}, {Value: "medium"}, {Value: "high"}, {Value: "xhigh"}}, []string{"minimal", "low", "medium", "high", "xhigh"}},
		// official catalog 的 GPT 档位含 max 与 ultra：max 就近映射为 xhigh，
		// ultra 本身是客户端词汇，必须原样保留而不是被当成未知值丢弃。
		{"official-gpt", []Effort{{Value: "low"}, {Value: "medium"}, {Value: "high"}, {Value: "xhigh"}, {Value: "max"}, {Value: "ultra"}}, []string{"low", "medium", "high", "xhigh", "ultra"}},
		{"unknown-only", []Effort{{Value: "turbo"}}, nil},
		{"collision", []Effort{{Value: "none"}, {Value: "low"}}, []string{"low"}},
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
	for level, want := range map[string]string{"low": "none", "high": "high", "xhigh": "max"} {
		got, ok := CodexUpstreamEffort(efforts, level)
		if !ok || got != want {
			t.Fatalf("level %q = (%q,%t), want %q", level, got, ok, want)
		}
	}
	for _, level := range []string{"", "minimal", "medium", "none", "max"} {
		if _, ok := CodexUpstreamEffort(efforts, level); ok {
			t.Fatalf("unconfirmed level %q accepted", level)
		}
	}
	// gpt-6.1-sol 的官方档位：xhigh 与 ultra 各自原样还原，不能互相折叠。
	official := []Effort{{Value: "low"}, {Value: "medium"}, {Value: "high"}, {Value: "xhigh"}, {Value: "max"}, {Value: "ultra"}}
	for level, want := range map[string]string{"low": "low", "xhigh": "xhigh", "ultra": "ultra"} {
		got, ok := CodexUpstreamEffort(official, level)
		if !ok || got != want {
			t.Fatalf("official level %q = (%q,%t), want %q", level, got, ok, want)
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
