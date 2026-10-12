package subscription

import (
	"context"
	"testing"
)

func TestCurrentResumesOnlyMatchingOperationWithoutCancelling(t *testing.T) {
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	resumed, err := f.manager.Current("builtin_openai", "")
	if err != nil || resumed.ID != view.ID || resumed.State != "waiting" {
		t.Fatal("current authorization not restored")
	}
	if _, err := f.manager.Current("builtin_openai", "other-account"); err == nil {
		t.Fatal("restored another account operation")
	}
	client.authorize("example@example.test")
	awaitState(t, f.manager, view.ID, "completed")
	empty, err := f.manager.Current("builtin_openai", "")
	if err != nil || empty.ID != "" {
		t.Fatal("completed authorization still active")
	}
	items, err := f.accounts.List(context.Background())
	if err != nil || len(items) != 1 {
		t.Fatal("background completion did not save account")
	}
}
