package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"omniproxy/internal/config"
	"omniproxy/internal/logs"
	"omniproxy/internal/storage"
	"omniproxy/internal/token"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceRoutesCodexResponsesToDeepSeek(t *testing.T) {
	var gotPath string
	var gotAuthorization string
	var gotBody string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		gotPath = r.URL.Path
		gotAuthorization = r.Header.Get("Authorization")
		gotBody = string(body)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: response.output_text.delta`,
			`data: {"type":"response.output_text.delta","delta":"done"}`,
			``,
			`event: response.completed`,
			`data: {"type":"response.completed","response":{"model":"deepseek-v4-flash","usage":{"input_tokens":12,"output_tokens":4,"total_tokens":16,"input_tokens_details":{"cached_tokens":6}}}}`,
			``,
		}, "\n")))
	}))
	defer upstream.Close()

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "deepseek-codex-tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	item, err := manager.Add(token.UpsertRequest{
		Name:           "deepseek",
		Provider:       token.ProviderDeepSeek,
		CredentialType: token.CredentialTypeAPIKey,
		TokenValue:     "sk-deepseek-codex",
	})
	if err != nil {
		t.Fatal(err)
	}

	service, err := NewService(config.Config{
		ProxyPort:       3000,
		ControlPort:     3890,
		DeepSeekBaseURL: upstream.URL,
		GatewayRoutes: config.GatewayRoutes{
			Codex: config.GatewayRouteConfig{Provider: token.ProviderOpenAI, Model: "gpt-5.6-sol"},
		},
		SwitchThreshold: 15,
		MaxRetries:      0,
	}, manager, logs.NewRecorder(10))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/codex/v1/responses", stringsReader(`{
		"model":"deepseek-v4-flash",
		"input":"apply the patch",
		"stream":true,
		"tools":[{"type":"custom","name":"apply_patch"}]
	}`))
	req.Header.Set("Authorization", "Bearer caller")
	res := httptest.NewRecorder()
	service.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
	}
	if gotPath != "/v1/responses" {
		t.Fatalf("expected DeepSeek Responses path, got %q", gotPath)
	}
	if gotAuthorization != "Bearer sk-deepseek-codex" {
		t.Fatalf("expected DeepSeek API key injection, got %q", gotAuthorization)
	}
	if !strings.Contains(gotBody, `"model":"deepseek-v4-flash"`) || !strings.Contains(gotBody, `"name":"apply_patch"`) {
		t.Fatalf("expected Codex Responses body to pass through, got %s", gotBody)
	}
	if !strings.Contains(res.Body.String(), "response.output_text.delta") || !strings.Contains(res.Body.String(), "response.completed") {
		t.Fatalf("expected DeepSeek SSE events to pass through, got %s", res.Body.String())
	}
	updated, err := manager.Get(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Stats.InputTokens != 12 || updated.Stats.OutputTokens != 4 || updated.Stats.TotalTokens != 16 || updated.Stats.CacheReadTokens != 6 {
		t.Fatalf("unexpected DeepSeek Responses usage: %#v", updated.Stats)
	}
}
