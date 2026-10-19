package subscription

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"omniproxy/internal/infrastructure/codex"
)

func TestDeleteDoesNotStartOfficialComponentOrLogout(t *testing.T) {
	f := newFixture(t)
	view, client, home := f.begin(t, "")
	client.authorize("delete@example.test")
	item := awaitState(t, f.manager, view.ID, "completed").Account
	client.mu.Lock()
	client.failLogout = true
	client.mu.Unlock()
	f.manager.factory = func(context.Context, string) (codex.Client, error) {
		t.Fatal("deletion started the official component")
		return nil, codex.ErrUnavailable
	}
	var removed []string
	f.manager.removeCredentials = func(path string) error {
		removed = append(removed, path)
		return nil
	}
	if err := f.manager.Delete(context.Background(), item.ID); err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0] != home {
		t.Fatal("deletion did not clean the isolated credential")
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatal("local profile remains")
	}
	if _, err := f.accounts.Metadata(context.Background(), item.ID); err == nil {
		t.Fatal("account metadata remains")
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.logoutCount != 0 || client.identity == nil {
		t.Fatal("deletion signed out the official account")
	}
}

func TestDeleteLocalCleanupFailureRetainsAccountForRetry(t *testing.T) {
	f := newFixture(t)
	view, client, home := f.begin(t, "")
	client.authorize("retry@example.test")
	item := awaitState(t, f.manager, view.ID, "completed").Account
	f.manager.removeCredentials = func(string) error { return errors.New("private-store-error") }
	err := f.manager.Delete(context.Background(), item.ID)
	if err == nil || err.Error() != "subscription: cleanup_failed" {
		t.Fatal("local cleanup failure was swallowed or exposed")
	}
	if _, _, err := f.accounts.SubscriptionProfile(context.Background(), item.ID); err != nil {
		t.Fatal("failed cleanup lost its credential reference")
	}
	if _, err := os.Stat(home); err != nil {
		t.Fatal("failed cleanup removed the profile")
	}
	f.manager.removeCredentials = func(string) error { return nil }
	if err := f.manager.Delete(context.Background(), item.ID); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteDoesNotRecreateMissingProfile(t *testing.T) {
	f := newFixture(t)
	view, client, home := f.begin(t, "")
	client.authorize("missing@example.test")
	item := awaitState(t, f.manager, view.ID, "completed").Account
	// 测试只删除 fixture 创建且已核实位于受控根目录下的随机授权目录。
	if parent := filepath.Dir(home); parent != f.manager.root {
		t.Fatal("profile escaped fixture root")
	}
	if err := os.RemoveAll(home); err != nil {
		t.Fatal(err)
	}
	if err := f.manager.Delete(context.Background(), item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatal("deletion recreated the missing profile")
	}
}
