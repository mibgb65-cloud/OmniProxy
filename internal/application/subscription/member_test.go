package subscription

import (
	"context"
	"testing"
)

func TestSharedTeamWorkspaceRejectsDifferentEmail(t *testing.T) {
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.authorize("first@example.invalid")
	saved := awaitState(t, f.manager, view.ID, "completed")
	client.mu.Lock()
	client.identity.Email = "other@example.invalid"
	client.mu.Unlock()
	if _, err := f.manager.ReadQuota(context.Background(), saved.Account.ID); err == nil {
		t.Fatal("different Team member read quota")
	}
	called := false
	if err := f.manager.UseToken(context.Background(), saved.Account.ID, func([]byte, string) error { called = true; return nil }); err == nil || called {
		t.Fatal("different Team member used credential")
	}
	if _, err := f.manager.Refresh(context.Background(), saved.Account.ID); err == nil {
		t.Fatal("refresh silently reassigned member")
	}
}
