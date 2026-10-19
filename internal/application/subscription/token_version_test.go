package subscription

import (
	"context"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/codex"
)

func TestUseTokenAtRejectsChangedAccountBeforeCredentialUse(t *testing.T) {
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.authorize("first@example.invalid")
	saved := awaitState(t, f.manager, view.ID, "completed")
	called := false
	err := f.manager.UseTokenAt(context.Background(), saved.Account.ID, "stale-version", func([]byte, string) error { called = true; return nil })
	if err == nil || called {
		t.Fatal("old identity snapshot used")
	}
}

func TestUseTokenAtRechecksVersionAfterWaitingForPreparation(t *testing.T) {
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.authorize("queued@example.invalid")
	item := awaitState(t, f.manager, view.ID, "completed").Account
	if err := f.manager.operations.Lock(context.Background()); err != nil {
		t.Fatal(err)
	}
	locked := true
	defer func() {
		if locked {
			f.manager.operations.Unlock()
		}
	}()
	opened, consumed := false, false
	f.manager.factory = func(context.Context, string) (codex.Client, error) {
		opened = true
		return nil, codex.ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- f.manager.UseTokenAt(ctx, item.ID, item.UpdatedAt, func([]byte, string) error { consumed = true; return nil })
	}()
	updated, err := f.accounts.UpdateSubscription(ctx, item.ID, account.SubscriptionMetadata{
		Email: item.AuthEmail, AccountID: item.AuthAccountID, State: item.AuthState, Plan: "team",
	})
	if err != nil || updated.UpdatedAt == item.UpdatedAt {
		t.Fatal("could not prepare a changed identity version")
	}
	f.manager.operations.Unlock()
	locked = false
	select {
	case err := <-done:
		if ErrorCode(err) != "identity_changed" || opened || consumed {
			t.Fatal("queued request used an obsolete identity version")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("queued request did not complete")
	}
}
