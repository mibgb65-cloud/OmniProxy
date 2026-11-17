package workbuddyai

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"omniproxy/internal/core/proxy"
)

// messagesRequest 是一条 Anthropic Messages 形态的最小请求。
func messagesRequest(body string) proxy.Request {
	return proxy.Request{
		Protocol: MessagesProtocol,
		Model:    "claude-opus-5.5",
		Stream:   true,
		Body:     []byte(body),
	}
}

// 国内版把 Claude 请求原样送到 /v1/messages：不 gzip、不改写系统提示词，
// 只补签名头、客户端头与会话 ID。
func TestForwardMessagesRelaysBodyAndHeaders(t *testing.T) {
	var (
		path, method string
		header       http.Header
		body         []byte
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, method, header = r.URL.Path, r.Method, r.Header.Clone()
		body, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "event: message_start\ndata: {\"type\":\"message_start\"}\n\n")
	}))
	t.Cleanup(server.Close)
	client, err := NewClientAtFor(China, server.URL, server.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	original := `{"model":"claude-opus-5.5","max_tokens":64,"system":[{"type":"text","text":"You are Claude Code."}],"messages":[{"role":"user","content":"hi"}],"stream":true}`
	var relayed []byte
	status := 0
	if err := client.ForwardMessages(context.Background(), testCredentials(), messagesRequest(original), func(response proxy.Response) error {
		status = response.StatusCode
		payload, _ := io.ReadAll(response.Body)
		relayed = payload
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPost || path != "/v1/messages" {
		t.Fatalf("request = %s %s", method, path)
	}
	// 正文按原样转发：不补 system、不改 max_tokens、不注入 stream_options。
	if !bytes.Equal(body, []byte(original)) {
		t.Fatalf("body rewritten: %s", body)
	}
	if header.Get("Content-Encoding") != "" {
		t.Fatalf("messages request was compressed: %q", header.Get("Content-Encoding"))
	}
	if header.Get("Authorization") != "Bearer access-token-1" || header.Get("X-User-Id") != "uid-1" || header.Get("X-Domain") != "www.workbuddy.ai" {
		t.Fatalf("signature headers = %v", header)
	}
	if header.Get("User-Agent") != "WorkBuddy/5.9.3 WorkBuddy/5.9.3 CLI/2.165.0" || header.Get("X-IDE-Version") != "5.9.3" {
		t.Fatalf("client headers = %v", header)
	}
	if header.Get("X-Agent-Type") != "main" || header.Get("X-Stainless-Package-Version") == "" {
		t.Fatalf("agent headers = %v", header)
	}
	if header.Get("X-Conversation-ID") == "" || header.Get("X-Conversation-Request-ID") == "" {
		t.Fatalf("conversation ids = %v", header)
	}
	if status != 200 || len(relayed) == 0 {
		t.Fatalf("status = %d body = %q", status, relayed)
	}
}

// 只有声明 Messages 的构建、且调用方用 Messages 协议时才允许走这里。
func TestForwardMessagesRejectsWrongBuildAndProtocol(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected upstream request")
	}))
	t.Cleanup(server.Close)
	international, err := NewClientAt(server.URL, server.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	if err := international.ForwardMessages(context.Background(), testCredentials(), messagesRequest(`{"messages":[]}`), func(proxy.Response) error { return nil }); proxyCode(err) != "unsupported_protocol" {
		t.Fatalf("international messages => %v", err)
	}
	china, err := NewClientAtFor(China, server.URL, server.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	request := messagesRequest(`{"messages":[]}`)
	request.Protocol = ChatProtocol
	if err := china.ForwardMessages(context.Background(), testCredentials(), request, func(proxy.Response) error { return nil }); proxyCode(err) != "unsupported_protocol" {
		t.Fatalf("chat protocol on messages path => %v", err)
	}
}

// Messages 的非 2xx 同样按稳定业务码映射，未识别时把正文原样交给网关。
func TestForwardMessagesMapsBusinessCodes(t *testing.T) {
	cases := map[string]int{
		"signature":   11159,
		"notentitled": 11158,
		"ua":          10085,
	}
	for name, code := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"code":`+itoa(code)+`,"msg":"refused"}`)
		}))
		client, err := NewClientAtFor(China, server.URL, server.Client().Transport)
		if err != nil {
			server.Close()
			t.Fatal(err)
		}
		err = client.ForwardMessages(context.Background(), testCredentials(), messagesRequest(`{"messages":[]}`), func(proxy.Response) error { return nil })
		server.Close()
		want, _ := businessFailure(code)
		if want == nil || codeOf(err) != codeOf(want) {
			t.Fatalf("%s => %v, want %v", name, err, want)
		}
	}
	// 未识别的 400 把正文交回网关，不吞掉、不改写成成功。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error":{"message":"bad shape"}}`)
	}))
	t.Cleanup(server.Close)
	client, err := NewClientAtFor(China, server.URL, server.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	var relayed []byte
	if err := client.ForwardMessages(context.Background(), testCredentials(), messagesRequest(`{"messages":[]}`), func(response proxy.Response) error {
		if response.StatusCode != http.StatusBadRequest {
			t.Fatalf("status = %d", response.StatusCode)
		}
		relayed, _ = io.ReadAll(response.Body)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(relayed, []byte("bad shape")) {
		t.Fatalf("replayed body = %q", relayed)
	}
}

// itoa 只用于拼测试里的业务码，避免为此引入 fmt。
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0, 8)
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}

// codeOf 取错误的稳定码文本，便于跨类型比较。
func codeOf(err error) string {
	if err == nil {
		return ""
	}
	return proxyCode(err) + ErrorCode(err)
}
