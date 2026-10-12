package subscription

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"omniproxy/internal/application/account"
)

func TestLoginAccountsAreIsolatedAndNeverExposeAPIKeys(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	first, one, homeOne := f.begin(t, "")
	one.authorize("one@example.test")
	savedOne := awaitState(t, f.manager, first.ID, "completed")
	second, two, homeTwo := f.begin(t, "")
	two.authorize("two@example.test")
	savedTwo := awaitState(t, f.manager, second.ID, "completed")
	if homeOne == homeTwo || savedOne.Account.ID == savedTwo.Account.ID {
		t.Fatal("accounts shared a profile")
	}
	if savedOne.URL != "" || savedOne.UserCode != "" || savedOne.Account.AuthMethod != account.AuthSubscription {
		t.Fatal("terminal state retained authorization data")
	}
	used := false
	if f.accounts.UseAPIKey(ctx, savedOne.Account.ID, func([]byte) error { used = true; return nil }) == nil || used {
		t.Fatal("subscription exposed as API key")
	}
	if f.accounts.Delete(ctx, savedOne.Account.ID) == nil {
		t.Fatal("unmanaged deletion allowed")
	}
	if err := f.manager.Delete(ctx, savedOne.Account.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(homeOne); !os.IsNotExist(err) {
		t.Fatal("deleted account profile remains")
	}
	identity, _ := two.Account(ctx, false)
	if identity == nil {
		t.Fatal("deleting one account logged out the other")
	}
}

func TestCancelAndDifferentIdentityKeepOriginalAuthorization(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	initial, original, originalHome := f.begin(t, "")
	original.authorize("original@example.test")
	item := awaitState(t, f.manager, initial.ID, "completed").Account
	reauth, _, stagedHome := f.begin(t, item.ID)
	if f.manager.Delete(ctx, item.ID) == nil {
		t.Fatal("deleted during reauthorization")
	}
	if _, err := f.manager.Cancel(reauth.ID); err != nil {
		t.Fatal(err)
	}
	awaitState(t, f.manager, reauth.ID, "cancelled")
	if _, err := os.Stat(stagedHome); !os.IsNotExist(err) {
		t.Fatal("cancelled profile remains")
	}
	if _, err := os.Stat(originalHome); err != nil {
		t.Fatal("original profile removed")
	}
	identity, _ := original.Account(ctx, false)
	if identity == nil {
		t.Fatal("original authorization lost")
	}
	wrong, other, _ := f.begin(t, item.ID)
	other.authorize("other@example.test")
	view := awaitState(t, f.manager, wrong.ID, "failed")
	if view.ErrorCode != "identity_changed" {
		t.Fatalf("unexpected error: %s", view.ErrorCode)
	}
	metadata, err := f.accounts.Metadata(ctx, item.ID)
	if err != nil || metadata.AuthEmail != "original@example.test" {
		t.Fatal("wrong identity replaced account")
	}
}

func TestReauthorizationRefreshAndLogoutLifecycle(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	first, old, oldHome := f.begin(t, "")
	old.authorize("one@example.test")
	item := awaitState(t, f.manager, first.ID, "completed").Account
	newLogin, newer, newHome := f.begin(t, item.ID)
	newer.authorize("one@example.test")
	view := awaitState(t, f.manager, newLogin.ID, "completed")
	if view.CleanupWarning || newHome == oldHome {
		t.Fatal("reauthorization failed to replace profile")
	}
	if _, err := os.Stat(oldHome); !os.IsNotExist(err) {
		t.Fatal("old profile remains")
	}
	refreshed, err := f.manager.Refresh(ctx, item.ID)
	if err != nil || refreshed.AuthState != "authenticated" || refreshed.AuthAccountID != item.AuthAccountID || refreshed.AuthAccountID == "" {
		t.Fatal("refresh failed")
	}
	loggedOut, err := f.manager.Logout(ctx, item.ID)
	if err != nil || loggedOut.AuthState != "logged_out" || loggedOut.AuthAccountID != item.AuthAccountID {
		t.Fatal("logout failed")
	}
	refreshed, err = f.manager.Refresh(ctx, item.ID)
	if err != nil || refreshed.AuthState != "reauth_required" {
		t.Fatal("missing credentials not detected")
	}
}

func TestFailedCleanupRetainsMarkerAndIsRetried(t *testing.T) {
	f := newFixture(t)
	view, client, home := f.begin(t, "")
	client.mu.Lock()
	client.failLogout = true
	client.mu.Unlock()
	_, _ = f.manager.Cancel(view.ID)
	failed := awaitState(t, f.manager, view.ID, "cancelled")
	if failed.ErrorCode != "cleanup_failed" {
		t.Fatal("cleanup error lost")
	}
	if _, err := os.Stat(filepath.Join(home, "omniproxy-pending")); err != nil {
		t.Fatal("cleanup location lost")
	}
	if _, err := f.manager.Begin(context.Background(), Input{ProviderID: "builtin_openai", Mode: "browser"}); err == nil {
		t.Fatal("started before orphan cleanup succeeded")
	}
	client.mu.Lock()
	client.failLogout = false
	client.mu.Unlock()
	newView, _, _ := f.begin(t, "")
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatal("orphan cleanup was not retried")
	}
	_, _ = f.manager.Cancel(newView.ID)
	awaitState(t, f.manager, newView.ID, "cancelled")
}

func TestInvalidInputsAndBusySessionDoNotStartAnotherProcess(t *testing.T) {
	f := newFixture(t)
	for _, input := range []Input{{ProviderID: "builtin_deepseek", Mode: "browser"}, {ProviderID: "builtin_openai", Mode: "invalid"}, {ProviderID: "builtin_openai", Mode: "browser", Name: "bad\x00name"}} {
		if _, err := f.manager.Begin(context.Background(), input); err == nil {
			t.Fatal("invalid input accepted")
		}
	}
	view, _, _ := f.begin(t, "")
	if _, err := f.manager.Begin(context.Background(), Input{ProviderID: "builtin_openai", Mode: "device"}); err == nil {
		t.Fatal("parallel login accepted")
	}
	if _, err := f.manager.profilePath("../outside"); err == nil {
		t.Fatal("path escape accepted")
	}
	_, _ = f.manager.Cancel(view.ID)
	awaitState(t, f.manager, view.ID, "cancelled")
}
