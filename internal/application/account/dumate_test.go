package account

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"omniproxy/internal/provider/dumate"
)

func dumateTestCredential(subject string) dumate.Credentials {
	return dumate.Credentials{
		Cookies:   []dumate.Cookie{{Name: dumate.SessionCookie, Value: "session-token-" + subject, Domain: ".baidu.com"}},
		Source:    dumate.SourceManual,
		AccountID: subject,
		Name:      "Tester",
	}
}

func TestDuMateAccountKeepsCookieOutOfMetadataAndPreview(t *testing.T) {
	service, _ := testService(t)
	store := &batchStore{values: map[string][]byte{}}
	service.store = store
	ctx := context.Background()
	item, err := service.SaveDuMateAccount(ctx, "builtin_dumate", "", "", dumateTestCredential("1001"))
	if err != nil {
		t.Fatal(err)
	}
	if item.AuthMethod != AuthDuMateCookie || item.AuthState != "authenticated" || item.AuthAccountID != "1001" || item.Name != "Tester" {
		t.Fatalf("item = %+v", item)
	}
	encoded, _ := json.Marshal(item)
	if strings.Contains(string(encoded), "session-token") {
		t.Fatal("cookie escaped into account metadata")
	}
	var ref string
	if err := service.db.QueryRow(`SELECT credential_ref FROM accounts WHERE id=?`, item.ID).Scan(&ref); err != nil || strings.Contains(ref, "session-token") {
		t.Fatalf("credential reference = %q err=%v", ref, err)
	}
	if _, err := service.KeyPreview(ctx, item.ID); err == nil {
		t.Fatal("cookie account produced an API key preview")
	}
	if err := service.UseAPIKey(ctx, item.ID, func([]byte) error { return nil }); err == nil {
		t.Fatal("cookie account accepted API key access")
	}
	called := false
	if err := service.UseDuMateCredentials(ctx, item.ID, func(got Item, credentials dumate.Credentials) error {
		called = true
		if got.ID != item.ID || credentials.AccountID != "1001" || len(credentials.Cookies) != 1 {
			t.Fatalf("credentials = %+v item=%+v", credentials, got)
		}
		return nil
	}); err != nil || !called {
		t.Fatalf("use = %v called=%v", err, called)
	}
}

func TestDuMateReimportRotatesCredentialAndKeepsIdentity(t *testing.T) {
	service, _ := testService(t)
	store := &batchStore{values: map[string][]byte{}}
	service.store = store
	ctx := context.Background()
	item, err := service.SaveDuMateAccount(ctx, "builtin_dumate", "", "", dumateTestCredential("1001"))
	if err != nil {
		t.Fatal(err)
	}
	firstRef := ""
	if err := service.db.QueryRow(`SELECT credential_ref FROM accounts WHERE id=?`, item.ID).Scan(&firstRef); err != nil {
		t.Fatal(err)
	}
	rotated := dumateTestCredential("1001")
	rotated.Cookies[0].Value = "session-token-rotated"
	updated, err := service.SaveDuMateAccount(ctx, "builtin_dumate", "", "", rotated)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != item.ID || len(store.values) != 1 || len(store.deleted) != 1 || store.deleted[0] != firstRef {
		t.Fatalf("rotate = %+v refs=%v deleted=%v", updated, len(store.values), store.deleted)
	}
	if _, err := service.SaveDuMateAccount(ctx, "builtin_dumate", item.ID, "", dumateTestCredential("2002")); dumate.ErrorCode(err) != "identity_changed" {
		t.Fatalf("identity change = %v", err)
	}
	if _, err := service.SaveDuMateAccount(ctx, "builtin_dumate", "", "", dumateTestCredential("2002")); err != nil {
		t.Fatal(err)
	}
	items, err := service.List(ctx)
	if err != nil || len(items) != 2 {
		t.Fatalf("accounts = %v %v", len(items), err)
	}
}

func TestDuMateAccountRequiresProviderAndBoundIdentity(t *testing.T) {
	service, _ := testService(t)
	store := &batchStore{values: map[string][]byte{}}
	service.store = store
	ctx := context.Background()
	if _, err := service.SaveDuMateAccount(ctx, "builtin_openai", "", "", dumateTestCredential("1001")); err == nil {
		t.Fatal("cookie credential entered another provider")
	}
	if _, err := service.SaveDuMateAccount(ctx, "builtin_dumate", "", "", dumate.Credentials{Source: dumate.SourceManual, AccountID: "1001"}); err == nil {
		t.Fatal("credential without cookies accepted")
	}
	item, err := service.SaveDuMateAccount(ctx, "builtin_dumate", "", "", dumateTestCredential("1001"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.db.Exec(`UPDATE accounts SET auth_account_id='other' WHERE id=?`, item.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.UseDuMateCredentials(ctx, item.ID, func(Item, dumate.Credentials) error { return nil }); err == nil {
		t.Fatal("misbound credential was consumed")
	}
}

func TestDuMateAccountRenameAndDelete(t *testing.T) {
	service, _ := testService(t)
	store := &batchStore{values: map[string][]byte{}}
	service.store = store
	ctx := context.Background()
	item, err := service.SaveDuMateAccount(ctx, "builtin_dumate", "", "", dumateTestCredential("1001"))
	if err != nil {
		t.Fatal(err)
	}
	renamed, err := service.Update(ctx, UpdateInput{ID: item.ID, Name: "我的搭子"})
	if err != nil || renamed.Account.Name != "我的搭子" || len(store.values) != 1 {
		t.Fatalf("rename = (%+v, %v) credentials=%d", renamed.Account, err, len(store.values))
	}
	if _, err := service.Update(ctx, UpdateInput{ID: item.ID, APIKey: "sk-should-not-be-accepted"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("API key replacement = %v", err)
	}
	if err := service.Delete(ctx, item.ID); err != nil || len(store.values) != 0 {
		t.Fatalf("delete = %v credentials=%d", err, len(store.values))
	}
}
