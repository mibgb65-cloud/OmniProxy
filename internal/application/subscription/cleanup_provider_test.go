package subscription

import (
	"context"
	"os"
	"testing"
)

func TestCodexLoginIgnoresOtherProvidersSubscriptionCredentials(t *testing.T) {
	for _, mode := range []string{"browser", "device"} {
		t.Run(mode, func(t *testing.T) {
			f := newFixture(t)
			first, original, originalHome := f.begin(t, "")
			original.authorize("existing@example.test")
			saved := awaitState(t, f.manager, first.ID, "completed")
			// 外部提供商引用故意不存在，Codex 不应尝试解密或修复它。
			_, err := f.db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method,auth_email,auth_state,auth_account_id) VALUES('foreign-google','builtin_google','Other provider','foreign-credential-ref',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z','subscription_login','google@example.invalid','authenticated','google-subject')`)
			if err != nil {
				t.Fatal(err)
			}
			second, err := f.manager.Begin(context.Background(), Input{ProviderID: "builtin_openai", Mode: mode})
			if err != nil || second.State != "waiting" {
				t.Fatal("new Codex authorization did not start", err)
			}
			f.mu.Lock()
			newClient := f.last
			f.mu.Unlock()
			newClient.authorize("new@example.test")
			awaitState(t, f.manager, second.ID, "completed")
			f.store.mu.Lock()
			for _, ref := range f.store.reads {
				if ref == "foreign-credential-ref" {
					t.Error("Codex accessed other provider credentials")
				}
			}
			f.store.mu.Unlock()
			if _, err = os.Stat(originalHome); err != nil {
				t.Fatal("existing Codex profile was removed")
			}
			metadata, err := f.accounts.Metadata(context.Background(), saved.Account.ID)
			if err != nil || metadata.AuthState != "authenticated" {
				t.Fatal("existing Codex login changed")
			}
			original.mu.Lock()
			defer original.mu.Unlock()
			if original.logoutCount != 0 {
				t.Fatal("existing account was logged out")
			}
		})
	}
}

func TestCodexLoginKeepsUnreadableOwnReferenceProtected(t *testing.T) {
	f := newFixture(t)
	_, err := f.db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method,auth_email,auth_state,auth_account_id) VALUES('own-unreadable','builtin_openai','Existing Codex','missing-own-reference',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z','subscription_login','existing@example.invalid','authenticated','existing-workspace')`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.manager.Begin(context.Background(), Input{ProviderID: "builtin_openai", Mode: "browser"}); ErrorCode(err) != "account_unavailable" {
		t.Fatal("unreadable own reference bypassed", err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.last != nil {
		t.Fatal("cleanup started without proving own references")
	}
}
