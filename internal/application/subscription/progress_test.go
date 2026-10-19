package subscription

import (
	"context"
	"reflect"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/codex"
)

type blockingSaveStore struct {
	*testStore
	started chan struct{}
	release chan struct{}
}

func (s *blockingSaveStore) Put(value []byte) (string, error) {
	close(s.started)
	<-s.release
	return s.testStore.Put(value)
}

type blockingAccountClient struct {
	codex.Client
	release <-chan struct{}
}

func (c blockingAccountClient) Account(ctx context.Context, refresh bool) (*codex.Account, error) {
	select {
	case <-c.release:
		return c.Client.Account(ctx, refresh)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestAuthorizationReportsActualProgressAndResumesDuringVerification(t *testing.T) {
	f := newFixture(t)
	release := make(chan struct{})
	factory := f.manager.factory
	f.manager.factory = func(ctx context.Context, home string) (codex.Client, error) {
		client, err := factory(ctx, home)
		return blockingAccountClient{client, release}, err
	}
	events := make(chan string, 10)
	f.manager.OnChange(func() { events <- f.manager.DiagnosticState().State })
	view, client, _ := f.begin(t, "")
	client.authorize("progress@example.test")
	verifying := awaitState(t, f.manager, view.ID, "verifying")
	if verifying.URL != "" || verifying.UserCode != "" || verifying.Account != nil {
		t.Fatal("verification retained consumed authorization data")
	}
	current, err := f.manager.Current("builtin_openai", "")
	if err != nil || current.State != "verifying" {
		t.Fatal("processing authorization could not resume")
	}
	close(release)
	awaitState(t, f.manager, view.ID, "completed")
	var states []string
	for len(states) < 4 {
		select {
		case state := <-events:
			states = append(states, state)
		case <-time.After(time.Second):
			t.Fatalf("missing progress notifications: %v", states)
		}
	}
	if !reflect.DeepEqual(states, []string{"waiting", "verifying", "saving", "completed"}) {
		t.Fatalf("unexpected stages: %v", states)
	}
}

func TestCancelDuringVerificationDoesNotSaveAccount(t *testing.T) {
	f := newFixture(t)
	release := make(chan struct{})
	factory := f.manager.factory
	f.manager.factory = func(ctx context.Context, home string) (codex.Client, error) {
		client, err := factory(ctx, home)
		return blockingAccountClient{client, release}, err
	}
	view, client, _ := f.begin(t, "")
	client.authorize("cancel@example.test")
	awaitState(t, f.manager, view.ID, "verifying")
	if _, err := f.manager.Cancel(view.ID); err != nil {
		t.Fatal(err)
	}
	close(release)
	awaitState(t, f.manager, view.ID, "cancelled")
	items, err := f.accounts.List(context.Background())
	if err != nil || len(items) != 0 {
		t.Fatal("cancelled authorization saved an account")
	}
}

func TestSavingProgressRemainsReadableDuringCredentialIO(t *testing.T) {
	f := newFixture(t)
	store := &blockingSaveStore{testStore: f.store, started: make(chan struct{}), release: make(chan struct{})}
	f.manager.accounts = account.NewService(f.db, store)
	view, client, _ := f.begin(t, "")
	client.authorize("saving@example.test")
	defer close(store.release)
	select {
	case <-store.started:
	case <-time.After(3 * time.Second):
		t.Fatal("credential save did not start")
	}
	result := make(chan State, 1)
	go func() {
		state, _ := f.manager.Read(view.ID)
		result <- state
	}()
	select {
	case state := <-result:
		if state.State != "saving" {
			t.Fatal("in-flight save not reported")
		}
	case <-time.After(time.Second):
		t.Fatal("credential IO blocked progress reads")
	}
}
