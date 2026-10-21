package modelcatalog

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestServiceTierMetadataBounds(t *testing.T) {
	for _, test := range []struct {
		name  string
		tiers []ServiceTier
		valid bool
	}{
		{"unknown", nil, true},
		{"empty", []ServiceTier{}, true},
		{"confirmed", []ServiceTier{{"priority", "Fast", "Official description"}}, true},
		{"boundary", []ServiceTier{{strings.Repeat("a", 32), strings.Repeat("n", 80), strings.Repeat("d", 500)}}, true},
		{"empty id", []ServiceTier{{"", "Fast", ""}}, false},
		{"long id", []ServiceTier{{strings.Repeat("a", 33), "Fast", ""}}, false},
		{"invalid id", []ServiceTier{{"priority\n", "Fast", ""}}, false},
		{"empty name", []ServiceTier{{"priority", " ", ""}}, false},
		{"long name", []ServiceTier{{"priority", strings.Repeat("a", 81), ""}}, false},
		{"control name", []ServiceTier{{"priority", "Fast\x00", ""}}, false},
		{"long description", []ServiceTier{{"priority", "Fast", strings.Repeat("a", 501)}}, false},
		{"control description", []ServiceTier{{"priority", "Fast", "\x1b"}}, false},
		{"duplicate", []ServiceTier{{"priority", "Fast", ""}, {"priority", "Other", ""}}, false},
		{"too many", make([]ServiceTier, 9), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateMetadata(Metadata{ServiceTiers: test.tiers}); (err == nil) != test.valid {
				t.Fatal("service tier validation mismatch", err)
			}
		})
	}
}

func TestCodexCatalogPreservesOnlyConfirmedNativeServiceTiers(t *testing.T) {
	tiers := []ServiceTier{{"priority", "Fast", "Official usage description"}}
	body, err := CodexJSON([]Entry{
		{ID: "gpt-native", ProviderKind: "openai", Metadata: Metadata{ServiceTiers: tiers}},
		{ID: "native-alias", TargetModel: "gpt-native", Alias: true, ProviderKind: "openai", Metadata: Metadata{ServiceTiers: tiers}},
		{ID: "gpt-unknown", ProviderKind: "openai"},
		{ID: "gpt-third-party", ProviderKind: "deepseek", Metadata: Metadata{ServiceTiers: tiers}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Models []struct {
			Slug         string        `json:"slug"`
			ServiceTiers []ServiceTier `json:"service_tiers"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &response); err != nil || len(response.Models) != 4 {
		t.Fatal("invalid catalog", err)
	}
	for _, model := range response.Models {
		want := model.Slug == "gpt-native" || model.Slug == "native-alias"
		if want && !reflect.DeepEqual(model.ServiceTiers, tiers) || !want && len(model.ServiceTiers) != 0 {
			t.Fatal("Fast capability was lost or invented", model.Slug)
		}
	}
}
