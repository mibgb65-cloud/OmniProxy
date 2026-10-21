package agentconfig

import (
	"bytes"
	"context"
	"encoding/json"
	catalog "omniproxy/internal/core/modelcatalog"
	"os"
	"path/filepath"
	"testing"
)

func TestRefreshCatalogRestoresFastWithoutChangingClientSelection(t *testing.T) {
	s, source := catalogFixture(t)
	source.entries[0].ProviderKind = "openai"
	original := []byte("model_provider='openai'\nmodel='gpt-default'\nservice_tier='default'\n")
	if err := os.MkdirAll(filepath.Dir(s.paths.Codex), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.paths.Codex, original, 0600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := s.Apply(ctx, Input{Client: "codex", Revision: digest(original, true)}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(s.paths.Codex)
	if err != nil {
		t.Fatal(err)
	}
	source.entries[0].Metadata.ServiceTiers = []catalog.ServiceTier{{ID: "priority", Name: "Fast", Description: "Official usage"}}
	if err := s.RefreshCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(s.paths.Codex)
	if err != nil || !bytes.Equal(before, after) || !bytes.Contains(after, []byte("service_tier='default'")) {
		t.Fatal("catalog refresh changed user's Fast selection", err)
	}
	body, err := os.ReadFile(s.catalogPath())
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Models []struct {
			Slug         string                `json:"slug"`
			ServiceTiers []catalog.ServiceTier `json:"service_tiers"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &result); err != nil || len(result.Models) != 2 {
		t.Fatal("invalid refreshed catalog", err)
	}
	if result.Models[0].Slug != "gpt-default" || len(result.Models[0].ServiceTiers) != 1 || result.Models[0].ServiceTiers[0].ID != "priority" || len(result.Models[1].ServiceTiers) != 0 {
		t.Fatal("catalog refresh lost or invented Fast capability")
	}
}
