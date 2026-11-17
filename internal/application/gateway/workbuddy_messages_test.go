package gateway

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
	"omniproxy/internal/provider/workbuddyai"
)

// 国内版的 Claude 模型由网关直接送到原生 /v1/messages：正文与流原样转发，
// 不经过 Chat 的聚合与改写。
func TestForwardWorkBuddyMessagesRelaysNativeEndpoint(t *testing.T) {
	var (
		path, method string
		body         []byte
		header       http.Header
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, method, header = r.URL.Path, r.Method, r.Header.Clone()
		body, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "event: message_start\ndata: {\"type\":\"message_start\"}\n\n")
	}))
	t.Cleanup(server.Close)
	client, err := workbuddyai.NewClientAtFor(workbuddyai.China, server.URL, server.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{}
	request := proxy.Request{
		Protocol: workbuddyai.MessagesProtocol,
		Model:    "claude-opus-5.5",
		Stream:   true,
		Body:     []byte(`{"model":"claude-opus-5.5","messages":[{"role":"user","content":"hi"}],"stream":true}`),
	}
	credential := workbuddyai.Credentials{AccessToken: "fix-access", UID: "uid-1", Domain: "copilot.tencent.com"}
	var status int
	var contentType string
	var relayed []byte
	if err := service.forwardWorkBuddyMessages(context.Background(), client, credential, request, func(response proxy.Response) error {
		status, contentType = response.StatusCode, response.ContentType
		relayed, _ = io.ReadAll(response.Body)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPost || path != "/v1/messages" {
		t.Fatalf("request = %s %s", method, path)
	}
	if header.Get("X-Domain") != "copilot.tencent.com" {
		t.Fatalf("domain header = %q", header.Get("X-Domain"))
	}
	if !strings.Contains(string(body), `"model":"claude-opus-5.5"`) || strings.Contains(string(body), "stream_options") {
		t.Fatalf("body rewritten: %s", body)
	}
	if status != http.StatusOK || contentType != "text/event-stream" {
		t.Fatalf("status/content = %d %q", status, contentType)
	}
	if !strings.Contains(string(relayed), "message_start") {
		t.Fatalf("relayed = %q", relayed)
	}
}

// 国际版没有 Messages 端点，调用方用 Messages 协议访问 Claude 模型时必须继续走 Chat
// 聚合路径（这里只断言分派条件本身，避免把协议判错变成静默 500）。
func TestWorkBuddyNativeMessagesDispatch(t *testing.T) {
	cases := []struct {
		kind     string
		model    string
		protocol string
		native   bool
	}{
		{kind: "workbuddy", model: "claude-opus-5.5", protocol: "messages", native: true},
		{kind: "workbuddy", model: "glm-5.3", protocol: "messages", native: false},
		{kind: "workbuddy", model: "claude-opus-5.5", protocol: "chat", native: false},
		{kind: "workbuddy_ai", model: "claude-opus-5.5", protocol: "messages", native: false},
	}
	for _, test := range cases {
		build, ok := workbuddyai.BuildOf(test.kind)
		if !ok {
			t.Fatalf("%s build missing", test.kind)
		}
		native := test.protocol == workbuddyai.MessagesProtocol && build.MessagesNative(test.model)
		if native != test.native {
			t.Errorf("%s/%s/%s native = %v, want %v", test.kind, test.model, test.protocol, native, test.native)
		}
	}
}
