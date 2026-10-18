package modelcatalog

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestCodexOnlyAdvertisesVerifiedNativeImageInput(t *testing.T) {
	entries := []Entry{
		{ID: "gpt-6.1-sol", ProviderKind: "openai", Metadata: Metadata{InputModalities: []string{"text", "image"}}},
		{ID: "vision-route", ProviderKind: "openai", Alias: true, TargetModel: "gpt-6.1-sol", Metadata: Metadata{InputModalities: []string{"text", "image"}}},
		{ID: "unknown-openai", ProviderKind: "openai"},
		{ID: "text-openai", ProviderKind: "openai", Metadata: Metadata{InputModalities: []string{"text"}}},
		{ID: "converted-vision", ProviderKind: "google", Metadata: Metadata{InputModalities: []string{"text", "image"}}},
		{ID: "unknown-route", Metadata: Metadata{InputModalities: []string{"text", "image"}}},
		{ID: "audio-openai", ProviderKind: "openai", Metadata: Metadata{InputModalities: []string{"text", "audio"}}},
	}
	body, err := CodexJSON(entries)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Models []struct {
			ID         string   `json:"slug"`
			Modalities []string `json:"input_modalities"`
		} `json:"models"`
	}
	if json.Unmarshal(body, &result) != nil || len(result.Models) != len(entries) {
		t.Fatal("catalog missing entries")
	}
	for _, m := range result.Models {
		want := []string{"text"}
		if m.ID == "gpt-6.1-sol" || m.ID == "vision-route" {
			want = append(want, "image")
		}
		if !slices.Equal(m.Modalities, want) {
			t.Errorf("%s modalities = %v, want %v", m.ID, m.Modalities, want)
		}
	}
}
