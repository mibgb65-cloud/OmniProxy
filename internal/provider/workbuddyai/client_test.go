package workbuddyai

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

// newTestClient 起一个仅回环的假上游，并返回指向它的客户端。
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClientAt(server.URL, server.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

// testCredentials 是测试用的假登录态，不含任何真实令牌。
func testCredentials() Credentials {
	return Credentials{
		AccessToken:  "access-token-1",
		RefreshToken: "refresh-token-1",
		UID:          "uid-1",
		Domain:       "www.workbuddy.ai",
		AccountName:  "Tester",
		Plan:         "Pro",
	}
}

// proxyCode 取边界错误码；其它错误返回空串。
func proxyCode(err error) string {
	var failure *proxy.Error
	if errors.As(err, &failure) {
		return failure.Code
	}
	return ""
}

// gunzipBytes 解压测试里的 gzip 请求体。
func gunzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer reader.Close()
	plain, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("gzip read: %v", err)
	}
	return plain
}

func TestForwardSendsDesktopClientShape(t *testing.T) {
	body := []byte(`{"model":"glm-5.3","stream":true,"messages":[{"role":"system","content":"s"},{"role":"user","content":"hello world"}]}`)
	var header http.Header
	var compressed []byte
	var requested struct{ method, path string }
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requested.method, requested.path = r.Method, r.URL.Path
		header = r.Header.Clone()
		compressed, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: [DONE]\n\n")
	})
	request := proxy.Request{Protocol: ChatProtocol, Stream: true, Model: "glm-5.3", Body: body}
	var status int
	var contentType string
	if err := client.Forward(context.Background(), testCredentials(), request, func(response proxy.Response) error {
		status, contentType = response.StatusCode, response.ContentType
		_, _ = io.Copy(io.Discard, response.Body)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if requested.method != http.MethodPost || requested.path != "/v2/chat/completions" {
		t.Fatalf("request = %+v", requested)
	}
	wantHeaders := map[string]string{
		"Authorization":               "Bearer access-token-1",
		"X-User-Id":                   "uid-1",
		"X-Domain":                    "www.workbuddy.ai",
		"X-Product":                   "SaaS",
		"X-IDE-Type":                  "WorkBuddy",
		"X-IDE-Name":                  "WorkBuddy",
		"X-IDE-Version":               AppVersion,
		"X-Requested-With":            "XMLHttpRequest",
		"X-Agent-Intent":              "craft",
		"X-Agent-Purpose":             "conversation",
		"X-Agent-Type":                "main",
		"X-Agent-Runtime":             "local-agent",
		"Accept":                      "application/json",
		"Content-Type":                "application/json",
		"Content-Encoding":            "gzip",
		"X-Stainless-Lang":            "js",
		"X-Stainless-Package-Version": "6.25.0",
		"X-Stainless-Runtime":         "node",
		"X-Stainless-Runtime-Version": "v22.21.1",
		"X-Stainless-Retry-Count":     "0",
		"X-Stainless-Timeout":         "1200",
		"X-Stainless-OS":              stainlessOS(runtime.GOOS),
		"X-Stainless-Arch":            stainlessArch(runtime.GOARCH),
		"User-Agent":                  "WorkBuddy/5.9.3 workbuddy-ai/5.9.3 CLI/2.165.0",
	}
	for name, want := range wantHeaders {
		if got := header.Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	if got := header.Get("X-Enterprise-Id"); got != "" {
		t.Errorf("personal account sent X-Enterprise-Id %q", got)
	}
	// 正文必须是调用方给的那一份：不补 stream、不补 stream_options、可解压还原。
	if plain := gunzipBytes(t, compressed); !bytes.Equal(plain, body) {
		t.Fatalf("body rewritten: %s", plain)
	}
	// 会话 ID 由第一条 user 消息内容的 sha256 前 16 字节决定。
	digest := sha256.Sum256([]byte("hello world"))
	if got := header.Get("X-Conversation-ID"); got != hex.EncodeToString(digest[:16]) {
		t.Fatalf("X-Conversation-ID = %q", got)
	}
	requestID := header.Get("X-Conversation-Request-ID")
	if len(requestID) != 32 {
		t.Fatalf("request id = %q", requestID)
	}
	if header.Get("X-Conversation-Message-ID") != requestID || header.Get("X-Request-ID") != requestID {
		t.Fatalf("ids = %v", header)
	}
	if status != http.StatusOK || contentType != "text/event-stream" {
		t.Fatalf("response = (%d, %q)", status, contentType)
	}
}

func TestForwardOmitsConversationIDWithoutUserMessage(t *testing.T) {
	var header http.Header
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		header = r.Header.Clone()
		io.WriteString(w, "data: [DONE]\n\n")
	})
	body := []byte(`{"model":"m","stream":true,"messages":[{"role":"system","content":"only system"}]}`)
	if err := client.Forward(context.Background(), testCredentials(), proxy.Request{Protocol: ChatProtocol, Stream: true, Body: body}, func(proxy.Response) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if got := header.Get("X-Conversation-ID"); got != "" {
		t.Fatalf("X-Conversation-ID = %q, want empty", got)
	}
	if header.Get("X-Conversation-Request-ID") == "" {
		t.Fatal("request id missing")
	}
}

func TestForwardSendsEnterpriseHeaders(t *testing.T) {
	var header http.Header
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		header = r.Header.Clone()
		io.WriteString(w, "data: [DONE]\n\n")
	})
	cred := testCredentials()
	cred.EnterpriseID = "ent-1"
	body := []byte(`{"model":"m","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	if err := client.Forward(context.Background(), cred, proxy.Request{Protocol: ChatProtocol, Stream: true, Body: body}, func(proxy.Response) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if header.Get("X-Enterprise-Id") != "ent-1" || header.Get("X-Tenant-Id") != "ent-1" {
		t.Fatalf("enterprise headers = %v", header)
	}
}

func TestForwardRejectsNonStreamingChat(t *testing.T) {
	called := false
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		called = true
	})
	body := []byte(`{"model":"m","messages":[{"role":"user","content":"hi"}]}`)
	err := client.Forward(context.Background(), testCredentials(), proxy.Request{Protocol: ChatProtocol, Stream: false, Body: body}, func(proxy.Response) error { return nil })
	if proxyCode(err) != "unsupported_protocol" || called {
		t.Fatalf("stream=false => (%q, called=%v)", proxyCode(err), called)
	}
}

func TestForwardRejectsUnsupportedProtocolAndInput(t *testing.T) {
	client := New(nil)
	valid := proxy.Request{Protocol: ChatProtocol, Stream: true, Body: []byte(`{"model":"m","stream":true}`)}
	for _, protocol := range []string{"responses", "messages", ""} {
		request := valid
		request.Protocol = protocol
		if code := proxyCode(client.Forward(context.Background(), testCredentials(), request, func(proxy.Response) error { return nil })); code != "unsupported_protocol" {
			t.Fatalf("protocol %q => %q", protocol, code)
		}
	}
	badCredentials := []Credentials{{}, {AccessToken: "bad token", UID: "uid-1"}, {AccessToken: strings.Repeat("a", 9000), UID: "uid-1"}}
	for _, cred := range badCredentials {
		if code := proxyCode(client.Forward(context.Background(), cred, valid, func(proxy.Response) error { return nil })); code != "invalid_credential" {
			t.Fatalf("credential %q => %q", cred.AccessToken, code)
		}
	}
	if code := proxyCode(client.Forward(context.Background(), testCredentials(), valid, nil)); code != "invalid_credential" {
		t.Fatalf("nil consumer => %q", code)
	}
	empty := valid
	empty.Body = nil
	if code := proxyCode(client.Forward(context.Background(), testCredentials(), empty, func(proxy.Response) error { return nil })); code != "invalid_request" {
		t.Fatalf("empty body => %q", code)
	}
	oversized := valid
	oversized.Body = make([]byte, proxy.MaxRequestBytes+1)
	if code := proxyCode(client.Forward(context.Background(), testCredentials(), oversized, func(proxy.Response) error { return nil })); code != "invalid_request" {
		t.Fatalf("oversized body => %q", code)
	}
}

func TestForwardMapsKnownUpstreamFailures(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{name: "unauthorized", status: http.StatusUnauthorized, want: CodeUnauthorized},
		{name: "rate limited", status: http.StatusTooManyRequests, want: CodeRateLimited},
		{name: "user agent", status: http.StatusBadRequest, body: `{"code":10085,"msg":"bad ua"}`, want: CodeUserAgentRefused},
		{name: "channel", status: http.StatusBadRequest, body: `{"code":11128,"msg":"unapproved channel"}`, want: CodeChannelRejected},
		{name: "signature", status: http.StatusBadRequest, body: `{"error":{"code":"400001","internal_code":"11159"}}`, want: CodeSignatureInvalid},
		{name: "entitlement", status: http.StatusBadRequest, body: `{"code":11102,"error":"11158:no entitlement"}`, want: CodeModelNotEntitled},
		{name: "signature text", status: http.StatusBadRequest, body: `{"error":{"message":"thinking_signature_invalid: replay"}}`, want: ""},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				io.WriteString(w, test.body)
			})
			body := []byte(`{"model":"m","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
			consumed := false
			err := client.Forward(context.Background(), testCredentials(), proxy.Request{Protocol: ChatProtocol, Stream: true, Body: body}, func(response proxy.Response) error {
				consumed = true
				_, _ = io.Copy(io.Discard, response.Body)
				return nil
			})
			if test.want == "" {
				// 只有源码里的固定业务码会被识别，措辞不参与分类。
				if err != nil || !consumed {
					t.Fatalf("unexpected classification: (%q, consumed=%v)", ErrorCode(err), consumed)
				}
				return
			}
			if ErrorCode(err) != test.want || consumed {
				t.Fatalf("code = %q (consumed=%v), want %q", ErrorCode(err), consumed, test.want)
			}
		})
	}
}

