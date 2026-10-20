package codex

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInstalledCodexLegacyFileLogin(t *testing.T) {
	if os.Getenv("OMNIPROXY_CODEX_LEGACY_QA") != "1" {
		t.Skip("需要显式启用本机旧版文件登录兼容验收")
	}
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("ALL_PROXY", "http://127.0.0.1:1")
	home := t.TempDir()
	payload, err := json.Marshal(map[string]any{
		"sub": "synthetic-user", "email": "legacy-trial@example.invalid",
		"exp": time.Now().Add(time.Hour).Unix(),
		"https://api.openai.com/auth": map[string]string{
			"chatgpt_plan_type": "plus", "chatgpt_account_id": "synthetic-account",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	token := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`)) + "." + base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString([]byte("synthetic-signature"))
	auth, err := json.Marshal(map[string]any{
		"auth_mode": "chatgpt",
		"tokens": map[string]string{
			"id_token": token, "access_token": token,
			"refresh_token": "synthetic-refresh", "account_id": "synthetic-account",
		},
		"last_refresh": time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "auth.json"), auth, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	executable, err := exec.LookPath("codex")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("model_provider='openai'\nforced_login_method='chatgpt'\ncli_auth_credentials_store='file'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, executable, "login", "status")
	hideWindow(cmd)
	cmd.Env = isolatedEnvironment(home)
	cmd.Dir = home
	output, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "Logged in using ChatGPT") {
		t.Fatal("当前 CLI 未识别合成 ChatGPT 文件登录", err)
	}
	if _, err := os.Stat(filepath.Join(home, "sessions")); !os.IsNotExist(err) {
		t.Fatal("登录兼容检查不应创建聊天")
	}
}
