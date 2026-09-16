package proxy

import (
	"net/http"
	"net/http/httptest"
	"omniproxy/internal/config"
	"omniproxy/internal/logs"
	"omniproxy/internal/storage"
	"omniproxy/internal/token"
	"path/filepath"
	"testing"
)

func TestServiceRoutesAtriaRequests(t *testing.T) {
	var upstreamPath string
	var authorization string
	var apiKey string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath = r.URL.Path
		authorization = r.Header.Get("Authorization")
		apiKey = r.Header.Get("X-Api-Key")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"usage":{"total_tokens":7}}`))
	}))
	defer upstream.Close()

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(token.UpsertRequest{Name: "atria", Provider: token.ProviderAtria, TokenValue: "atr_test_token"}); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(config.Config{
		ProxyPort:       3000,
		ControlPort:     3890,
		AtriaBaseURL:    upstream.URL,
		SwitchThreshold: 15,
		MaxRetries:      0,
	}, manager, logs.NewRecorder(10))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("openai protocol", func(t *testing.T) {
		upstreamPath, authorization, apiKey = "", "", ""
		req := httptest.NewRequest(http.MethodPost, "/atria/v1/chat/completions", stringsReader(`{"model":"Atria-Dawn-Preview","messages":[]}`))
		req.Header.Set("Authorization", "Bearer caller")
		res := httptest.NewRecorder()
		service.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
		}
		if upstreamPath != "/v1/chat/completions" || authorization != "Bearer atr_test_token" {
			t.Fatalf("unexpected atria openai route path=%q authorization=%q", upstreamPath, authorization)
		}
	})

	t.Run("anthropic protocol", func(t *testing.T) {
		upstreamPath, authorization, apiKey = "", "", ""
		req := httptest.NewRequest(http.MethodPost, "/atria/anthropic/v1/messages", stringsReader(`{"model":"Atria-Dawn-Preview","messages":[]}`))
		req.Header.Set("Authorization", "Bearer caller")
		res := httptest.NewRecorder()
		service.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
		}
		if upstreamPath != "/v1/messages" || apiKey != "atr_test_token" {
			t.Fatalf("unexpected atria anthropic route path=%q x-api-key=%q", upstreamPath, apiKey)
		}
		if authorization != "" {
			t.Fatalf("expected anthropic protocol to drop Authorization header, got %q", authorization)
		}
	})
}
