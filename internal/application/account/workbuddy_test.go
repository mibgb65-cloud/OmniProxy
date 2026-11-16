package account

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/provider/workbuddyai"
)

func workbuddyTestCredential() workbuddyai.Credentials {
	return workbuddyai.Credentials{
		AccessToken:      "fake-access-token",
		RefreshToken:     "fake-refresh-token",
		ExpiresAt:        time.Now().Add(time.Hour).UTC(),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour).UTC(),
		Domain:           "example.test",
		TokenType:        "Bearer",
		UID:              "workbuddy-uid-1",
		EnterpriseID:     "workbuddy-ent-1",
		AccountName:      "Tester",
		Plan:             "pro",
	}
}

// 新建 WorkBuddy AI 账号必须真正落库：这条 INSERT 曾因列数与值数不匹配而永远失败，
// 表现为浏览器登录成功后界面只报通用错误、账号列表仍为空。
func TestWorkBuddyLoginAccountIsSaved(t *testing.T) {
	service, _ := testService(t)
	store := &batchStore{values: map[string][]byte{}}
	service.store = store
	ctx := context.Background()
	cred := workbuddyTestCredential()

	item, err := service.SaveWorkBuddyAccount(ctx, "builtin_workbuddy_ai", "", "", cred)
	if err != nil {
		t.Fatalf("save account: %v", err)
	}
	if item.AuthMethod != AuthWorkBuddyLogin || item.AuthState != "authenticated" {
		t.Fatalf("auth fields = %q/%q", item.AuthMethod, item.AuthState)
	}
	if item.AuthAccountID != cred.UID || item.AuthPlan != cred.Plan || item.Name != cred.AccountName {
		t.Fatalf("identity lost: %+v", item)
	}
	// 令牌只进受保护存储：账号元数据里不能出现令牌原文。
	encoded, _ := json.Marshal(item)
	for _, secret := range []string{cred.AccessToken, cred.RefreshToken} {
		if strings.Contains(string(encoded), secret) {
			t.Fatal("credential escaped into account metadata")
		}
	}
	if len(store.values) != 1 {
		t.Fatalf("protected entries = %d, want 1", len(store.values))
	}

	// 同一远端身份重新登录只更新凭据，不重复建号。
	var count int
	if err := service.db.QueryRow("SELECT COUNT(*) FROM accounts WHERE provider_id=? AND auth_method=?", "builtin_workbuddy_ai", AuthWorkBuddyLogin).Scan(&count); err != nil {
		t.Fatal(err)
	}
	cred.AccessToken = "fake-rotated-access"
	again, err := service.SaveWorkBuddyAccount(ctx, "builtin_workbuddy_ai", "", "", cred)
	if err != nil {
		t.Fatalf("relogin: %v", err)
	}
	if again.ID != item.ID {
		t.Fatalf("relogin created another account: %q vs %q", again.ID, item.ID)
	}
	var after int
	if err := service.db.QueryRow("SELECT COUNT(*) FROM accounts WHERE provider_id=? AND auth_method=?", "builtin_workbuddy_ai", AuthWorkBuddyLogin).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != 1 {
		t.Fatalf("accounts after relogin = %d, want 1", after)
	}
}

// 普通续期只换令牌：目录必须跟着新凭据引用走，否则模型会静默消失。
func TestWorkBuddyRotationKeepsCatalogVersion(t *testing.T) {
	service, _ := testService(t)
	store := &batchStore{values: map[string][]byte{}}
	service.store = store
	ctx := context.Background()
	cred := workbuddyTestCredential()
	item, err := service.SaveWorkBuddyAccount(ctx, "builtin_workbuddy_ai", "", "", cred)
	if err != nil {
		t.Fatal(err)
	}
	version, err := service.CatalogVersion(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version) VALUES(?,'default-model','Default','now',?)`, item.ID, version); err != nil {
		t.Fatal(err)
	}

	cred.AccessToken = "fake-rotated-access"
	updated, err := service.RotateWorkBuddyCredentials(ctx, item, cred)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	newVersion, err := service.CatalogVersion(ctx, updated.ID)
	if err != nil {
		t.Fatal(err)
	}
	if newVersion == version {
		t.Fatal("rotation did not change the credential reference")
	}
	var bound string
	if err := service.db.QueryRow("SELECT account_version FROM account_models WHERE account_id=?", item.ID).Scan(&bound); err != nil {
		t.Fatal(err)
	}
	if bound != newVersion {
		t.Fatalf("catalog still bound to %q, want %q", bound, newVersion)
	}
	if len(store.values) != 1 {
		t.Fatalf("old protected entry not cleaned: %d", len(store.values))
	}
}

// 保存阶段的失败必须给出稳定码，不能和「登录失败」混成一个通用错误。
func TestWorkBuddySaveFailureIsRecognizable(t *testing.T) {
	service, _ := testService(t)
	store := &batchStore{values: map[string][]byte{}, failPut: 1}
	service.store = store
	_, err := service.SaveWorkBuddyAccount(context.Background(), "builtin_workbuddy_ai", "", "", workbuddyTestCredential())
	if code := workbuddyai.ErrorCode(err); code != workbuddyai.CodeCredentialUnavailable {
		t.Fatalf("protected storage failure code = %q (err %v)", code, err)
	}
}
