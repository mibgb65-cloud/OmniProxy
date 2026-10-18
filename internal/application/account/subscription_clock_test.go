package account

import (
	"context"
	"strings"
	"testing"
)

func TestSubscriptionMutationsAdvancePersistedVersion(t *testing.T) {
	s, _ := testService(t)
	s.store = &batchStore{values: map[string][]byte{}}
	ctx := context.Background()
	meta := SubscriptionMetadata{Email: "test@example.invalid", Plan: "plus", State: "authenticated", AccountID: "account-test"}
	item, err := s.CreateSubscription(ctx, "builtin_openai", "test", strings.Repeat("a", 32), meta)
	if err != nil {
		t.Fatal(err)
	}
	item = pinFutureVersion(t, s, item.ID)
	renamed, err := s.Update(ctx, UpdateInput{ID: item.ID, Name: "renamed"})
	if err != nil {
		t.Fatal(err)
	}
	requireAdvancedVersion(t, item.UpdatedAt, renamed.Account.UpdatedAt)
	meta.Plan = "team"
	updated, err := s.UpdateSubscription(ctx, item.ID, meta)
	if err != nil {
		t.Fatal(err)
	}
	requireAdvancedVersion(t, renamed.Account.UpdatedAt, updated.UpdatedAt)
	replaced, err := s.ReplaceSubscription(ctx, item.ID, strings.Repeat("b", 32), meta)
	if err != nil {
		t.Fatal(err)
	}
	requireAdvancedVersion(t, updated.UpdatedAt, replaced.UpdatedAt)
}

func TestGoogleRotationAndLogoutAdvancePersistedVersion(t *testing.T) {
	s, _ := testService(t)
	s.store = &batchStore{values: map[string][]byte{}}
	ctx := context.Background()
	cred := googleTestCredential()
	item, err := s.SaveGoogleSubscription(ctx, "builtin_google", "", "test", cred)
	if err != nil {
		t.Fatal(err)
	}
	item = pinFutureVersion(t, s, item.ID)
	cred.AccessToken = "placeholder-refreshed-token"
	rotated, err := s.RotateGoogleCredentials(ctx, item, cred)
	if err != nil {
		t.Fatal(err)
	}
	requireAdvancedVersion(t, item.UpdatedAt, rotated.UpdatedAt)
	if _, err := s.RotateGoogleCredentials(ctx, item, cred); err == nil {
		t.Fatal("stale Google snapshot was accepted")
	}
	loggedOut, err := s.LogoutGoogleSubscription(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	requireAdvancedVersion(t, rotated.UpdatedAt, loggedOut.UpdatedAt)
	if loggedOut.AuthState != "logged_out" {
		t.Fatal("logout not saved")
	}
}
