package agentconfig

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func TestCodexConfigurationLeavesNativeWebSocketDefaults(t *testing.T) {
	body, _, err := buildCodex(nil, []byte("test-only-key"), "http://127.0.0.1:3001/codex/v1", "gpt-test")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "responses_websockets") {
		t.Fatal("仍在写入已移除的传输标记")
	}
	body, _, err = buildCodex([]byte("model_provider='custom'\n[model_providers.custom]\nsupports_websockets=false\n"),
		[]byte("test-only-key"), "http://127.0.0.1:3001/codex/v1", "gpt-test")
	var config map[string]any
	if err != nil || toml.Unmarshal(body, &config) != nil {
		t.Fatal("自定义受管入口配置失败", err)
	}
	entry := config["model_providers"].(map[string]any)["custom"].(map[string]any)
	if entry["supports_websockets"] != true {
		t.Fatal("已支持 WS 的受管入口仍禁止客户端使用")
	}
}

func TestManagedCodexRefreshCleansOnlyOwnedLegacyFlagsAndRestores(t *testing.T) {
	for _, mode := range []string{"managed_flags", "user_false", "user_true"} {
		t.Run(mode, func(t *testing.T) {
			userOwned := mode != "managed_flags"
			s := fixture(t)
			ctx := context.Background()
			original := "# keep original comment\nmodel='old'\n[features]\nmulti_agent=true\n"
			if userOwned {
				value := "false"
				if mode == "user_true" {
					value = "true"
				}
				original += "responses_websockets=" + value + "\nresponses_websockets_v2=" + value + "\n"
			}
			if err := os.MkdirAll(filepath.Dir(s.paths.Codex), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(s.paths.Codex, []byte(original), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Apply(ctx, Input{Client: "codex", Model: "gpt-test", Revision: digest([]byte(original), true)}); err != nil {
				t.Fatal(err)
			}
			applied, err := os.ReadFile(s.paths.Codex)
			if err != nil {
				t.Fatal(err)
			}
			legacy, err := patchTOML(applied, "openai", map[string]string{
				"features.responses_websockets": "false", "features.responses_websockets_v2": "false",
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(s.paths.Codex, legacy, 0600); err != nil {
				t.Fatal(err)
			}
			// 模拟旧版本已经提交的受管配置，而不是允许刷新覆盖外部修改。
			if _, err := s.db.ExecContext(ctx, `UPDATE agent_backups SET applied_hash=? WHERE client='codex'`, digest(legacy, true)); err != nil {
				t.Fatal(err)
			}
			if err := s.RefreshCatalog(ctx); err != nil {
				t.Fatal(err)
			}
			refreshed, err := os.ReadFile(s.paths.Codex)
			if err != nil || strings.Contains(string(refreshed), "responses_websockets") != userOwned ||
				!strings.Contains(string(refreshed), "multi_agent") {
				t.Fatal("误删用户值或未清理受管旧标记", err)
			}
			if mode == "user_true" && !strings.Contains(string(refreshed), "responses_websockets = true") {
				t.Fatal("旧版本覆盖的用户传输偏好没有还原")
			}
			if _, err := s.Restore(ctx, "codex"); err != nil {
				t.Fatal("传输清理破坏了恢复", err)
			}
			restored, err := os.ReadFile(s.paths.Codex)
			if err != nil || string(restored) != original {
				t.Fatal("原配置未精确恢复", err)
			}
		})
	}
}
