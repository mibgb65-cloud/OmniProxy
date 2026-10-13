package gateway

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	catalog "omniproxy/internal/core/modelcatalog"
)

// TestLiveClaudeCodeDeepSeek 通过隔离 CLI 配置验证真实 beta Messages、SSE 与客户端上下文。
func TestLiveClaudeCodeDeepSeek(t *testing.T) {
	binary := os.Getenv("OMNIPROXY_LIVE_CLAUDE_BINARY")
	if binary == "" {
		t.Skip("真实 Claude Code 验证需显式提供本机 CLI")
	}
	service := liveDeepSeekService(t)
	for _, id := range []string{"deepseek-flash", "deepseek-v4-pro"} {
		for _, long := range []bool{false, true} {
			model := id
			if long {
				model += catalog.ClaudeLongContextSuffix
			}
			// 桌面同版本核心使用兼容 ID；CLI 也验证真实模型 ID。
			if id == "deepseek-v4-pro" {
				model = catalog.DesktopAlias(model)
			}
			t.Run(model, func(t *testing.T) {
				directory := t.TempDir()
				settings := filepath.Join(directory, "settings.json")
				if err := os.WriteFile(settings, []byte(`{}`), 0600); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()
				args := []string{"-p", "Reply OK.", "--model", model, "--system-prompt", "Reply briefly.", "--settings", settings, "--setting-sources", "", "--no-session-persistence", "--output-format", "json"}
				// Pro 普通模式还验证真实默认工具目录，其他组合使用最小请求降低实机消耗。
				if id != "deepseek-v4-pro" || long {
					args = append(args, "--tools", "")
				}
				cmd := exec.CommandContext(ctx, binary, args...)
				cmd.Dir = directory
				for _, item := range os.Environ() {
					key, _, _ := strings.Cut(item, "=")
					key = strings.ToUpper(key)
					if !strings.HasPrefix(key, "ANTHROPIC_") && !strings.HasPrefix(key, "CLAUDE_") {
						cmd.Env = append(cmd.Env, item)
					}
				}
				cmd.Env = append(cmd.Env, "ANTHROPIC_BASE_URL="+service.Info().Runtime.Address, "ANTHROPIC_AUTH_TOKEN=isolated-gateway-placeholder", "CLAUDE_CONFIG_DIR="+directory, "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1")
				output, err := cmd.Output()
				defer clear(output)
				if err != nil {
					// CLI 原始输出可能含环境信息，只报告稳定分类，禁止转储诊断正文。
					for _, code := range []string{"invalid_request", "unsupported_content", "invalid_model", "unsupported_model", "credential_unavailable", "no_enabled_deepseek_account", "upstream_unavailable", "invalid_upstream_response", "upstream_rejected_request", "upstream_authentication_failed", "upstream_insufficient_balance"} {
						if strings.Contains(string(output), code) {
							t.Logf("gateway_code=%s", code)
						}
					}
					t.Fatalf("isolated Claude Code failed: %T", err)
				}
				var result struct {
					IsError    bool   `json:"is_error"`
					Result     string `json:"result"`
					ModelUsage map[string]struct {
						ContextWindow int64 `json:"contextWindow"`
					} `json:"modelUsage"`
				}
				if json.Unmarshal(output, &result) != nil || result.IsError || strings.TrimSpace(result.Result) == "" || len(result.ModelUsage) == 0 {
					t.Fatal("isolated Claude Code did not complete the conversation")
				}
				window := int64(200000)
				if long {
					window = catalog.ClaudeLongContextWindow
				}
				for _, usage := range result.ModelUsage {
					t.Logf("provider=DeepSeek model=%s long_context=%t client_context=%d completed=true", id, long, usage.ContextWindow)
					if usage.ContextWindow != window {
						t.Fatal("Claude Code context window does not match selected option")
					}
				}
			})
		}
	}
}
