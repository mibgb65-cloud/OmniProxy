package modelcatalog

import (
	"encoding/json"
	"testing"
)

func TestCodexReasoningAlwaysHasSupportedDefault(t *testing.T) {
	// Codex 只接受 official catalog 的档位词汇：上游 none／max 就近映射为 minimal／xhigh，
	// 因此默认档位也必须按映射后的值发布，否则选择器会拿到不认识的默认值。
	for _, tc := range []struct {
		name, explicit, want string
		efforts              []Effort
	}{
		{"prefer_high", "", "high", []Effort{{Value: "low"}, {Value: "high"}, {Value: "max"}}},
		{"first_supported", "", "minimal", []Effort{{Value: "none"}, {Value: "medium"}}},
		{"preserve_explicit", "max", "xhigh", []Effort{{Value: "high"}, {Value: "max"}}},
		{"unknown", "", "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := CodexJSON([]Entry{{ID: "third-party", Metadata: Metadata{ReasoningEfforts: tc.efforts, DefaultEffort: tc.explicit}}})
			if err != nil {
				t.Fatal(err)
			}
			var result struct {
				Models []struct {
					Default *string `json:"default_reasoning_level"`
				} `json:"models"`
			}
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatal(err)
			}
			got := result.Models[0].Default
			if tc.want == "" && got != nil || tc.want != "" && (got == nil || *got != tc.want) {
				t.Fatal("missing or fabricated default", got)
			}
		})
	}
}
