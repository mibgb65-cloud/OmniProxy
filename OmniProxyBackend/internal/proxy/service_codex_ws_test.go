package proxy

import (
	"github.com/gorilla/websocket"
	"net/http"
	"net/http/httptest"
	"omniproxy/internal/config"
	"omniproxy/internal/logs"
	"omniproxy/internal/storage"
	"omniproxy/internal/token"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestServiceDoesNotRefreshAPIKeyQuotaAfterTask(t *testing.T) {
	quotaRefreshCalled := make(chan struct{}, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/responses":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"usage":{"input_tokens":1,"output_tokens":2,"total_tokens":3}}`))
		case "/v1/models":
			quotaRefreshCalled <- struct{}{}
			w.Header().Set("x-ratelimit-remaining-tokens", "64")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
		default:
			t.Fatalf("unexpected upstream path: %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := manager.Flush(); err != nil {
			t.Errorf("flush token manager: %v", err)
		}
	})
	item, err := manager.Add(token.UpsertRequest{Name: "primary", Provider: token.ProviderOpenAI, TokenValue: "sk-primary-token"})
	if err != nil {
		t.Fatal(err)
	}

	service, err := NewService(config.Config{
		ProxyPort:       3000,
		ControlPort:     3890,
		OpenAIBaseURL:   upstream.URL,
		SwitchThreshold: 15,
		MaxRetries:      0,
	}, manager, logs.NewRecorder(10))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", stringsReader(`{"input":"hello"}`))
	res := httptest.NewRecorder()
	service.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
	}

	select {
	case <-quotaRefreshCalled:
		t.Fatal("post-task API key quota refresh should not run from the proxy service")
	case <-time.After(150 * time.Millisecond):
	}

	updated, err := manager.Get(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Remaining != 100 || updated.Usage.APIRemaining != 0 {
		t.Fatalf("expected quota to remain unchanged until scheduled refresh, got remaining=%d usage=%#v", updated.Remaining, updated.Usage)
	}
}

func TestServiceIgnoresIncomingCodexAccountIDForGatewayScheduling(t *testing.T) {
	var gotAuth string
	var gotAccount string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccount = r.Header.Get("ChatGPT-Account-Id")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"usage":{"input_tokens":4,"output_tokens":2,"total_tokens":6}}`))
	}))
	defer upstream.Close()

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	preferred, err := manager.Add(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForServiceTestWithCredentials(t, "preferred@example.com", "account-preferred", "preferred-access-token"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.RecordUsage(preferred.ID, 8); err != nil {
		t.Fatal(err)
	}
	_, err = manager.Add(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForServiceTestWithCredentials(t, "other@example.com", "account-other", "other-access-token"),
	})
	if err != nil {
		t.Fatal(err)
	}

	service, err := NewService(config.Config{
		ProxyPort:          3000,
		ControlPort:        3890,
		CodexBaseURL:       upstream.URL + "/backend-api/codex",
		SwitchThreshold:    15,
		MaxRetries:         0,
		CodexUsageEndpoint: "https://chatgpt.com/backend-api/wham/usage",
	}, manager, logs.NewRecorder(10))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/backend-api/codex/v1/responses", stringsReader(`{"input":"hello"}`))
	req.Header.Set("Authorization", "Bearer caller-access-token")
	req.Header.Set("ChatGPT-Account-Id", "account-preferred")
	res := httptest.NewRecorder()
	service.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
	}
	if gotAuth != "Bearer other-access-token" {
		t.Fatalf("expected gateway-selected account auth, got %q", gotAuth)
	}
	if gotAccount != "account-other" {
		t.Fatalf("expected gateway-selected account id, got %q", gotAccount)
	}
}

func TestServiceClearsStaleIncomingCodexAccountID(t *testing.T) {
	var gotAuth string
	var gotAccount string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccount = r.Header.Get("ChatGPT-Account-Id")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`))
	}))
	defer upstream.Close()

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.Add(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForServiceTestWithCredentials(t, "selected@example.com", "", "selected-access-token"),
	})
	if err != nil {
		t.Fatal(err)
	}

	service, err := NewService(config.Config{
		ProxyPort:          3000,
		ControlPort:        3890,
		CodexBaseURL:       upstream.URL + "/backend-api/codex",
		SwitchThreshold:    15,
		MaxRetries:         0,
		CodexUsageEndpoint: "https://chatgpt.com/backend-api/wham/usage",
	}, manager, logs.NewRecorder(10))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/backend-api/codex/v1/responses", stringsReader(`{"input":"hello"}`))
	req.Header.Set("Authorization", "Bearer caller-access-token")
	req.Header.Set("ChatGPT-Account-Id", "last-login-account")
	res := httptest.NewRecorder()
	service.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
	}
	if gotAuth != "Bearer selected-access-token" {
		t.Fatalf("expected selected account auth, got %q", gotAuth)
	}
	if gotAccount != "" {
		t.Fatalf("expected stale incoming account id to be cleared, got %q", gotAccount)
	}
}

