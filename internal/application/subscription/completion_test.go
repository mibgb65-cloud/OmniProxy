package subscription

import (
	"context"
	"os"
	"testing"
)

func TestMissingAccountIDPreservesAuthorizationAndRefreshRecoversIt(t *testing.T) {
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.mu.Lock()
	client.accountIDUnavailable = true
	client.mu.Unlock()
	client.authorize("example@example.test")
	result := awaitState(t, f.manager, view.ID, "completed")
	if result.Account == nil || result.Account.AuthState != "authenticated" || result.ErrorCode != "account_id_pending" {
		t.Fatal("optional account ID failure lost successful authorization")
	}
	client.mu.Lock()
	if client.logoutCount != 0 {
		t.Fatal("account ID failure logged out the account")
	}
	client.accountIDUnavailable = false
	client.mu.Unlock()
	updated, err := f.manager.Refresh(context.Background(), result.Account.ID)
	if err != nil || updated.AuthAccountID == "" || updated.AuthState != "authenticated" {
		t.Fatal("refresh did not recover account ID")
	}
	client.mu.Lock()
	client.accountIDUnavailable = true
	client.mu.Unlock()
	retained, err := f.manager.Refresh(context.Background(), result.Account.ID)
	if err != nil || retained.AuthAccountID != updated.AuthAccountID || retained.AuthState != "authenticated" {
		t.Fatal("known account ID unnecessarily depended on the optional network read")
	}
}

func TestCompletedAuthorizationSynchronizesAccountCache(t *testing.T) {
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.mu.Lock()
	client.requireRefresh = true
	client.mu.Unlock()
	client.authorize("example@example.test")
	result := awaitState(t, f.manager, view.ID, "completed")
	if result.Account == nil || result.Account.AuthState != "authenticated" || result.ErrorCode != "" {
		t.Fatal("completed login read a stale account snapshot")
	}
}

func TestAccountMetadataFailurePreservesSuccessfulAuthorization(t *testing.T) {
	f := newFixture(t)
	view, client, home := f.begin(t, "")
	client.mu.Lock()
	client.metadataUnavailable = true
	client.mu.Unlock()
	client.authorize("example@example.test")
	result := awaitState(t, f.manager, view.ID, "completed")
	if result.Account == nil || result.Account.AuthState != "unknown" || result.ErrorCode != "account_metadata_pending" {
		t.Fatal("metadata failure did not report a saved pending account")
	}
	client.mu.Lock()
	count := client.logoutCount
	client.metadataUnavailable = false
	client.mu.Unlock()
	if count != 0 {
		t.Fatal("successful authorization was logged out on metadata read failure")
	}
	if _, err := os.Stat(home); err != nil {
		t.Fatal("saved authorization profile removed")
	}
	updated, err := f.manager.Refresh(context.Background(), result.Account.ID)
	if err != nil || updated.AuthState != "authenticated" || updated.AuthEmail != "example@example.test" {
		t.Fatal("saved authorization could not recover metadata")
	}
}