func TestForwardPassesUnknownFailureToConsumer(t *testing.T) {
	upstreamBody := `{"error":{"message":"upstream exploded"},"requestId":"r1"}`
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Retry-After", "12")
		w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(w, upstreamBody)
	})
	body := []byte(`{"model":"m","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	var received proxy.Response
	if err := client.Forward(context.Background(), testCredentials(), proxy.Request{Protocol: ChatProtocol, Stream: true, Body: body}, func(response proxy.Response) error {
		received = response
		data, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		if string(data) != upstreamBody {
			t.Fatalf("replayed body = %q", data)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if received.StatusCode != http.StatusServiceUnavailable || received.ContentType != "application/json; charset=utf-8" || received.RetryAfter != "12" {
		t.Fatalf("response = %+v", received)
	}
}

func TestForwardPropagatesConsumerError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "data: [DONE]\n\n")
	})
	sentinel := errors.New("consumer stopped")
	body := []byte(`{"model":"m","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	err := client.Forward(context.Background(), testCredentials(), proxy.Request{Protocol: ChatProtocol, Stream: true, Body: body}, func(proxy.Response) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("consumer error = %v", err)
	}
}

func TestNewClientAtRejectsUnsafeBase(t *testing.T) {
	for _, base := range []string{"http://example.com", "ftp://www.workbuddy.ai", "https://www.workbuddy.ai?x=1", "https://www.workbuddy.ai/#f", "://bad", ""} {
		if _, err := NewClientAt(base, nil); proxyCode(err) != "invalid_configuration" {
			t.Fatalf("base %q => %v", base, err)
		}
	}
	if _, err := NewClientAt("http://127.0.0.1:8080", nil); err != nil {
		t.Fatalf("loopback rejected: %v", err)
	}
	if _, err := NewClientAt("https://www.workbuddy.ai/", nil); err != nil {
		t.Fatalf("https rejected: %v", err)
	}
}

