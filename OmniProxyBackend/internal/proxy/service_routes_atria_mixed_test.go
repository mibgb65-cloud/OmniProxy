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

// TestServiceRoutesAtriaAlongsideAnthropicToken makes sure the
// Atria-Dawn-Preview model still routes to (and records usage under) the
// atria provider when an official anthropic token is also configured, which
// is the normal setup after running the Claude Code one-click config.
func TestServiceRoutesAtriaAlongsideAnthropicToken(t *testing.T) {
	var hitProvider string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitProvider = "atria"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":210,"output_tokens":90}}`))
	}))
	defer upstream.Close()
	anthropicUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitProvider = "anthropic"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer anthropicUpstream.Close()

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(token.UpsertRequest{Name: "official", Provider: token.ProviderAnthropic, TokenValue: "sk-ant-official"}); err != nil {
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
		ProxyPort:        3000,
		ControlPort:      3890,
		AtriaBaseURL:     upstream.URL,
		AnthropicBaseURL: anthropicUpstream.URL,
		SwitchThreshold:  15,
		MaxRetries:       0,
	}, manager, logs.NewRecorder(10), recorder)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/anthropic-router/v1/messages", stringsReader(`{"model":"Atria-Dawn-Preview","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Authorization", "Bearer caller")
	res := httptest.NewRecorder()
	service.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
	}
	if hitProvider != "atria" {
		t.Fatalf("expected request to reach atria upstream, got %q", hitProvider)
	}
	entry := lastHistoryEntry(recorder)
	if entry.Provider != token.ProviderAtria {
		t.Fatalf("expected history provider atria, got %q (token=%q)", entry.Provider, entry.TokenName)
	}
	if entry.InputTokens != 210 || entry.OutputTokens != 90 || entry.TotalTokens != 300 {
		t.Fatalf("expected usage recorded, got %#v", entry)
	}
}
