package subscription

import (
	"context"
	"os"
	"testing"
)

func TestReloginWithoutAccountIDReusesSubscriptionAccount(t *testing.T) {
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.authorize("example@example.test")
	first := awaitState(t, f.manager, view.ID, "completed")
	if first.Account == nil || first.Account.ID == "" {
		t.Fatalf("first login did not save an account: %+v", first)
	}
	// 前端「添加账号」不携带 accountId；再次授权同一官方身份只能更新原记录。
	view, client, _ = f.begin(t, "")
	client.authorize("example@example.test")
	second := awaitState(t, f.manager, view.ID, "completed")
	if second.Account == nil || second.Account.ID != first.Account.ID {
		t.Fatalf("relogin did not reuse the existing account: %+v", second)
	}
	if second.Account.CreatedAt != first.Account.CreatedAt || second.Account.AuthState != "authenticated" {
		t.Fatal("identity reuse did not keep the original account metadata")
	}
	var count int
	if err := f.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM accounts WHERE provider_id='builtin_openai'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("openai subscription accounts after relogin = (%d, %v)", count, err)
	}
	// 旧隔离配置由既有清理路径移除，只保留当前授权。
	_, profiles, err := f.accounts.SubscriptionProfiles(context.Background(), first.Account.ID)
	if err != nil || len(profiles) != 1 {
		t.Fatalf("retired profiles after relogin = (%v, %v)", profiles, err)
	}
}

func TestIdentityLookupFailureKeepsSuccessfulAuthorization(t *testing.T) {
	f := newFixture(t)
	view, client, home := f.begin(t, "")
	client.mu.Lock()
	// 身份读取返回错误而不是空身份：这次登录拿不到订阅身份，不能因此丢弃已完成的授权。
	client.metadataFailure = true
	client.mu.Unlock()
	client.authorize("example@example.test")
	result := awaitState(t, f.manager, view.ID, "completed")
	if result.Account == nil || result.ErrorCode != "account_metadata_pending" {
		t.Fatalf("metadata failure result = %+v", result)
	}
	var count int
	if err := f.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM accounts WHERE provider_id='builtin_openai'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("saved accounts after metadata failure = (%d, %v)", count, err)
	}
	if _, err := os.Stat(home); err != nil {
		t.Fatal("saved authorization profile removed")
	}
}
