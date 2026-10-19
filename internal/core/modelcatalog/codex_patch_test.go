package modelcatalog

import (
	"encoding/json"
	"testing"
)

func TestCodexNativePatchToolDoesNotDisappearFromSubscriptionCatalog(t *testing.T) {
	body, err := CodexJSON([]Entry{{ID: "gpt-native", Name: "Native", ProviderKind: "openai"}, {ID: "third-party", Name: "Third party", ProviderKind: "deepseek"}})
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Models []struct {
			Slug  string  `json:"slug"`
			Patch *string `json:"apply_patch_tool_type"`
		} `json:"models"`
	}
	if json.Unmarshal(body, &result) != nil || len(result.Models) != 2 {
		t.Fatal("invalid generated catalog")
	}
	for _, model := range result.Models {
		if model.Slug == "gpt-native" && (model.Patch == nil || *model.Patch != "freeform") {
			t.Fatal("native subscription catalog disables structured apply_patch")
		}
		if model.Slug == "third-party" && model.Patch != nil {
			t.Fatal("unverified third-party patch capability was invented")
		}
	}
}
