package dumate

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

func testGatewayClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClientAt(server.URL, server.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func proxyCode(err error) string {
	var failure *proxy.Error
	if errors.As(err, &failure) {
		return failure.Code
	}
	return ""
}

func TestClientForwardsChatWithBearerToken(t *testing.T) {
	client := testGatewayClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token-1" || r.Header.Get("X-Api-Key") != "" {
			t.Errorf("headers = %v", r.Header)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"model":"model-text"`) {
			t.Errorf("body = %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"content":"OK"}}],"usage":{"prompt_tokens":1,"completion_tokens":2}}`)
	})
	request := proxy.Request{Protocol: "chat", Model: "model-text", Body: []byte(`{"model":"model-text","messages":[{"role":"user","content":"hi"}]}`)}
	var received proxy.Response
	if err := client.Forward(context.Background(), "token-1", request, func(response proxy.Response) error {
		received = response
		body, _ := io.ReadAll(response.Body)
		if !strings.Contains(string(body), "OK") {
			t.Fatalf("body = %s", body)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if received.StatusCode != 200 || received.ContentType != "application/json" {
		t.Fatalf("response = %+v", received)
	}
}

func TestClientAddsIncludeUsageOnlyForStreams(t *testing.T) {
	var bodies, accepts []string
	client := testGatewayClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))
		accepts = append(accepts, r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: [DONE]\n\n")
	})
	stream := proxy.Request{Protocol: "chat", Stream: true, Body: []byte(`{"model":"m","stream":true,"stream_options":{"other":true}}`)}
	if err := client.Forward(context.Background(), "token", stream, func(proxy.Response) error { return nil }); err != nil {
		t.Fatal(err)
	}
	plain := proxy.Request{Protocol: "chat", Body: []byte(`{"model":"m","messages":[]}`)}
	if err := client.Forward(context.Background(), "token", plain, func(proxy.Response) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 2 || !strings.Contains(bodies[0], `"include_usage":true`) || !strings.Contains(bodies[0], `"other":true`) {
		t.Fatalf("stream body = %v", bodies)
	}
	if strings.Contains(bodies[1], "stream_options") {
		t.Fatalf("non-stream body was rewritten: %s", bodies[1])
	}
	if accepts[0] != "text/event-stream" || accepts[1] != "" {
		t.Fatalf("accept headers = %v", accepts)
	}
}

func TestClientMapsTokenRejectionAndPassesOtherStatuses(t *testing.T) {
	rejecting := testGatewayClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	err := rejecting.Forward(context.Background(), "token", proxy.Request{Protocol: "chat", Body: []byte(`{}`)}, func(proxy.Response) error { return nil })
	if proxyCode(err) != CodeReauthRequired {
		t.Fatalf("401 code = %v", err)
	}
	limited := testGatewayClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(http.StatusTooManyRequests)
	})
	var status int
	if err := limited.Forward(context.Background(), "token", proxy.Request{Protocol: "chat", Body: []byte(`{}`)}, func(response proxy.Response) error {
		status = response.StatusCode
		return nil
	}); err != nil || status != http.StatusTooManyRequests {
		t.Fatalf("429 = (%d, %v)", status, err)
	}
}

func TestClientRejectsUnsupportedInput(t *testing.T) {
	client := New(nil)
	if code := proxyCode(client.Forward(context.Background(), "token", proxy.Request{Protocol: "responses", Body: []byte(`{}`)}, func(proxy.Response) error { return nil })); code != "unsupported_protocol" {
		t.Fatalf("protocol code = %q", code)
	}
	for _, token := range []string{"", "bad token", strings.Repeat("t", 9000)} {
		if code := proxyCode(client.Forward(context.Background(), token, proxy.Request{Protocol: "chat", Body: []byte(`{}`)}, func(proxy.Response) error { return nil })); code != "invalid_credential" {
			t.Fatalf("token %q code = %q", token[:min(len(token), 8)], code)
		}
	}
	oversized := proxy.Request{Protocol: "chat", Body: make([]byte, proxy.MaxRequestBytes+1)}
	if code := proxyCode(client.Forward(context.Background(), "token", oversized, func(proxy.Response) error { return nil })); code != "invalid_request" {
		t.Fatalf("oversized code = %q", code)
	}
}
