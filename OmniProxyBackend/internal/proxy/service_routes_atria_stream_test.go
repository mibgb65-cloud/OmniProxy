package proxy

import (
	"net/http"
	"net/http/httptest"
	"omniproxy/internal/config"
	"omniproxy/internal/history"
	"omniproxy/internal/logs"
	"omniproxy/internal/storage"
	"omniproxy/internal/token"
	"path/filepath"
	"testing"
)

func TestServiceRoutesAtriaStreamingUsage(t *testing.T) {
	var upstreamPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath = r.URL.Path
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		if upstreamPath == "/v1/messages" {
			_, _ = w.Write([]byte("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":210,\"output_tokens\":1}}}\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
			_, _ = w.Write([]byte("event: message_delta\ndata: {\"type\":\"message_delta\",\"usage\":{\"input_tokens\":210,\"output_tokens\":90}}\n\n"))
			_, _ = w.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
		} else {
			_, _ = w.Write([]byte("event: response.created\ndata: {\"type\":\"response.created\",\"response\":{}}\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
			_, _ = w.Write([]byte("event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":210,\"output_tokens\":90}}}\n\n"))
		}
	}))
	defer upstream.Close()

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(token.UpsertRequest{Name: "atria", Provider: token.ProviderAtria, TokenValue: "atr_test_token"}); err != nil {
		t.Fatal(err)
	}
	recorder, err := history.NewRecorder(storage.NewJSONStore[[]history.Entry](filepath.Join(t.TempDir(), "history.json")), 100)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(config.Config{
		ProxyPort:       3000,
		ControlPort:     3890,
		AtriaBaseURL:    upstream.URL,
		SwitchThreshold: 15,
		MaxRetries:      0,
	}, manager, logs.NewRecorder(10), recorder)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("claude router sse", func(t *testing.T) {
		upstreamPath = ""
		req := httptest.NewRequest(http.MethodPost, "/anthropic-router/v1/messages", stringsReader(`{"model":"Atria-Dawn-Preview","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
		req.Header.Set("Authorization", "Bearer caller")
		res := httptest.NewRecorder()
		service.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
		}
		entry := lastHistoryEntry(recorder)
		// Anthropic streams report input at message_start and output at
		// message_delta. Atria repeats input_tokens in the delta, so the
		// last parsed usage carries both.
		if entry.InputTokens != 210 || entry.OutputTokens != 90 || entry.TotalTokens != 300 {
			t.Fatalf("expected sse usage from final delta, got %#v", entry)
		}
	})

	t.Run("codex responses sse", func(t *testing.T) {
		upstreamPath = ""
		req := httptest.NewRequest(http.MethodPost, "/codex/v1/responses", stringsReader(`{"model":"Atria-Dawn-Preview","stream":true,"input":"hi"}`))
		req.Header.Set("Authorization", "Bearer caller")
		res := httptest.NewRecorder()
		service.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
		}
		entry := lastHistoryEntry(recorder)
		if entry.InputTokens != 210 || entry.OutputTokens != 90 || entry.TotalTokens != 300 {
			t.Fatalf("expected codex sse usage recorded, got %#v", entry)
		}
	})
}