func TestStainlessRuntimeMapping(t *testing.T) {
	operatingSystems := map[string]string{"windows": "Windows", "darwin": "MacOS", "linux": "Linux", "freebsd": "FreeBSD", "openbsd": "OpenBSD", "android": "Android", "js": "Unknown", "": "Unknown"}
	for goos, want := range operatingSystems {
		if got := stainlessOS(goos); got != want {
			t.Errorf("stainlessOS(%q) = %q, want %q", goos, got, want)
		}
	}
	architectures := map[string]string{"amd64": "x64", "arm64": "arm64", "arm": "arm", "386": "x32", "mips": "Unknown", "": "Unknown"}
	for goarch, want := range architectures {
		if got := stainlessArch(goarch); got != want {
			t.Errorf("stainlessArch(%q) = %q, want %q", goarch, got, want)
		}
	}
}

func TestConversationIDDerivation(t *testing.T) {
	digest := sha256.Sum256([]byte("first message"))
	want := hex.EncodeToString(digest[:16])
	if got := conversationID([]byte(`{"messages":[{"role":"system","content":"s"},{"role":"user","content":"first message"},{"role":"user","content":"second"}]}`)); got != want {
		t.Fatalf("string content id = %q", got)
	}
	// 结构化内容用紧凑 JSON 参与摘要，与字符串形式不同但稳定。
	structured := conversationID([]byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"first message"}]}]}`))
	if len(structured) != 32 || structured == want {
		t.Fatalf("structured id = %q", structured)
	}
	if conversationID([]byte(`{"messages":[{"role":"assistant","content":"x"}]}`)) != "" {
		t.Fatal("assistant-only conversation got an id")
	}
	if conversationID([]byte(`not json`)) != "" {
		t.Fatal("invalid body got an id")
	}
}
