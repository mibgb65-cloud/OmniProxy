package agentconfig

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
)

func TestCodexImageCapabilityApplyRefreshAndExternalEditProtection(t *testing.T) {
	s, c := catalogFixture(t)
	c.entries = []catalog.Entry{{ID: "gpt-6.1-sol", ProviderKind: "openai", Metadata: catalog.Metadata{InputModalities: []string{"text", "image"}}}}
	original := []byte("model_provider='openai'\n")
	if err := os.MkdirAll(filepath.Dir(s.paths.Codex), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.paths.Codex, original, 0600); err != nil {
		t.Fatal(err)
	}
	input := Input{Client: "codex", Revision: digest(original, true), Model: "gpt-6.1-sol"}
	plan, err := s.PreviewPlan(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	input.CatalogRevision = plan.CatalogRevision
	if _, err = s.Apply(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	check := func(want []string) []byte {
		t.Helper()
		raw, e := os.ReadFile(s.catalogPath())
		if e != nil {
			t.Fatal(e)
		}
		var r struct {
			Models []struct {
				Modalities []string `json:"input_modalities"`
			} `json:"models"`
		}
		if json.Unmarshal(raw, &r) != nil || len(r.Models) != 1 || !slices.Equal(r.Models[0].Modalities, want) {
			t.Fatal("managed image capability incorrect")
		}
		return raw
	}
	check([]string{"text", "image"})
	c.entries[0].Metadata.InputModalities = []string{"text"}
	if err = s.RefreshCatalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	before := check([]string{"text"})
	config, err := os.ReadFile(s.paths.Codex)
	if err != nil {
		t.Fatal(err)
	}
	changed := append(config, []byte("\n# external change\n")...)
	if err = os.WriteFile(s.paths.Codex, changed, 0600); err != nil {
		t.Fatal(err)
	}
	c.entries[0].Metadata.InputModalities = []string{"text", "image"}
	if err = s.RefreshCatalog(context.Background()); err == nil {
		t.Fatal("external edit was overwritten")
	}
	after, _ := os.ReadFile(s.catalogPath())
	if string(after) != string(before) {
		t.Fatal("catalog changed after protected config drift")
	}
	after, _ = os.ReadFile(s.paths.Codex)
	if string(after) != string(changed) {
		t.Fatal("external config changed")
	}
}
