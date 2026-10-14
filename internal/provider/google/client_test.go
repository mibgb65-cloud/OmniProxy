package google

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/core/proxy"
)

func TestChatPreservesExtensionsAndUsage(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(map[bool]string{false: "json", true: "sse"}[stream], func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1beta/openai/chat/completions" || r.URL.RawQuery != "" || r.Header.Get("Authorization") != "Bearer test-key" {
					t.Error("incorrect request destination or credential transport")
				}
				var body map[string]any
				if json.NewDecoder(r.Body).Decode(&body) != nil || body["extra_body"] == nil || body["reasoning_effort"] != "high" {
					t.Error("Gemini extensions lost")
				}
				if stream {
					options := body["stream_options"].(map[string]any)
					if options["include_usage"] != true || options["other"] != "kept" {
						t.Error("stream options lost")
					}
					w.Header().Set("Content-Type", "text/event-stream")
					io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
					w.(http.Flusher).Flush()
					select {
					case <-r.Context().Done():
						return
					case <-time.After(2 * time.Second):
					}
				} else {
					io.WriteString(w, `{"choices":[{"message":{"tool_calls":[{"extra_content":{"google":{"thought_signature":"opaque"}}}]}}],"usage":{"prompt_tokens":3,"completion_tokens":1}}`)
				}
			}))
			defer upstream.Close()
			raw, _ := json.Marshal(map[string]any{"model": "gemini-test", "messages": []any{map[string]any{"role": "user", "content": "hi"}}, "stream": stream, "stream_options": map[string]any{"other": "kept"}, "reasoning_effort": "high", "extra_body": map[string]any{"google": map[string]any{"thinking_config": map[string]any{"thinking_level": "high"}}}})
			request, err := proxy.ParseProtocolRequest(raw, "/v1/chat/completions")
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			started := time.Now()
			err = New(nil).Forward(ctx, upstream.URL+"/v1beta", []byte("test-key"), request, nil, func(response proxy.Response) error {
				if stream {
					first := make([]byte, 5)
					if _, err := io.ReadFull(response.Body, first); err != nil || string(first) != "data:" {
						t.Fatal("stream did not arrive incrementally")
					}
					if time.Since(started) > time.Second {
						t.Error("stream buffered until completion")
					}
					cancel()
				} else {
					data, err := io.ReadAll(response.Body)
					if err != nil || !strings.Contains(string(data), "thought_signature") || !strings.Contains(string(data), "prompt_tokens") {
						t.Error("response extensions/usage lost")
					}
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestValidationAndRedirectDoNotForwardCredential(t *testing.T) {
	for _, base := range []string{"http://example.com", "https://user:password@example.com", "https://example.com/?key=secret", "https://example.com/#fragment", "https://example.com:65536", "https://example.com:0"} {
		if _, err := endpoint(base, "/models"); err == nil {
			t.Errorf("invalid configuration accepted: %s", base)
		}
	}
	for _, field := range []string{`"max_tokens":0`, `"max_tokens":1.5`, `"max_tokens":"3"`, `"max_completion_tokens":-1`, `"stream_options":[]`} {
		raw := []byte(`{"model":"gemini-test","messages":[{"role":"user","content":"hi"}],"stream":true,` + field + `}`)
		request, err := proxy.ParseProtocolRequest(raw, "/v1/chat/completions")
		if err == nil {
			_, err = chatBody(request)
		}
		if err == nil {
			t.Error("invalid chat field accepted", field)
		}
	}
	called := false
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer receiver.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, receiver.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	if _, err := New(nil).Models(context.Background(), redirect.URL, []byte("test-key")); err == nil || called {
		t.Fatal("redirect followed or discovery accepted")
	}
}

func TestFailureSanitizesAndDoesNotGuessDailyQuota(t *testing.T) {
	raw := []byte(`{"error":{"type":"quota_exceeded","message":"private-key","details":[{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"1.1s"},{"@type":"type.googleapis.com/google.rpc.QuotaFailure","violations":[{"quotaMetric":"generativelanguage.googleapis.com/request_count","quotaId":"RequestsPerMinute","quotaValue":"15"},{"quotaMetric":"generativelanguage.googleapis.com/token_count","quotaId":"TokensPerDay","quotaValue":"999"}]}]}}`)
	item := ParseFailure(429, "", raw)
	if item.Category != "rate_limited" || item.RetrySeconds != 2 || item.Limit == nil || *item.Limit != 15 {
		t.Fatalf("metric association or retry incorrect: %+v", item)
	}
	encoded, _ := json.Marshal(item)
	if strings.Contains(string(encoded), "private-key") {
		t.Fatal("raw upstream message retained")
	}
	daily := ParseFailure(429, "", []byte(strings.ReplaceAll(string(raw), "RequestsPerMinute", "RequestsPerDay")))
	if daily.Category != "daily_exhausted" {
		t.Fatal("daily metric not recognized")
	}
	if ParseFailure(402, "", nil).Category != "payment_required" || ParseFailure(403, "", nil).Category != "permission_denied" {
		t.Fatal("status classification incorrect")
	}
}
