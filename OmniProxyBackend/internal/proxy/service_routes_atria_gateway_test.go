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

// TestServiceRoutesAtriaViaOneClickGateways covers the exact paths the
// one-click configuration writes: /anthropic-router/v1/messages for Claude
// Code and /codex/v1/responses for Codex, both inferred to the atria
// provider purely from the Atria-Dawn-Preview model name.
func TestServiceRoutesAtriaViaOneClickGateways(t *testing.T) {
	var upstreamPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Atria documents input_tokens/output_tokens for messages and responses.
		_, _ = w.Write([]byte(`{"model":"Atria-Dawn-Preview","usage":{"input_tokens":210,"output_tokens":90}}`))
	}))
	defer upstream.Close()

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	item, err := manager.Add(token.UpsertRequest{Name: "atria", Provider: token.ProviderAtria, TokenValue: "atr_test_token"})
	if err != nil {
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

	t.Run("claude router", func(t *testing.T) {
		upstreamPath = ""
		req := httptest.NewRequest(http.MethodPost, "/anthropic-router/v1/messages", stringsReader(`{"model":"Atria-Dawn-Preview","messages":[{"role":"user","content":"hi"}]}`))
		req.Header.Set("Authorization", "Bearer caller")
		res := httptest.NewRecorder()
		service.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
		}
		if upstreamPath != "/v1/messages" {
			t.Fatalf("expected claude router to reach /v1/messages, got %q", upstreamPath)
		}
		entry := lastHistoryEntry(recorder)
		if entry.Provider != token.ProviderAtria {
			t.Fatalf("expected provider atria, got %q", entry.Provider)
		}
		if entry.InputTokens != 210 || entry.OutputTokens != 90 || entry.TotalTokens != 300 {
			t.Fatalf("expected claude router usage recorded, got %#v", entry)
		}
		updated, err := manager.Get(item.ID)
		if err != nil {
			t.Fatal(err)
		}
		if updated.Stats.TotalTokens != 300 || updated.Stats.RequestCount != 1 {
			t.Fatalf("expected token stats accumulated, got %#v", updated.Stats)
		}
	})

	t.Run("codex responses", func(t *testing.T) {
		upstreamPath = ""
		req := httptest.NewRequest(http.MethodPost, "/codex/v1/responses", stringsReader(`{"model":"Atria-Dawn-Preview","input":"hi"}`))
		req.Header.Set("Authorization", "Bearer caller")
		res := httptest.NewRecorder()
		service.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
		}
		if upstreamPath != "/v1/responses" {
			t.Fatalf("expected codex route to reach /v1/responses, got %q", upstreamPath)
		}
		entry := lastHistoryEntry(recorder)
		if entry.Provider != token.ProviderAtria {
			t.Fatalf("expected provider atria, got %q", entry.Provider)
		}
		if entry.InputTokens != 210 || entry.OutputTokens != 90 || entry.TotalTokens != 300 {
			t.Fatalf("expected codex usage recorded, got %#v", entry)
		}
	})
}
