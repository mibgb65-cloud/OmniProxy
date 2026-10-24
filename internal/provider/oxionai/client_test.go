package oxionai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"omniproxy/internal/core/proxy"
)

func TestForwardSendsBearerChatCompletionsOnly(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method %q", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer cc_test_key" {
			t.Errorf("Authorization = %q", got)
		}
		if r.Header.Get("X-Api-Key") != "" {
			t.Error("must not send Anthropic style header")
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"model":"m","messages":[]}` {
			t.Errorf("body changed: %q", body)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {}\n\n")
	}))
	defer upstream.Close()

	client := New(nil)
	var got proxy.Response
	err := client.Forward(context.Background(), upstream.URL+"/v1", []byte("cc_test_key"), proxy.Request{Protocol: "chat", Stream: true, Body: []byte(`{"model":"m","messages":[]}`)}, func(response proxy.Response) error {
		got = response
		return nil
	})
	if err != nil || got.StatusCode != http.StatusOK || got.ContentType != "text/event-stream" {
		t.Fatalf("forward failed: %v %+v", err, got)
	}
}

func TestForwardRejectsNonChatProtocolAndUnsafeConfig(t *testing.T) {
	client := New(nil)
	for _, protocol := range []string{"responses", "messages"} {
		if err := client.Forward(context.Background(), DefaultBaseURL, []byte("cc_test_key"), proxy.Request{Protocol: protocol}, func(proxy.Response) error { return nil }); err == nil {
			t.Fatalf("protocol %q must not be forwarded natively", protocol)
		}
	}
	cases := []struct {
		name   string
		base   string
		key    []byte
		reason string
	}{
		{"空密钥", DefaultBaseURL, nil, "empty key"},
		{"非 ASCII 密钥", DefaultBaseURL, []byte("cc_键"), "non-ascii key"},
		{"明文远端 HTTP", "http://oxionai.com/v1", []byte("cc_test_key"), "plaintext remote"},
		{"基址带查询串", DefaultBaseURL + "?x=1", []byte("cc_test_key"), "query"},
		{"基址带用户信息", "https://user@oxionai.com/v1", []byte("cc_test_key"), "userinfo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := client.Forward(context.Background(), tc.base, tc.key, proxy.Request{Protocol: "chat", Body: []byte(`{}`)}, func(proxy.Response) error { return nil }); err == nil {
				t.Fatalf("%s must be rejected", tc.reason)
			}
		})
	}
}

func TestForwardAllowsLoopbackHTTP(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":{"code":"invalid_api_key"}}`)
	}))
	defer upstream.Close()

	client := New(nil)
	var got proxy.Response
	if err := client.Forward(context.Background(), upstream.URL, []byte("cc_test_key"), proxy.Request{Protocol: "chat", Body: []byte(`{}`)}, func(response proxy.Response) error {
		got = response
		return nil
	}); err != nil || got.StatusCode != http.StatusUnauthorized {
		t.Fatalf("loopback forward failed: %v %+v", err, got)
	}
	if got := completionsURLSuffix(t, upstream.URL); got != "/chat/completions" {
		t.Fatalf("suffix = %q", got)
	}
}

func completionsURLSuffix(t *testing.T, base string) string {
	t.Helper()
	address, err := completionsURL(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(address) < len(base) {
		t.Fatalf("address %q shorter than base", address)
	}
	return address[len(base):]
}
