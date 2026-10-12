package modelcatalog

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCatalogOrderingAndUnknownCapabilities(t *testing.T) {
	entries, err := Normalize([]Entry{{ID: "z-model", Metadata: Metadata{IsDefault: true}}, {ID: "a-model"}, {ID: "a-model"}})
	if err != nil || len(entries) != 2 || entries[0].ID != "z-model" || entries[1].Name != "a-model" {
		t.Fatal("unstable catalog", err)
	}
	body, err := CodexJSON(entries)
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Models []map[string]json.RawMessage `json:"models"`
	}
	if json.Unmarshal(body, &response) != nil || len(response.Models) != 2 {
		t.Fatal("invalid Codex catalog")
	}
	for _, model := range response.Models {
		if _, found := model["context_window"]; found {
			t.Fatal("unknown context was invented")
		}
		if string(model["supported_reasoning_levels"]) != "[]" || string(model["input_modalities"]) != `["text"]` {
			t.Fatal("unsupported gateway capabilities advertised")
		}
		if string(model["visibility"]) != `"list"` || len(model["model_messages"]) == 0 {
			t.Fatal("picker or coding instructions missing")
		}
	}
}

func TestConfirmedMetadataAndBounds(t *testing.T) {
	window := int64(100000)
	metadata := Metadata{ContextWindow: &window, DefaultEffort: "high", ReasoningEfforts: []Effort{{Value: "high", Description: "Confirmed"}}}
	if ValidateMetadata(metadata) != nil {
		t.Fatal("confirmed metadata rejected")
	}
	for _, invalid := range []Metadata{
		{Description: strings.Repeat("x", 2001)},
		{DefaultEffort: "unsupported"},
		{InputModalities: []string{"unknown"}},
		{ReasoningEfforts: []Effort{{Value: "high"}, {Value: "high"}}},
	} {
		if ValidateMetadata(invalid) == nil {
			t.Fatal("invalid metadata accepted")
		}
	}
	window = -1
	if ValidateMetadata(metadata) == nil {
		t.Fatal("negative context accepted")
	}
}
