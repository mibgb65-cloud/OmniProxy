package subscription

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/codex"
	"omniproxy/internal/infrastructure/database"
)

type testStore struct {
	mu    sync.Mutex
	refs  map[string][]byte
	next  int
	reads []string
}

func (s *testStore) Put(data []byte) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	ref := fmt.Sprintf("ref-%d", s.next)
	s.refs[ref] = append([]byte(nil), data...)
	return ref, nil
}
func (s *testStore) Get(ref string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reads = append(s.reads, ref)
	data, ok := s.refs[ref]
	if !ok {
		return nil, errors.New("missing")
	}
	return append([]byte(nil), data...), nil
}
func (s *testStore) Delete(ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.refs, ref)
	return nil
}

type testClient struct {
	mu                   sync.Mutex
	ready                chan struct{}
	identity             *codex.Account
	logoutCount          int
	failLogout           bool
	requireRefresh       bool
	metadataUnavailable  bool
	accountIDUnavailable bool
	rateLimits           *codex.RateLimits
	failQuota            bool
	resetKeys            []string
	resetCreditIDs       []string
	failReset            bool
	resetOutcome         string
	failQuotaAfterReset  bool
}

func (c *testClient) RateLimits(context.Context) (codex.RateLimits, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.failQuota {
		return codex.RateLimits{}, errors.New("private-quota-error")
	}
	if c.rateLimits != nil {
		return *c.rateLimits, nil
	}
	used := int32(0)
	five, week := int64(300), int64(10080)
	return codex.RateLimits{AccountID: "11111111-2222-3333-4444-555555555555", Snapshot: &codex.RateLimitSnapshot{Plan: "plus", Primary: &codex.RateLimitWindow{UsedPercent: &used, DurationMins: &five}, Secondary: &codex.RateLimitWindow{UsedPercent: &used, DurationMins: &week}}}, nil
}

func (c *testClient) AccountID(context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.accountIDUnavailable {
		return "", errors.New("subscription: request_failed")
	}
	return "11111111-2222-3333-4444-555555555555", nil
}

func (c *testClient) Begin(_ context.Context, mode string) (codex.Login, error) {
	return codex.Login{ID: "official-session", AuthURL: "https://auth.openai.com/authorize", VerificationURL: "https://auth.openai.com/codex/device", UserCode: "TEST-CODE"}, nil
}
func (c *testClient) Wait(ctx context.Context, _ string) error {
	select {
	case <-c.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (c *testClient) Account(_ context.Context, refresh bool) (*codex.Account, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.identity == nil || c.metadataUnavailable || c.requireRefresh && !refresh {
		return nil, nil
	}
	copy := *c.identity
	return &copy, nil
}
func (c *testClient) Cancel(context.Context, string) error { return nil }
func (c *testClient) Logout(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.failLogout {
		return errors.New("simulated credential failure")
	}
	c.identity = nil
	c.logoutCount++
	return nil
}
func (c *testClient) Close() {}
func (c *testClient) authorize(email string) {
	c.mu.Lock()
	c.identity = &codex.Account{Type: "chatgpt", Email: email, Plan: "plus"}
	c.mu.Unlock()
	close(c.ready)
}

type fixture struct {
	db       *sql.DB
	store    *testStore
	manager  *Service
	accounts *account.Service
	mu       sync.Mutex
	clients  map[string]*testClient
	last     *testClient
	lastHome string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "test.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	store := &testStore{refs: map[string][]byte{}}
	f := &fixture{db: db, store: store, accounts: account.NewService(db, store), clients: map[string]*testClient{}}
	f.manager = New(f.accounts, filepath.Join(t.TempDir(), "profiles"), func(_ context.Context, home string) (codex.Client, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		client := f.clients[home]
		if client == nil {
			client = &testClient{ready: make(chan struct{})}
			f.clients[home] = client
		}
		f.last, f.lastHome = client, home
		return client, nil
	})
	f.manager.removeCredentials = func(string) error { return nil }
	t.Cleanup(f.manager.Close)
	return f
}
func (f *fixture) begin(t *testing.T, accountID string) (State, *testClient, string) {
	t.Helper()
	view, err := f.manager.Begin(context.Background(), Input{ProviderID: "builtin_openai", Mode: "browser", AccountID: accountID})
	if err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return view, f.last, f.lastHome
}
func awaitState(t *testing.T, s *Service, id, state string) State {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		view, err := s.Read(id)
		if err != nil {
			t.Fatal(err)
		}
		if view.State == state {
			return view
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("session did not reach %s", state)
	return State{}
}
