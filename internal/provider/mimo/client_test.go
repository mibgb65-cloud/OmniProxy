package mimo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

func testRequest(protocol string, stream bool) proxy.Request {
	input := `"messages":[{"role":"user","content":"hello"}]`
	if protocol == "messages" {
		input += `,"max_tokens":16`
	}
	if protocol == "responses" {
		input = `"input":"hello","store":false`
	}
	return proxy.Request{Model: "mimo-v2.6-pro", Protocol: protocol, Stream: stream, Body: []byte(fmt.Sprintf(`{"model":"mimo-v2.6-pro",%s,"stream":%t}`, input, stream))}
}

func TestNativeEndpointsAuthenticationAndResponses(t *testing.T) {
	paths := map[string]string{"chat": "/v1/chat/completions", "responses": "/v1/responses", "messages": "/anthropic/v1/messages"}
	for protocol, path := range paths {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", protocol, stream), func(t *testing.T) {
				request := testRequest(protocol, stream)
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != path || r.Method != http.MethodPost {
						t.Errorf("unexpected endpoint: %s %s", r.Method, r.URL.Path)
					}
					if r.Header.Get("Api-Key") != "not-a-real-mimo-key" || r.Header.Get("Authorization") != "" || r.Header.Get("X-Api-Key") != "" {
						t.Error("incorrect authentication")
					}
					var body map[string]json.RawMessage
					if json.NewDecoder(r.Body).Decode(&body) != nil || string(body["model"]) != `"mimo-v2.6-pro"` || string(body["stream"]) != fmt.Sprint(stream) {
						t.Error("request contract lost")
					}
					if stream && protocol == "chat" && !strings.Contains(string(body["stream_options"]), `"include_usage":true`) {
						t.Error("usage not requested")
					}
					w.Header().Set("Content-Type", "application/json")
					if stream {
						w.Header().Set("Content-Type", "text/event-stream")
					}
					w.Header().Set("Retry-After", "42")
					w.WriteHeader(http.StatusTooManyRequests)
					fmt.Fprint(w, "native-response")
				}))
				defer upstream.Close()
				err := New(nil).Forward(context.Background(), upstream.URL+"/v1/", []byte("not-a-real-mimo-key"), request, func(response proxy.Response) error {
					body, err := io.ReadAll(response.Body)
					if response.StatusCode != http.StatusTooManyRequests || response.RetryAfter != "42" || string(body) != "native-response" {
						t.Error("response metadata changed")
					}
					return err
				})
				if err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestEndpointCredentialRedirectAndCancellationSafety(t *testing.T) {
	for _, base := range []string{"https://api.xiaomimimo.com.evil.test/v1", "https://user:pass@api.xiaomimimo.com/v1", "https://api.xiaomimimo.com/v1?key=private", "https://api.xiaomimimo.com:8443/v1", "http://api.xiaomimimo.com/v1", "https://token-plan-cn.xiaomimimo.com/v1", "https://api.xiaomimimo.com/v1/../other", "http://127.0.0.1:0/v1", "https://api.xiaomimimo.com/v1#fragment"} {
		if _, err := endpoint(base, "/v1/models"); err == nil {
			t.Fatal("unsafe endpoint accepted", base)
		}
	}
	for _, base := range []string{"", "https://api.xiaomimimo.com", DefaultBaseURL, DefaultBaseURL + "/"} {
		url, err := endpoint(base, "/anthropic/v1/messages")
		if err != nil || url != "https://api.xiaomimimo.com/anthropic/v1/messages" {
			t.Fatal("incorrect Messages URL", url, err)
		}
	}
	for _, key := range []string{"", "key\nheader", strings.Repeat("k", 8193)} {
		if err := New(nil).Forward(context.Background(), "", []byte(key), testRequest("chat", false), func(proxy.Response) error { t.Error("invalid key consumed"); return nil }); err == nil {
			t.Fatal("invalid credential accepted")
		}
	}
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("credential followed redirect") }))
	defer target.Close()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer upstream.Close()
	if err := New(nil).Forward(context.Background(), upstream.URL, []byte("test-key"), testRequest("chat", false), func(response proxy.Response) error {
		if response.StatusCode != http.StatusTemporaryRedirect {
			t.Error("redirect status changed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := New(nil).Forward(ctx, upstream.URL, []byte("test-key"), testRequest("chat", false), func(proxy.Response) error { t.Error("cancelled request consumed"); return nil }); err == nil {
		t.Fatal("cancellation ignored")
	}
}
