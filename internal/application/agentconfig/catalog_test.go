package agentconfig

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	catalog "omniproxy/internal/core/modelcatalog"
)

type changingCatalog struct{ entries []catalog.Entry }

func (c *changingCatalog) Catalog(context.Context) ([]catalog.Entry, error) { return c.entries, nil }
func (c *changingCatalog) AvailableModels(context.Context) ([]string, error) {
	ids := make([]string, 0, len(c.entries))
	for _, model := range c.entries {
		ids = append(ids, model.ID)
	}
	return ids, nil
}
func catalogFixture(t *testing.T) (*Service, *changingCatalog) {
	t.Helper()
	s := fixture(t)
	c := &changingCatalog{entries: []catalog.Entry{
		{ID: "gpt-default", Name: "Default GPT", Metadata: catalog.Metadata{IsDefault: true}},
		{ID: "coding-route", Name: "Coding Route", Alias: true},
	}}
	s.catalog = c
	return s, c
}

func TestAutomaticLocalCatalogPreservesIdentityAndRestoresFiles(t *testing.T) {
	s, _ := catalogFixture(t)
	original := []byte("model_provider='openai'\nmodel='coding-route'\nmodel_reasoning_effort='high'\n# preserved\n")
	if err := os.MkdirAll(filepath.Dir(s.paths.Codex), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.paths.Codex, original, 0600); err != nil {
		t.Fatal(err)
	}
	input := Input{Client: "codex", Revision: digest(original, true)}
	plan, err := s.PreviewPlan(context.Background(), input)
	if err != nil || plan.DefaultModel != "coding-route" || plan.Mode != "local-catalog" || len(plan.Models) != 2 {
		t.Fatal("automatic preview", err)
	}
	if _, err := os.Stat(s.catalogPath()); !os.IsNotExist(err) {
		t.Fatal("preview created catalog")
	}
	input.CatalogRevision = plan.CatalogRevision
	if _, err := s.Apply(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(s.paths.Codex)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if toml.Unmarshal(body, &config) != nil || config["model_provider"] != "openai" || config["model"] != "coding-route" || config["model_reasoning_effort"] != "high" || config["model_catalog_json"] != s.catalogPath() {
		t.Fatal("identity or preference changed")
	}
	data, err := os.ReadFile(s.catalogPath())
	if err != nil || !strings.Contains(string(data), "gpt-default") || !strings.Contains(string(data), "coding-route") || strings.Contains(string(data), "opk_") {
		t.Fatal("catalog incomplete or unsafe", err)
	}
	if _, err := s.Restore(context.Background(), "codex"); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(s.paths.Codex)
	if err != nil || string(data) != string(original) {
		t.Fatal("original config not restored")
	}
	if _, err := os.Stat(s.catalogPath()); !os.IsNotExist(err) {
		t.Fatal("managed catalog remained after restore")
	}
}

func TestCatalogRevisionAndExternalFileProtection(t *testing.T) {
	s, c := catalogFixture(t)
	input := Input{Client: "codex", Revision: digest(nil, false)}
	plan, err := s.PreviewPlan(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	input.CatalogRevision = plan.CatalogRevision
	c.entries = append(c.entries, catalog.Entry{ID: "new-model"})
	if _, err := s.Apply(context.Background(), input); err == nil || err.Error() != "agent: catalog_changed" {
		t.Fatal("stale preview applied", err)
	}
	input.CatalogRevision = ""
	if _, err := s.Apply(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.catalogPath(), []byte("external edit"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.RefreshCatalog(context.Background()); err == nil {
		t.Fatal("external catalog edit overwritten")
	}
	if _, err := s.Restore(context.Background(), "codex"); err == nil {
		t.Fatal("external catalog edit deleted")
	}
	data, err := os.ReadFile(s.catalogPath())
	if err != nil || string(data) != "external edit" {
		t.Fatal("external data changed")
	}
}

func TestDirectoryRefreshAndDesktopFullList(t *testing.T) {
	s, c := catalogFixture(t)
	ctx := context.Background()
	if _, err := s.Apply(ctx, Input{Client: "claude-desktop", Revision: digest(nil, false)}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(s.paths.ClaudeDesktop)
	if err != nil {
		t.Fatal(err)
	}
	var startup map[string]any
	if json.Unmarshal(data, &startup) != nil || startup["disableDeploymentModeChooser"] != true {
		t.Fatal("gateway profile still allows official sign-in instead of local startup")
	}
	var config struct {
		Models []struct {
			Name  string `json:"name"`
			Label string `json:"labelOverride"`
		} `json:"inferenceModels"`
		Discovery bool `json:"modelDiscoveryEnabled"`
	}
	if json.Unmarshal(data, &config) != nil || len(config.Models) != 2 || config.Models[0].Name != catalog.DesktopAlias("gpt-default") || config.Models[0].Label != "Default GPT" || config.Discovery {
		t.Fatal("desktop models not configured")
	}
	c.entries = []catalog.Entry{{ID: "new-model", Name: "New Model"}}
	if err := s.RefreshCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(s.paths.ClaudeDesktop)
	if err != nil || !strings.Contains(string(data), catalog.DesktopAlias("new-model")) || !strings.Contains(string(data), "New Model") || strings.Contains(string(data), catalog.DesktopAlias("gpt-default")) {
		t.Fatal("desktop directory not refreshed")
	}
	if _, err := s.Restore(ctx, "claude-desktop"); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteModeRequiresSupportedCustomProvider(t *testing.T) {
	for _, test := range []struct{ provider, version, mode string }{
		{"existing", "codex-cli 0.155.1", "local-catalog"},
		{"existing", "codex-cli 0.157.1", "remote-catalog"},
		{"openai", "codex-cli 0.157.1", "local-catalog"},
	} {
		t.Run(test.provider+test.version, func(t *testing.T) {
			s, _ := catalogFixture(t)
			s.SetCodexVersion(test.version)
			body := []byte("model_provider='" + test.provider + "'\n")
			if err := os.MkdirAll(filepath.Dir(s.paths.Codex), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(s.paths.Codex, body, 0600); err != nil {
				t.Fatal(err)
			}
			plan, err := s.PreviewPlan(context.Background(), Input{Client: "codex", Revision: digest(body, true)})
			if err != nil || plan.Mode != test.mode {
				t.Fatal("incorrect version strategy", err)
			}
		})
	}
}
