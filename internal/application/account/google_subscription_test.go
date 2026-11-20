package account

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/provider/antigravity"
)

func googleTestCredential() antigravity.Credentials {
	return antigravity.Credentials{OAuthConfig: antigravity.OAuthConfig{ClientID: "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com", ClientSecret: "GOCSPX-" + strings.Repeat("x", 28)}, AccessToken: "fake-access-token", RefreshToken: "fake-refresh-token", Subject: "1001", Email: "account@example.test", Project: "test-project", Plan: "test-tier", ExpiresAt: time.Now().Add(time.Hour)}
}

func TestGoogleCredentialsStayProtectedAndRotationKeepsCatalog(t *testing.T) {
	service, _ := testService(t)
	store := &batchStore{values: map[string][]byte{}}
	service.store = store
	ctx := context.Background()
	cred := googleTestCredential()
	item, err := service.SaveGoogleSubscription(ctx, "builtin_google", "", "", cred)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(item)
	if strings.Contains(string(encoded), cred.AccessToken) || strings.Contains(string(encoded), cred.RefreshToken) || item.Name != cred.Email {
		t.Fatal("credential escaped or identity name missing")
	}
	version, err := service.CatalogVersion(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version) VALUES(?,'gemini-test','Gemini','now',?)`, item.ID, version)
	if err != nil {
		t.Fatal(err)
	}
	cred.AccessToken = "fake-rotated-access"
	updated, err := service.RotateGoogleCredentials(ctx, item, cred)
	if err != nil {
		t.Fatal(err)
	}
	newVersion, _ := service.CatalogVersion(ctx, item.ID)
	var bound string
	if service.db.QueryRow("SELECT account_version FROM account_models WHERE account_id=?", item.ID).Scan(&bound) != nil || bound != newVersion || newVersion == version {
		t.Fatal("refresh invalidated existing model snapshot")
	}
	if len(store.values) != 1 {
		t.Fatal("old credential was not cleaned")
	}
	if _, err := service.RotateGoogleCredentials(ctx, item, cred); err == nil {
		t.Fatal("stale update overwrote current token")
	}
	cred.Subject = "2002"
	if _, err := service.SaveGoogleSubscription(ctx, "builtin_google", item.ID, "", cred); antigravity.ErrorCode(err) != "identity_changed" {
		t.Fatal("relogin accepted another identity")
	}
	cred.Subject = updated.AuthAccountID
	if _, err := service.SaveGoogleSubscription(ctx, "builtin_google", item.ID, "", cred); err != nil {
		t.Fatal(err)
	}
	latestVersion, _ := service.CatalogVersion(ctx, item.ID)
	if bound == latestVersion {
		t.Fatal("new authorization retained old model grant")
	}
	if _, err := service.LogoutGoogleSubscription(ctx, item.ID); err != nil {
		t.Fatal(err)
	}
	if len(store.values) != 0 {
		t.Fatal("logout kept a credential")
	}
	if err := service.UseGoogleCredentials(ctx, item.ID, func(Item, antigravity.Credentials) error { t.Fatal("logged out token was consumed"); return nil }); err == nil {
		t.Fatal("logged out credential accepted")
	}
	if _, err := service.SaveGoogleSubscription(ctx, "builtin_openai", "", "", cred); err == nil {
		t.Fatal("Google credential entered Codex account")
	}
}

func TestGoogleLogoutCleanupFailureCanBeRetried(t *testing.T) {
	service, _ := testService(t)
	store := &batchStore{values: map[string][]byte{}}
	service.store = store
	ctx := context.Background()
	item, err := service.SaveGoogleSubscription(ctx, "builtin_google", "", "", googleTestCredential())
	if err != nil {
		t.Fatal(err)
	}
	store.failDelete = true
	if _, err := service.LogoutGoogleSubscription(ctx, item.ID); err == nil {
		t.Fatal("cleanup failure was swallowed")
	}
	item, err = service.Metadata(ctx, item.ID)
	if err != nil || item.AuthState != "logged_out" {
		t.Fatal("failed cleanup kept account active")
	}
	store.failDelete = false
	if _, err := service.LogoutGoogleSubscription(ctx, item.ID); err != nil || len(store.values) != 0 {
		t.Fatal("cleanup could not be retried")
	}
}

func TestGoogleFailedRotationCleanupIsRecoveredOnLogout(t *testing.T) {
	service, _ := testService(t)
	store := &batchStore{values: map[string][]byte{}}
	service.store = store
	ctx := context.Background()
	cred := googleTestCredential()
	item, err := service.SaveGoogleSubscription(ctx, "builtin_google", "", "", cred)
	if err != nil {
		t.Fatal(err)
	}
	store.failDelete = true
	updated, err := service.RotateGoogleCredentials(ctx, item, cred)
	if err == nil || updated.ID == "" || len(store.values) != 2 {
		t.Fatal("rotation cleanup failure was not retained")
	}
	store.failDelete = false
	if _, err := service.LogoutGoogleSubscription(ctx, item.ID); err != nil || len(store.values) != 0 {
		t.Fatal("retired credential was orphaned")
	}
}

func TestGoogleFreshLoginReusesExistingIdentity(t *testing.T) {
	service, _ := testService(t)
	service.store = &batchStore{values: map[string][]byte{}}
	ctx := context.Background()
	cred := googleTestCredential()
	first, err := service.SaveGoogleSubscription(ctx, "builtin_google", "", "", cred)
	if err != nil {
		t.Fatal(err)
	}
	// 用户先退出登录，再用「添加账号」重新授权同一个 Google 账号。
	if _, err := service.LogoutGoogleSubscription(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	cred.AccessToken = "fake-relogin-access"
	second, err := service.SaveGoogleSubscription(ctx, "builtin_google", "", "", cred)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatal("relogin created a duplicate account for the same identity")
	}
	if second.CreatedAt != first.CreatedAt || second.Name != first.Name || !second.Enabled {
		t.Fatal("identity reuse did not keep the original account")
	}
	if second.AuthState != "authenticated" || second.AuthAccountID != cred.Subject || second.AuthEmail != cred.Email {
		t.Fatal("identity reuse did not apply the new authorization")
	}
	if second.UpdatedAt == first.UpdatedAt {
		t.Fatal("identity reuse did not advance the update time")
	}
	var count int
	if err := service.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM accounts WHERE provider_id='builtin_google'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("google accounts after relogin = (%d, %v)", count, err)
	}
	item, ref, err := service.get(ctx, second.ID)
	if err != nil || ref == "" || item.AuthState != "authenticated" {
		t.Fatal("reused account lost its protected credential")
	}
}

func TestGoogleFreshLoginKeepsDistinctIdentity(t *testing.T) {
	service, _ := testService(t)
	service.store = &batchStore{values: map[string][]byte{}}
	ctx := context.Background()
	cred := googleTestCredential()
	first, err := service.SaveGoogleSubscription(ctx, "builtin_google", "", "", cred)
	if err != nil {
		t.Fatal(err)
	}
	other := cred
	other.Subject, other.Email = "2002", "other@example.test"
	second, err := service.SaveGoogleSubscription(ctx, "builtin_google", "", "", other)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID == first.ID {
		t.Fatal("different google identity reused an unrelated account")
	}
	// Team 工作区同 account_id 的不同成员邮箱是不同账号。
	member := cred
	member.Email = "member@example.test"
	third, err := service.SaveGoogleSubscription(ctx, "builtin_google", "", "", member)
	if err != nil {
		t.Fatal(err)
	}
	if third.ID == first.ID || third.ID == second.ID {
		t.Fatal("workspace member reused another mailbox account")
	}
}

func TestGoogleFreshLoginCompletesMissingIdentity(t *testing.T) {
	service, _ := testService(t)
	service.store = &batchStore{values: map[string][]byte{}}
	ctx := context.Background()
	cred := googleTestCredential()
	created, err := service.SaveGoogleSubscription(ctx, "builtin_google", "", "", cred)
	if err != nil {
		t.Fatal(err)
	}
	// 旧记录可能在 account_id 功能之前创建，登录时应按邮箱认领并补全。
	if _, err := service.db.ExecContext(ctx, `UPDATE accounts SET auth_account_id='' WHERE id=?`, created.ID); err != nil {
		t.Fatal(err)
	}
	again, err := service.SaveGoogleSubscription(ctx, "builtin_google", "", "", cred)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != created.ID {
		t.Fatal("legacy account without account_id was duplicated")
	}
	if again.AuthAccountID != cred.Subject || again.AuthState != "authenticated" {
		t.Fatal("relogin did not complete the missing identity")
	}
}
