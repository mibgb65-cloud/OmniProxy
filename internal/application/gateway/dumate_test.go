package gateway

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/application/provider"
	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/dumate"
)

// duMateStore 保留真实密文，Cookie 登录态才能在用例中解密使用。
type duMateStore struct {
	values  map[string][]byte
	counter int
}

func newDuMateStore() *duMateStore { return &duMateStore{values: map[string][]byte{}} }

func (s *duMateStore) Put(secret []byte) (string, error) {
	s.counter++
	ref := fmt.Sprintf("dumate-ref-%d", s.counter)
	s.values[ref] = append([]byte(nil), secret...)
	return ref, nil
}

func (s *duMateStore) Get(ref string) ([]byte, error) {
	value, ok := s.values[ref]
	if !ok {
		return nil, account.ErrNotFound
	}
	return append([]byte(nil), value...), nil
}

func (s *duMateStore) Delete(ref string) error {
	if _, ok := s.values[ref]; !ok {
		return account.ErrNotFound
	}
	delete(s.values, ref)
	return nil
}

type duMateFixture struct {
	service    *Service
	accounts   *account.Service
	db         *sql.DB
	tokenCalls atomic.Int32
	stop       func()
}

func (f *duMateFixture) Close() {
	if f.stop != nil {
		f.stop()
	}
}

func TestDuMateForwardsChatAndRecordsUsage(t *testing.T) {
	fixture := newDuMateFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer gateway-token" {
			t.Errorf("upstream request = %s %v", r.URL.Path, r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"chatcmpl-1","choices":[{"index":0,"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":11,"completion_tokens":3}}`)
	}))
	defer fixture.Close()
	response, err := http.Post(fixture.service.Info().Runtime.Address+"/v1/chat/completions", "application/json",
		strings.NewReader(`{"model":"model-text","messages":[{"role":"user","content":"hi"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), "OK") {
		t.Fatalf("gateway response = %d %s", response.StatusCode, body)
	}
	if fixture.tokenCalls.Load() != 1 {
		t.Fatalf("token calls = %d", fixture.tokenCalls.Load())
	}
}

func TestDuMateConvertsResponsesAndMessagesToChat(t *testing.T) {
	fixture := newDuMateFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(raw), `"messages"`) || strings.Contains(string(raw), `"input"`) {
			t.Errorf("converted body = %s", raw)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"chatcmpl-1","choices":[{"index":0,"message":{"role":"assistant","content":"converted"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2}}`)
	}))
	defer fixture.Close()
	cases := []struct{ path, body string }{
		{"/v1/responses", `{"model":"model-text","stream":false,"input":"hello"}`},
		{"/v1/messages", `{"model":"model-text","stream":false,"max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`},
	}
	for _, item := range cases {
		response, err := http.Post(fixture.service.Info().Runtime.Address+item.path, "application/json", strings.NewReader(item.body))
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if response.StatusCode != http.StatusOK || !strings.Contains(string(body), "converted") {
			t.Fatalf("%s = %d %s", item.path, response.StatusCode, body)
		}
	}
}

func TestDuMateTokenRejectionRefreshesBeforeNextRequest(t *testing.T) {
	var upstreamCalls int32
	fixture := newDuMateFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&upstreamCalls, 1) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"content":"after refresh"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`)
	}))
	defer fixture.Close()
	first, err := http.Post(fixture.service.Info().Runtime.Address+"/v1/chat/completions", "application/json",
		strings.NewReader(`{"model":"model-text","messages":[{"role":"user","content":"hi"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	first.Body.Close()
	if first.StatusCode != http.StatusUnauthorized {
		t.Fatalf("first status = %d", first.StatusCode)
	}
	second, err := http.Post(fixture.service.Info().Runtime.Address+"/v1/chat/completions", "application/json",
		strings.NewReader(`{"model":"model-text","messages":[{"role":"user","content":"hi"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer second.Body.Close()
	body, _ := io.ReadAll(second.Body)
	if second.StatusCode != http.StatusOK || !strings.Contains(string(body), "after refresh") {
		t.Fatalf("second = %d %s", second.StatusCode, body)
	}
	if fixture.tokenCalls.Load() != 2 {
		t.Fatalf("token calls = %d, want refresh after 401", fixture.tokenCalls.Load())
	}
}

func TestDuMateRejectsExpiredCookieAccount(t *testing.T) {
	fixture := newDuMateFixture(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"content":"OK"}}]}`)
	}))
	defer fixture.Close()
	if _, err := fixture.db.Exec(`UPDATE accounts SET auth_state='reauth_required' WHERE provider_id='builtin_dumate'`); err != nil {
		t.Fatal(err)
	}
	response, err := http.Post(fixture.service.Info().Runtime.Address+"/v1/chat/completions", "application/json",
		strings.NewReader(`{"model":"model-text","messages":[{"role":"user","content":"hi"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		t.Fatal("expired cookie account was used")
	}
}

func newDuMateFixture(t *testing.T, upstream http.HandlerFunc) *duMateFixture {
	t.Helper()
	server := httptest.NewServer(upstream)
	t.Cleanup(server.Close)
	fixture := &duMateFixture{}
	console := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/dumate/app/token/get" {
			t.Errorf("console path = %q", r.URL.Path)
		}
		fixture.tokenCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"code":0,"success":true,"result":{"token":"gateway-token","expired_at":"2030-01-02T03:04:05Z"}}`)
	}))
	t.Cleanup(console.Close)
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "gateway.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	providers, accounts := provider.NewService(db), account.NewService(db, newDuMateStore())
	if _, err := accounts.SaveDuMateAccount(context.Background(), "builtin_dumate", "", "", dumate.Credentials{
		Cookies: []dumate.Cookie{{Name: dumate.SessionCookie, Value: "session"}}, Source: dumate.SourceManual, AccountID: "1001",
	}); err != nil {
		t.Fatal(err)
	}
	observer := activity.New(db)
	service := New(proxy.Options{Observer: observer}, providers, accounts, nil)
	consoleClient, err := dumate.NewConsoleAt(console.URL, console.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	chatClient, err := dumate.NewClientAt(server.URL, server.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	service.dumateConsole, service.dumateClient = consoleClient, chatClient
	service.SetRouter(routes.New(db, providers, accounts))
	if _, err := service.Start(); err != nil {
		t.Fatal(err)
	}
	fixture.service, fixture.accounts, fixture.db = service, accounts, db
	fixture.stop = func() { _, _ = service.Stop(context.Background()) }
	return fixture
}
