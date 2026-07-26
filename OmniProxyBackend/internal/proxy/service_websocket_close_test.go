package proxy

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"omniproxy/internal/config"
	"omniproxy/internal/logs"
	"omniproxy/internal/storage"
	"omniproxy/internal/token"
)

func TestServiceCloseTearsDownWebSocketBridges(t *testing.T) {
	bridged := make(chan struct{})
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		close(bridged)
		// Hold the session open so only an explicit teardown can end it.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer upstream.Close()

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "ws-close-tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(token.UpsertRequest{
		Name:           "coder@example.com",
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForServiceTestWithCredentials(t, "coder@example.com", "account-ws", "ws-access-token"),
	}); err != nil {
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
	local := httptest.NewServer(service)
	defer local.Close()

	dialURL := "ws" + strings.TrimPrefix(local.URL, "http") + "/backend-api/codex/responses"
	conn, _, err := websocket.DefaultDialer.Dial(dialURL, http.Header{
		"ChatGPT-Account-Id": []string{"account-ws"},
		"OpenAI-Beta":        []string{"responses_websockets=2026-02-06"},
		"Origin":             []string{"http://localhost:5173"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	select {
	case <-bridged:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out establishing the websocket bridge")
	}

	// Shutdown never touches hijacked connections, so Close is the only thing
	// that can end a bridge that is already running.
	service.Close()

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("expected Service.Close to end the client connection")
	}
}

func TestServiceCloseRejectsLaterSessions(t *testing.T) {
	service, err := NewService(config.Config{ProxyPort: 3000, ControlPort: 3890}, nil, logs.NewRecorder(10))
	if err != nil {
		t.Fatal(err)
	}
	service.Close()

	closed := &recordingCloser{}
	untrack := service.trackWebSocketSession(closed)
	defer untrack()

	// A connection that arrives after teardown must not be left running.
	if !closed.closed {
		t.Fatal("expected a session registered after Close to be closed immediately")
	}
}

type recordingCloser struct {
	closed bool
}

func (c *recordingCloser) Close() error {
	c.closed = true
	return nil
}
