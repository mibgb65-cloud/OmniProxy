package codex

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAuthorizationURLBoundary(t *testing.T) {
	for _, value := range []string{"https://auth.openai.com/codex/device", "https://chatgpt.com/auth?state=test"} {
		if !ValidAuthorizationURL(value) {
			t.Fatal("official URL rejected")
		}
	}
	for _, value := range []string{"http://auth.openai.com", "https://auth.openai.com.evil.test", "https://user:pass@auth.openai.com", "https://auth.openai.com:443", "https://auth.openai.com/#token", "https://auth.openai.com\n", "file:///tmp/auth"} {
		if ValidAuthorizationURL(value) {
			t.Fatalf("unsafe URL accepted: %s", value)
		}
	}
}
func TestEnvironmentDoesNotReuseGlobalAuthentication(t *testing.T) {
	t.Setenv("CODEX_HOME", "global-home")
	t.Setenv("OPENAI_API_KEY", "private-test-key")
	t.Setenv("CODEX_ACCESS_TOKEN", "private-test-token")
	environment := strings.Join(isolatedEnvironment("isolated-home"), "\n")
	if strings.Contains(environment, "private-test") || strings.Contains(environment, "global-home") || !strings.Contains(environment, "CODEX_HOME=isolated-home") {
		t.Fatal("global credentials reused")
	}
}
func TestInstalledCodexIsolatedProbe(t *testing.T) {
	if os.Getenv("OMNIPROXY_CODEX_SMOKE") != "1" {
		t.Skip("requires explicitly enabled local Codex installation")
	}
	// 模拟从 Codex 桌面启动开发环境，授权必须使用本应用初始化的身份。
	t.Setenv("CODEX_INTERNAL_ORIGINATOR_OVERRIDE", "Codex Desktop")
	t.Setenv("CODEX_APP_SERVER_LOGIN_CLIENT_ID", "launcher-client")
	t.Setenv("CODEX_APP_SERVER_LOGIN_ISSUER", "https://launcher.invalid")
	home := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client, err := Open(ctx, home)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	identity, err := client.Account(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if identity != nil {
		t.Fatal("isolated home inherited an account")
	}
	login, err := client.Begin(ctx, "browser")
	if err != nil || !ValidAuthorizationURL(login.AuthURL) {
		t.Fatal("official browser authorization did not start")
	}
	parsed, err := url.Parse(login.AuthURL)
	if err != nil {
		t.Fatal("invalid authorization URL")
	}
	query := parsed.Query()
	if parsed.Host != "auth.openai.com" || query.Get("originator") != legacyBrowserOriginator || query.Get("client_id") == "launcher-client" {
		t.Fatal("login inherited launcher identity or issuer")
	}
	if query.Get("codex_cli_simplified_flow") != "true" || query.Get("code_challenge_method") != "S256" || query.Get("state") == "" || query.Get("code_challenge") == "" {
		t.Fatal("official authorization security parameters changed")
	}
	if err := client.Cancel(ctx, login.ID); err != nil {
		t.Fatal(err)
	}
	if err := client.Logout(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, "auth.json")); !os.IsNotExist(err) {
		t.Fatal("plaintext credential cache present")
	}
}
