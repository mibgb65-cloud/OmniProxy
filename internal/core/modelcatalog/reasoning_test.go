package modelcatalog

import (
	"encoding/json"
	"testing"
)

func TestCodexReasoningAlwaysHasSupportedDefault(t *testing.T) {
	for _, tc := range []struct {
		name, explicit, want string
		efforts              []Effort
	}{
		{"prefer_high", "", "high", []Effort{{Value: "low"}, {Value: "high"}, {Value: "max"}}},
		{"first_supported", "", "none", []Effort{{Value: "none"}, {Value: "medium"}}},
		{"preserve_explicit", "max", "max", []Effort{{Value: "high"}, {Value: "max"}}},
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
