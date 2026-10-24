package account

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

// codexAuthFixture 生成一份可导入的 auth.json：id_token 携带展示用身份。
func codexAuthFixture(t *testing.T, accountID, email, plan, access, refresh string) []byte {
	t.Helper()
	claims := map[string]any{
		"email": email,
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": accountID,
			"chatgpt_plan_type":  plan,
		},
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	idToken := "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
	body, err := json.Marshal(map[string]any{
		"auth_mode": "chatgpt",
		"tokens": map[string]string{
			"id_token": idToken, "access_token": access, "refresh_token": refresh, "account_id": accountID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func codexService(t *testing.T) (*Service, *batchStore) {
	t.Helper()
	service, _ := testService(t)
	store := &batchStore{values: map[string][]byte{}}
	service.store = store
	return service, store
}

// TestImportCodexAuthStoresCredentialOutsideMetadata 断言 auth.json 只进保护存储，
// 元数据表里只留可展示身份。
func TestImportCodexAuthStoresCredentialOutsideMetadata(t *testing.T) {
	service, store := codexService(t)
	raw := codexAuthFixture(t, "acct-1", "user@example.test", "plus", "access-1", "refresh-1")
	item, err := service.ImportCodexAuth(context.Background(), "builtin_openai", "", raw)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if item.AuthMethod != AuthCodexFile || item.AuthEmail != "user@example.test" || item.AuthPlan != "plus" || item.AuthAccountID != "acct-1" {
		t.Fatalf("unexpected metadata: %+v", item)
	}
	if len(store.values) != 1 {
		t.Fatalf("expected exactly one protected credential, got %d", len(store.values))
	}
	var got []byte
	if err := service.UseCodexAuth(context.Background(), item.ID, func(data []byte) error {
		got = append([]byte(nil), data...)
		return nil
	}); err != nil {
		t.Fatalf("use: %v", err)
	}
	if !strings.Contains(string(got), `"refresh_token": "refresh-1"`) {
		t.Fatalf("protected credential lost refresh token: %s", got)
	}
}

// TestImportCodexAuthUpdatesSameOfficialAccount 断言同一官方账号重复导入不新增账号。
func TestImportCodexAuthUpdatesSameOfficialAccount(t *testing.T) {
	service, _ := codexService(t)
	ctx := context.Background()
	first, err := service.ImportCodexAuth(ctx, "builtin_openai", "", codexAuthFixture(t, "acct-1", "user@example.test", "plus", "access-1", "refresh-1"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.ImportCodexAuth(ctx, "builtin_openai", "", codexAuthFixture(t, "acct-1", "user@example.test", "plus", "access-2", "refresh-2"))
	if err != nil {
		t.Fatal(err)
	}
	items, err := service.List(ctx)
	if err != nil || len(items) != 1 || second.ID != first.ID {
		t.Fatalf("duplicate import created a second account: %d items, err=%v", len(items), err)
	}
	var got []byte
	if err := service.UseCodexAuth(ctx, first.ID, func(data []byte) error {
		got = append([]byte(nil), data...)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `"access_token": "access-2"`) {
		t.Fatalf("credential not refreshed by re-import: %s", got)
	}
}

// TestReplaceCodexAuthTokensKeepsVersion 断言续期回写不改身份与路由版本。
func TestReplaceCodexAuthTokensKeepsVersion(t *testing.T) {
	service, _ := codexService(t)
	ctx := context.Background()
	item, err := service.ImportCodexAuth(ctx, "builtin_openai", "", codexAuthFixture(t, "acct-1", "user@example.test", "plus", "access-1", "refresh-1"))
	if err != nil {
		t.Fatal(err)
	}
	refreshed := codexAuthFixture(t, "acct-1", "user@example.test", "plus", "access-2", "refresh-2")
	if err := service.ReplaceCodexAuthTokens(ctx, item.ID, refreshed); err != nil {
		t.Fatalf("replace: %v", err)
	}
	current, err := service.Metadata(ctx, item.ID)
	if err != nil || current.UpdatedAt != item.UpdatedAt || current.AuthAccountID != item.AuthAccountID {
		t.Fatalf("version or identity changed: %+v err=%v", current, err)
	}
	// 旧版本快照仍可读：续期不能让进行中的请求被判定为路由已变。
	var got []byte
	if err := service.UseCodexAuthAt(ctx, item.ID, item.UpdatedAt, func(data []byte) error {
		got = append([]byte(nil), data...)
		return nil
	}); err != nil || !strings.Contains(string(got), `"access_token":"access-2"`) {
		t.Fatalf("stale snapshot rejected after refresh: %v", err)
	}
}

// TestImportCodexAuthRejectsForeignOrBrokenPayload 断言非登录 JSON 一律拒绝。
func TestImportCodexAuthRejectsForeignOrBrokenPayload(t *testing.T) {
	service, _ := codexService(t)
	ctx := context.Background()
	for _, raw := range [][]byte{
		[]byte(`{"OPENAI_API_KEY":"sk-test"}`),
		[]byte(`{"tokens":{"access_token":"only-access"}}`),
		[]byte(`not json`),
	} {
		if _, err := service.ImportCodexAuth(ctx, "builtin_openai", "", raw); err == nil {
			t.Fatalf("accepted unusable payload: %s", raw)
		}
	}
}
