package proxy

import (
	"context"
	"github.com/coder/websocket"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestScopedAgentKeyAndWebsocketConversation(t *testing.T) {
	key := "opk_" + strings.Repeat("a", 64)
	received := []string{}
	upstream := upstreamFunc(func(_ context.Context, request Request, consume func(Response) error) error {
		received = append(received, string(request.Body))
		if request.Protocol != "responses" {
			t.Error("wrong protocol")
		}
		return consume(Response{StatusCode: 200, ContentType: "text/event-stream", Body: strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_test\",\"output\":[{\"type\":\"message\",\"role\":\"assistant\",\"content\":\"OK\"}]}}\n\n")})
	})
	server := New(Options{Port: 0, Upstream: upstream, Models: []string{"deepseek-flash"}, Authorize: func(_ context.Context, k string) (string, bool) { return "", k == key }})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop(context.Background())
	root := server.Status().Address
	resp, err := http.Get(root + "/v1/models")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatal("missing key accepted")
	}
	req, _ := http.NewRequest("GET", root+"/agent/codex/"+key+"/v1/models", nil)
	req.Header.Set("Authorization", "Bearer original-official-auth")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal("scoped API key rejected")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	socket, _, err := websocket.Dial(ctx, strings.Replace(root, "http:", "ws:", 1)+"/agent/codex/"+key+"/v1/responses", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer socket.CloseNow()
	for _, body := range []string{`{"type":"response.create","model":"deepseek-flash","input":[{"role":"user","content":"first"}]}`, `{"type":"response.create","model":"deepseek-flash","previous_response_id":"resp_test","input":[{"role":"user","content":"second"}]}`} {
		if err := socket.Write(ctx, websocket.MessageText, []byte(body)); err != nil {
			t.Fatal(err)
		}
		_, result, err := socket.Read(ctx)
		if err != nil || !strings.Contains(string(result), "response.completed") {
			t.Fatal("socket failed", err)
		}
	}
	if len(received) != 2 || !strings.Contains(received[1], "first") || !strings.Contains(received[1], "second") || strings.Contains(received[1], "previous_response_id") {
		t.Fatal("socket history lost")
	}
}
func TestNativeProtocolJSONResponses(t *testing.T) {
	for _, protocol := range []string{"responses", "messages"} {
		body := `{"output":[],"object":"response"}`
		if protocol == "messages" {
			body = `{"type":"message","content":[]}`
		}
		server := testServer(t, upstreamFunc(func(_ context.Context, r Request, c func(Response) error) error {
			return c(Response{StatusCode: 200, ContentType: "application/json", Body: strings.NewReader(body)})
		}), 0)
		input := `{"model":"deepseek-flash","input":"hello"}`
		if protocol == "messages" {
			input = `{"model":"deepseek-flash","max_tokens":10,"messages":[{"role":"user","content":"hello"}]}`
		}
		resp, err := http.Post(server.Status().Address+"/v1/"+protocol, "application/json", strings.NewReader(input))
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 || string(b) != body {
			t.Fatalf("native JSON rejected: %s", protocol)
		}
	}
}
