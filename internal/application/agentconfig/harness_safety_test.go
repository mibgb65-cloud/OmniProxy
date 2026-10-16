package agentconfig

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHarnessLegacyCredentialsSurviveModernMigrationAndRestore(t *testing.T) {
	s, _ := gatewayClientFixture(t)
	ctx := context.Background()
	writeGatewayFixture(t, s.paths.HarnessCredentials, "EXISTING_KEY: original-test\n")
	before, err := s.snapshotClient("harness-web", s.paths.HarnessWeb)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(ctx, Input{Client: "harness-web", Revision: before.revision}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(s.paths.HarnessCredentials)
	key, err := harnessCredential(data, harnessRef("harness-web"))
	if err != nil || key == nil || strings.Contains(string(data), "version:") {
		t.Fatal("legacy layout changed", err)
	}
	// 新版 Harness 启动时迁移布局，不改变本 profile 的受管版本和可恢复性。
	writeGatewayFixture(t, s.paths.HarnessCredentials, "version: 1\nrefs:\n  EXISTING_KEY: original-test\n  "+harnessRef("harness-web")+": "+*key+"\n")
	if _, err := s.Restore(ctx, "harness-web"); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(s.paths.HarnessCredentials)
	if strings.Contains(string(data), harnessRef("harness-web")) || !strings.Contains(string(data), "original-test") {
		t.Fatal("restore lost migrated credentials")
	}
}

func TestHarnessCommitFailureRollsBackBothFiles(t *testing.T) {
	s, _ := gatewayClientFixture(t)
	ctx := context.Background()
	writeGatewayFixture(t, s.paths.HarnessWeb, "# original config\n[]\n")
	writeGatewayFixture(t, s.paths.HarnessCredentials, "version: 1\nrefs:\n  EXISTING_KEY: original-test\n")
	before, err := s.snapshotClient("harness-web", s.paths.HarnessWeb)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.db.Exec(`CREATE TRIGGER fail_harness_commit BEFORE UPDATE ON agent_backups WHEN NEW.client='harness-web' BEGIN SELECT RAISE(ABORT,'fixture'); END`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(ctx, Input{Client: "harness-web", Revision: before.revision}); err == nil {
		t.Fatal("database failure hidden")
	}
	after, err := s.snapshotClient("harness-web", s.paths.HarnessWeb)
	if err != nil || after.revision != before.revision {
		t.Fatal("config or credential not rolled back", err)
	}
}

func TestOpenCodePathsHonorOverridesAndPreferJSONC(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OPENCODE_CONFIG", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", root)
	jsonc := filepath.Join(root, "opencode", "opencode.jsonc")
	writeGatewayFixture(t, jsonc, "{}")
	if openCodePath(root) != jsonc {
		t.Fatal("existing JSONC ignored")
	}
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(root, "custom"))
	if openCodePath(root) != filepath.Join(root, "custom", "opencode.json") {
		t.Fatal("custom directory ignored")
	}
	custom := filepath.Join(root, "explicit.json")
	t.Setenv("OPENCODE_CONFIG", custom)
	if openCodePath(root) != custom {
		t.Fatal("explicit file ignored")
	}
}

func TestOpenCodeManagedPortChangeKeepsUserProviderCollisionProtection(t *testing.T) {
	s, _ := gatewayClientFixture(t)
	ctx := context.Background()
	before, err := s.snapshotClient("opencode", s.paths.OpenCode)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(ctx, Input{Client: "opencode", Revision: before.revision}); err != nil {
		t.Fatal(err)
	}
	s.port = 4001
	if err := s.RefreshCatalog(ctx); err != nil {
		t.Fatal("managed port update rejected", err)
	}
	body, err := os.ReadFile(s.paths.OpenCode)
	if err != nil || !strings.Contains(string(body), "127.0.0.1:4001/v1") {
		t.Fatal("managed gateway URL not refreshed")
	}
	other, _ := gatewayClientFixture(t)
	writeGatewayFixture(t, other.paths.OpenCode, `{"provider":{"omniproxy":{"options":{"baseURL":"https://different.example/v1"}}}}`)
	snapshot, err := other.snapshotClient("opencode", other.paths.OpenCode)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.PreviewPlan(ctx, Input{Client: "opencode", Revision: snapshot.revision}); err == nil {
		t.Fatal("unowned provider collision accepted")
	}
}
