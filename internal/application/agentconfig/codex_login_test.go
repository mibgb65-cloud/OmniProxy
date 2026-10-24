package agentconfig

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestApplyWritesPoolAccountAsClientLogin 断言 v1 语义：网关拥有登录。
// 一键配置会把池账号写成客户端 auth.json，并保留用户原来的登录供恢复。
func TestApplyWritesPoolAccountAsClientLogin(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	dir := filepath.Dir(s.paths.Codex)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte("model_provider='openai'\nmodel='old'\n")
	if err := os.WriteFile(s.paths.Codex, original, 0o600); err != nil {
		t.Fatal(err)
	}
	authPath := filepath.Join(dir, "auth.json")
	userAuth := []byte(`{"auth_mode":"chatgpt","tokens":{"access_token":"user-official"}}`)
	if err := os.WriteFile(authPath, userAuth, 0o600); err != nil {
		t.Fatal(err)
	}
	poolAuth := []byte(`{"auth_mode":"chatgpt","tokens":{"access_token":"pool-access","refresh_token":"pool-refresh","account_id":"acct-1"}}`)
	s.SetCodexLogin(func(context.Context) ([]byte, error) { return poolAuth, nil })

	if _, err := s.Apply(ctx, Input{Client: "codex", Model: "gpt-test", Revision: digest(original, true)}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	written, err := os.ReadFile(authPath)
	if err != nil || !strings.Contains(string(written), "pool-access") {
		t.Fatalf("client login not replaced by pool account: %s err=%v", written, err)
	}
	backup, err := os.ReadFile(authPath + codexAuthBackupSuffix)
	if err != nil || !strings.Contains(string(backup), "user-official") {
		t.Fatalf("original login not backed up: %s err=%v", backup, err)
	}
	if _, err := s.Restore(ctx, "codex"); err != nil {
		t.Fatalf("restore: %v", err)
	}
	restored, err := os.ReadFile(authPath)
	if err != nil || !strings.Contains(string(restored), "user-official") {
		t.Fatalf("original login not restored: %s err=%v", restored, err)
	}
	if _, statErr := os.Lstat(authPath + codexAuthBackupSuffix); !os.IsNotExist(statErr) {
		t.Fatal("backup file should be removed after restore")
	}
}

// TestApplyWithoutPoolAccountKeepsClientLogin 断言没有池账号时不接管登录。
func TestApplyWithoutPoolAccountKeepsClientLogin(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	dir := filepath.Dir(s.paths.Codex)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte("model_provider='openai'\nmodel='old'\n")
	if err := os.WriteFile(s.paths.Codex, original, 0o600); err != nil {
		t.Fatal(err)
	}
	authPath := filepath.Join(dir, "auth.json")
	userAuth := []byte(`{"auth_mode":"chatgpt","tokens":{"access_token":"user-official"}}`)
	if err := os.WriteFile(authPath, userAuth, 0o600); err != nil {
		t.Fatal(err)
	}
	s.SetCodexLogin(func(context.Context) ([]byte, error) { return nil, nil })
	if _, err := s.Apply(ctx, Input{Client: "codex", Model: "gpt-test", Revision: digest(original, true)}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	written, err := os.ReadFile(authPath)
	if err != nil || !strings.Contains(string(written), "user-official") {
		t.Fatalf("client login must stay untouched: %s err=%v", written, err)
	}
	if _, statErr := os.Lstat(authPath + codexAuthBackupSuffix); !os.IsNotExist(statErr) {
		t.Fatal("no backup should be written when the login is not taken over")
	}
}
