package gateway

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
)

type testStore struct {
	last []byte
	fail bool
}

func (s *testStore) Put([]byte) (string, error) { return "test-ref", nil }
func (s *testStore) Delete(string) error        { return nil }
func (s *testStore) Get(string) ([]byte, error) {
	if s.fail {
		return nil, errors.New("private credential failure")
	}
	s.last = []byte("placeholder-gateway-test-key")
	return s.last, nil
}

func setupService(t *testing.T, handler http.Handler) (*Service, *sql.DB, *testStore) {
	t.Helper()
	upstream := httptest.NewServer(handler)
	t.Cleanup(upstream.Close)
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "gateway.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	_, err = db.Exec(`CREATE TABLE providers(id TEXT PRIMARY KEY,kind TEXT,source TEXT,name TEXT,base_url TEXT,enabled INTEGER,created_at TEXT,updated_at TEXT);
	CREATE TABLE accounts(id TEXT PRIMARY KEY,provider_id TEXT,name TEXT,credential_ref TEXT,enabled INTEGER,created_at TEXT,updated_at TEXT,auth_method TEXT DEFAULT 'api_key',auth_email TEXT DEFAULT '',auth_plan TEXT DEFAULT '',auth_state TEXT DEFAULT '',auth_account_id TEXT DEFAULT '');
	INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at) VALUES('acct_first','builtin_deepseek','name must not be logged','test-ref',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z');`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO providers VALUES('builtin_deepseek','deepseek','builtin','DeepSeek',?,1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`, upstream.URL); err != nil {
		t.Fatal(err)
	}
	store := &testStore{}
	stepSchema, err := os.ReadFile("../../../migrations/021_stepfun_account_options.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(stepSchema)); err != nil {
		t.Fatal(err)
	}
	service := New(proxy.Options{}, provider.NewService(db), account.NewService(db, store), nil)
	if _, err := service.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if _, err := service.Stop(ctx); err != nil {
			t.Error(err)
		}
	})
	return service, db, store
}

func successfulHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer placeholder-gateway-test-key" {
		w.WriteHeader(401)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"choices":[{"message":{"content":"OK"}}]}`)
}

func TestFullLocalRequestAndProtectedKeyLifecycle(t *testing.T) {
	service, _, store := setupService(t, http.HandlerFunc(successfulHandler))
	status, body := localGatewayRequest(t, service, "deepseek-flash", false)
	if status != http.StatusOK || !gatewayReplyHasContent(body, false) {
		t.Fatalf("invalid gateway response: status=%d", status)
	}
	for _, value := range store.last {
		if value != 0 {
			t.Fatal("protected key bytes remain after callback")
		}
	}
}

func TestFreshAccountEligibilityAndCredentialErrors(t *testing.T) {
	service, db, store := setupService(t, http.HandlerFunc(successfulHandler))
	if _, err := db.Exec(`UPDATE accounts SET enabled=0`); err != nil {
		t.Fatal(err)
	}
	status, body := localGatewayRequest(t, service, "deepseek-flash", false)
	if code := gatewayResponseError(t, body); code != "no_enabled_deepseek_account" || status != http.StatusServiceUnavailable {
		t.Fatalf("disabled account used: status=%d code=%s", status, code)
	}
	if _, err := db.Exec(`UPDATE accounts SET enabled=1`); err != nil {
		t.Fatal(err)
	}
	store.fail = true
	_, body = localGatewayRequest(t, service, "deepseek-flash", false)
	if code := gatewayResponseError(t, body); code != "credential_unavailable" {
		t.Fatalf("credential failure: code=%s", code)
	}
	if _, err := db.Exec(`UPDATE providers SET enabled=0`); err != nil {
		t.Fatal(err)
	}
	_, body = localGatewayRequest(t, service, "deepseek-flash", false)
	if code := gatewayResponseError(t, body); code != "no_enabled_deepseek_provider" {
		t.Fatalf("disabled provider used: code=%s", code)
	}
}

func TestStreamingGatewayForwardsContentAndDone(t *testing.T) {
	service, _, _ := setupService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"OK\"}}]}\n\ndata: [DONE]\n\n")
	}))
	status, body := localGatewayRequest(t, service, "deepseek-flash", true)
	if status != http.StatusOK || !gatewayReplyHasContent(body, true) {
		t.Fatalf("invalid streaming gateway response: status=%d", status)
	}
}

func TestGatewayRejectsUnsupportedModelAndStopsListener(t *testing.T) {
	service, _, _ := setupService(t, http.HandlerFunc(successfulHandler))
	_, body := localGatewayRequest(t, service, "unsupported-model", false)
	if code := gatewayResponseError(t, body); code != "unsupported_model" {
		t.Fatalf("unsupported model accepted: code=%s", code)
	}
	if _, err := service.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if service.Info().Runtime.State != "stopped" {
		t.Fatal("gateway listener did not stop")
	}
}
