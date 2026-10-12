package modelcatalog

import (
	"encoding/json"
	"testing"
)

func TestRemoteCodexCatalogUsesTheClientCompatibilityFilter(t *testing.T) {
	noTools := false
	body, err := CodexJSON([]Entry{{ID: "text-model"}, {ID: "messages-only", Protocols: []string{"messages"}}, {ID: "without-tools", Metadata: Metadata{ToolUse: &noTools}}})
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Models []struct {
			Slug string `json:"slug"`
		} `json:"models"`
	}
	if json.Unmarshal(body, &response) != nil || len(response.Models) != 1 || response.Models[0].Slug != "text-model" {
		t.Fatal("remote catalog exposed incompatible models")
	}
}
