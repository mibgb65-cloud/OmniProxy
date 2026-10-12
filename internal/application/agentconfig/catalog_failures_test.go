package agentconfig

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
)

func TestApplyRollsBackBothFilesWhenVersionCommitFails(t *testing.T) {
	s, _ := catalogFixture(t)
	if _, err := s.db.Exec(`CREATE TRIGGER reject_agent_version BEFORE UPDATE ON agent_backups BEGIN SELECT RAISE(FAIL,'private failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(context.Background(), Input{Client: "codex", Revision: digest(nil, false)}); err == nil || strings.Contains(err.Error(), "private") {
		t.Fatal("failed commit accepted or leaked", err)
	}
	for _, path := range []string{s.paths.Codex, s.catalogPath()} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("partial file remained", path)
		}
	}
}

func TestLocalDirectoryRefreshUsesOwnedVersionAndRestoresOriginal(t *testing.T) {
	s, source := catalogFixture(t)
	ctx := context.Background()
	if _, err := s.Apply(ctx, Input{Client: "codex", Revision: digest(nil, false)}); err != nil {
		t.Fatal(err)
	}
	source.entries = []catalog.Entry{{ID: "new-model"}}
	if err := s.RefreshCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(s.catalogPath())
	if err != nil || !strings.Contains(string(body), "new-model") || strings.Contains(string(body), "gpt-default") {
		t.Fatal("local catalog did not refresh")
	}
	if _, err := s.Restore(ctx, "codex"); err != nil {
		t.Fatal("refresh broke restore ownership", err)
	}
}

func TestUnownedCatalogFileAndEmptyDirectoryAreRejected(t *testing.T) {
	s, source := catalogFixture(t)
	if err := os.MkdirAll(filepath.Dir(s.catalogPath()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.catalogPath(), []byte("unowned"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PreviewPlan(context.Background(), Input{Client: "codex", Revision: digest(nil, false)}); err == nil || err.Error() != "agent: catalog_conflict" {
		t.Fatal("unowned file accepted", err)
	}
	source.entries = nil
	if _, err := s.Apply(context.Background(), Input{Client: "claude-desktop", Revision: digest(nil, false)}); err == nil || err.Error() != "agent: no_models" {
		t.Fatal("empty directory applied", err)
	}
	body, err := os.ReadFile(s.catalogPath())
	if err != nil || string(body) != "unowned" {
		t.Fatal("unowned file changed")
	}
}

func TestPlanRevisionIncludesClientDirectoryStrategy(t *testing.T) {
	s, _ := catalogFixture(t)
	body := []byte("model_provider='existing'\n")
	if err := os.MkdirAll(filepath.Dir(s.paths.Codex), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.paths.Codex, body, 0600); err != nil {
		t.Fatal(err)
	}
	input := Input{Client: "codex", Revision: digest(body, true)}
	s.SetCodexVersion("codex-cli 0.155.1")
	plan, err := s.PreviewPlan(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	input.CatalogRevision = plan.CatalogRevision
	s.SetCodexVersion("codex-cli 0.157.1")
	if _, err := s.Apply(context.Background(), input); err == nil || err.Error() != "agent: catalog_changed" {
		t.Fatal("strategy changed after preview", err)
	}
}
