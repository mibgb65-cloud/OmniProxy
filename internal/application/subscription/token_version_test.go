package subscription

import (
	"context"
	"testing"
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
