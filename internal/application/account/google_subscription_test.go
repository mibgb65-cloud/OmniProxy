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
