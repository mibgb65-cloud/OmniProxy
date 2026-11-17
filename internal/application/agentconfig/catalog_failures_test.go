package agentconfig

import (
	"context"
	"errors"
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

// 后台刷新失败的原因要能被界面读到：只有日志时用户只会看到「模型没更新」。
func TestInspectSurfacesRefreshIssueCodes(t *testing.T) {
	s, _ := catalogFixture(t)
	ctx := context.Background()
	if _, err := s.Apply(ctx, Input{Client: "claude", Revision: digest(nil, false)}); err != nil {
		t.Fatal(err)
	}
	// 模拟后台刷新写失败（哈希仍匹配，单看文件看不出来），错误码必须出现在 Item 上。
	s.setIssues(map[string]string{"claude": "write_failed"})
	items, err := s.Inspect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if code := itemErrorCode(items, "claude"); code != "write_failed" {
		t.Fatalf("refresh issue not surfaced: %q", code)
	}
	// 文件被外部改写时 config_changed 优先（行动方式相同，且更具体）。
	external := []byte(`{"external":true}`)
	if err := os.WriteFile(s.paths.Claude, external, 0o600); err != nil {
		t.Fatal(err)
	}
	items, err = s.Inspect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if code := itemErrorCode(items, "claude"); code != "config_changed" {
		t.Fatalf("config_changed not preferred: %q", code)
	}
	// 重新应用后失败原因清除。
	if _, err := s.Apply(ctx, Input{Client: "claude", Revision: digest(external, true), AcceptCurrentConfig: true}); err != nil {
		t.Fatal(err)
	}
	items, err = s.Inspect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if code := itemErrorCode(items, "claude"); code != "" {
		t.Fatalf("issue survived reapply: %q", code)
	}
}

// 失败码只从本包固定前缀的错误里取；聚合错误取第一个可识别的码。
func TestFailureCodeExtractsStableCodes(t *testing.T) {
	if code := failureCode(failure("config_changed")); code != "config_changed" {
		t.Fatalf("single = %q", code)
	}
	joined := errors.Join(failure("write_failed"), errors.New("agent: unsafe"))
	if code := failureCode(joined); code != "write_failed" {
		t.Fatalf("joined = %q", code)
	}
	for _, err := range []error{
		errors.New("some other failure"),
		errors.New("agent: bad-Code"),
		errors.New("agent: " + strings.Repeat("x", 80)),
		nil,
	} {
		if code := failureCode(err); code != "" {
			t.Fatalf("unexpected code %q for %v", code, err)
		}
	}
}

func itemErrorCode(items []Item, client string) string {
	for _, item := range items {
		if item.Client == client {
			return item.ErrorCode
		}
	}
	return ""
}