func TestServiceBalancedSchedulingChoosesHigherRemainingCodexAccount(t *testing.T) {
	var gotAuth string
	var gotAccount string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccount = r.Header.Get("ChatGPT-Account-Id")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"usage":{"input_tokens":4,"output_tokens":2,"total_tokens":6}}`))
	}))
	defer upstream.Close()

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "balanced-codex-tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	lower, err := manager.Add(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForServiceTestWithCredentials(t, "lower@example.com", "account-lower", "lower-access-token"),
	})
	if err != nil {
		t.Fatal(err)
	}
	higher, err := manager.Add(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForServiceTestWithCredentials(t, "higher@example.com", "account-higher", "higher-access-token"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.RecordUsage(lower.ID, 35); err != nil {
		t.Fatal(err)
	}
	if err := manager.RecordUsage(higher.ID, 90); err != nil {
		t.Fatal(err)
	}

	service, err := NewService(config.Config{
		ProxyPort:          3000,
		ControlPort:        3890,
		SchedulingMode:     config.SchedulingModeBalanced,
		CodexBaseURL:       upstream.URL + "/backend-api/codex",
		SwitchThreshold:    15,
		MaxRetries:         0,
		CodexUsageEndpoint: "https://chatgpt.com/backend-api/wham/usage",
	}, manager, logs.NewRecorder(10))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/backend-api/codex/v1/responses", stringsReader(`{"input":"hello"}`))
	req.Header.Set("ChatGPT-Account-Id", "account-lower")
	res := httptest.NewRecorder()
	service.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
	}
	if gotAuth != "Bearer higher-access-token" {
		t.Fatalf("expected balanced scheduler to choose higher quota auth, got %q", gotAuth)
	}
	if gotAccount != "account-higher" {
		t.Fatalf("expected balanced scheduler to choose higher quota account, got %q", gotAccount)
	}
}

func TestServiceProxiesCodexResponsesWebSocket(t *testing.T) {
	type capture struct {
		path    string
		auth    string
		account string
		beta    string
		origin  string
		message string
		err     error
	}
	captures := make(chan capture, 1)
	releaseUpstream := make(chan struct{})
	defer func() {
		select {
		case <-releaseUpstream:
		default:
			close(releaseUpstream)
		}
	}()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			captures <- capture{err: err}
			return
		}
		defer conn.Close()

		_, payload, err := conn.ReadMessage()
		if err != nil {
			captures <- capture{err: err}
			return
		}
		captures <- capture{
			path:    r.URL.Path,
			auth:    r.Header.Get("Authorization"),
			account: r.Header.Get("ChatGPT-Account-Id"),
			beta:    r.Header.Get("OpenAI-Beta"),
			origin:  r.Header.Get("Origin"),
			message: string(payload),
		}
		responsePayload := `{"type":"response.completed","response":{"model":"gpt-5.6-sol","usage":{"input_tokens":11,"output_tokens":7,"total_tokens":18}}}`
		if err := conn.WriteMessage(websocket.TextMessage, []byte(responsePayload)); err != nil {
			t.Errorf("failed to write upstream websocket response: %v", err)
		}
		<-releaseUpstream
	}))
	defer upstream.Close()

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "ws-tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	item, err := manager.Add(token.UpsertRequest{
		Name:           "coder@example.com",
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForServiceTestWithCredentials(t, "coder@example.com", "account-ws", "ws-access-token"),
	})
	if err != nil {
		t.Fatal(err)
	}

	logRecorder := logs.NewRecorder(10)
	service, err := NewService(config.Config{
		ProxyPort:          3000,
		ControlPort:        3890,
		CodexBaseURL:       upstream.URL + "/backend-api/codex",
		SwitchThreshold:    15,
		MaxRetries:         0,
		CodexUsageEndpoint: "https://chatgpt.com/backend-api/wham/usage",
	}, manager, logRecorder)
	if err != nil {
		t.Fatal(err)
	}
	local := httptest.NewServer(service)
	defer local.Close()

	dialURL := "ws" + strings.TrimPrefix(local.URL, "http") + "/backend-api/codex/responses"
	headers := http.Header{
		"ChatGPT-Account-Id": []string{"account-ws"},
		"OpenAI-Beta":        []string{"responses_websockets=2026-02-06"},
		"Origin":             []string{"http://localhost:5173"},
	}
	conn, _, err := websocket.DefaultDialer.Dial(dialURL, headers)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"response.create","response":{"model":"gpt-5.6-sol"}}`)); err != nil {
		t.Fatal(err)
	}
	messageType, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.TextMessage || string(payload) != `{"type":"response.completed","response":{"model":"gpt-5.6-sol","usage":{"input_tokens":11,"output_tokens":7,"total_tokens":18}}}` {
		t.Fatalf("unexpected websocket response type=%d payload=%q", messageType, string(payload))
	}

	activeDeadline := time.After(2 * time.Second)
	for {
		active := service.ActiveRequests()
		if len(active) == 1 && active[0].Model == "gpt-5.6-sol" {
			break
		}
		select {
		case <-activeDeadline:
			t.Fatalf("expected active websocket request model gpt-5.6-sol, got %#v", active)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	var got capture
	select {
	case got = <-captures:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for upstream websocket capture")
	}
	if got.err != nil {
		t.Fatal(got.err)
	}
	if got.path != "/backend-api/codex/responses" {
		t.Fatalf("unexpected upstream websocket path: %s", got.path)
	}
	if got.auth != "Bearer ws-access-token" {
		t.Fatalf("unexpected websocket Authorization header: %q", got.auth)
	}
	if got.account != "account-ws" {
		t.Fatalf("unexpected websocket account header: %q", got.account)
	}
	if got.beta != "responses_websockets=2026-02-06" {
		t.Fatalf("expected websocket beta header to be preserved, got %q", got.beta)
	}
	if got.origin != "" {
		t.Fatalf("expected local browser origin not to be forwarded upstream, got %q", got.origin)
	}
	if got.message != `{"type":"response.create","response":{"model":"gpt-5.6-sol"}}` {
		t.Fatalf("expected websocket message to be proxied, got %q", got.message)
	}

	deadline := time.After(2 * time.Second)
	for {
		updated, err := manager.Get(item.ID)
		if err != nil {
			t.Fatal(err)
		}
		if updated.Stats.RequestCount == 1 && updated.Stats.TotalTokens == 18 {
			if updated.Stats.InputTokens != 11 || updated.Stats.OutputTokens != 7 {
				t.Fatalf("unexpected websocket token stats: %#v", updated.Stats)
			}
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for websocket token stats: %#v", updated.Stats)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	close(releaseUpstream)
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	_ = conn.Close()

	deadline = time.After(2 * time.Second)
	for {
		entries := logRecorder.List()
		if len(entries) > 0 {
			if entries[len(entries)-1].Model != "gpt-5.6-sol" {
				t.Fatalf("expected websocket model from response message, got %#v", entries[len(entries)-1])
			}
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for websocket log entry")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestServiceRejectsCodexResponsesWebSocketFromNonLocalOrigin(t *testing.T) {
	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "ws-origin-tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(config.Config{
		ProxyPort:          3000,
		ControlPort:        3890,
		CodexBaseURL:       "https://chatgpt.com/backend-api/codex",
		SwitchThreshold:    15,
		MaxRetries:         0,
		CodexUsageEndpoint: "https://chatgpt.com/backend-api/wham/usage",
	}, manager, logs.NewRecorder(10))
	if err != nil {
		t.Fatal(err)
	}
	local := httptest.NewServer(service)
	defer local.Close()

	dialURL := "ws" + strings.TrimPrefix(local.URL, "http") + "/backend-api/codex/responses"
	conn, resp, err := websocket.DefaultDialer.Dial(dialURL, http.Header{
		"Origin": []string{"https://evil.example"},
	})
	if conn != nil {
		_ = conn.Close()
	}
	if err == nil {
		t.Fatal("expected websocket dial to fail for non-local origin")
	}
	if resp == nil {
		t.Fatal("expected handshake response")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", resp.StatusCode)
	}
}

func TestServiceRejectsCodexResponsesWebSocketWhenDisabled(t *testing.T) {
	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "ws-disabled-tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(config.Config{
		ProxyPort:       3000,
		ControlPort:     3890,
		CodexBaseURL:    "https://chatgpt.com/backend-api/codex",
		SwitchThreshold: 15,
		MaxRetries:      0,
		WebSocketMode:   config.WebSocketModeDisabled,
	}, manager, logs.NewRecorder(10))
	if err != nil {
		t.Fatal(err)
	}
	local := httptest.NewServer(service)
	defer local.Close()

	dialURL := "ws" + strings.TrimPrefix(local.URL, "http") + "/backend-api/codex/responses"
	conn, resp, err := websocket.DefaultDialer.Dial(dialURL, http.Header{
		"OpenAI-Beta": []string{"responses_websockets=2026-02-06"},
	})
	if conn != nil {
		_ = conn.Close()
	}
	if err == nil {
		t.Fatal("expected websocket dial to fail when disabled")
	}
	if resp == nil {
		t.Fatal("expected handshake response")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", resp.StatusCode)
	}
}

func TestServiceHandlesCodexResponsesProbe(t *testing.T) {
	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "codex-probe-tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(config.Config{
		ProxyPort:       3000,
		ControlPort:     3890,
		CodexBaseURL:    "https://chatgpt.com/backend-api/codex",
		SwitchThreshold: 15,
		MaxRetries:      0,
	}, manager, logs.NewRecorder(10))
	if err != nil {
		t.Fatal(err)
	}

	headRes := httptest.NewRecorder()
	service.ServeHTTP(headRes, httptest.NewRequest(http.MethodHead, "/backend-api/codex/v1/responses", nil))
	if headRes.Code != http.StatusOK {
		t.Fatalf("expected HEAD codex probe status 200, got %d body=%s", headRes.Code, headRes.Body.String())
	}
	if headRes.Body.Len() != 0 {
		t.Fatalf("expected HEAD codex probe to have empty body, got %q", headRes.Body.String())
	}

	getRes := httptest.NewRecorder()
	service.ServeHTTP(getRes, httptest.NewRequest(http.MethodGet, "/backend-api/codex/responses", nil))
	if getRes.Code != http.StatusOK {
		t.Fatalf("expected GET codex probe status 200, got %d body=%s", getRes.Code, getRes.Body.String())
	}
	if !strings.Contains(getRes.Body.String(), `"ok":true`) {
		t.Fatalf("expected GET codex probe health body, got %q", getRes.Body.String())
	}
}
