package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClaudeCodeBetaMessagesAndQueryBoundaries(t *testing.T) {
	called := 0
	server := New(Options{Models: []string{"deepseek-flash"}, Authorize: func(_ context.Context, key string) (string, bool) { return "", key == "test-key" },
		Upstream: upstreamFunc(func(_ context.Context, request Request, consume func(Response) error) error {
			called++
			if request.Protocol != "messages" || request.Model != "deepseek-flash" {
				t.Fatal("beta request changed protocol or model")
			}
			return consume(Response{StatusCode: 200, ContentType: "application/json", Body: strings.NewReader(`{"type":"message","content":[{"type":"text","text":"OK"}]}`)})
		})})
	for _, test := range []struct {
		path, key string
		status    int
	}{
		{"/v1/messages?beta=true", "test-key", 200},
		{"/v1/messages", "test-key", 200},
		{"/v1/messages?beta=true", "wrong-key", 401},
		{"/v1/messages?beta=true", "", 401},
		{"/v1/messages?beta=false", "test-key", 400},
		{"/v1/messages?beta=true&beta=true", "test-key", 400},
		{"/v1/messages?beta=true&redirect=x", "test-key", 400},
		{"/v1/messages?beta=%00", "test-key", 400},
		{"/v1/messages?beta=%", "test-key", 400},
		{"/v1/messages?beta=" + strings.Repeat("a", 512), "test-key", 400},
		{"/v1/responses?beta=true", "test-key", 400},
		{"/v1/chat/completions?beta=true", "test-key", 400},
	} {
		t.Run(test.path+test.key, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:0"+test.path, strings.NewReader(`{"model":"deepseek-flash","max_tokens":32000,"messages":[{"role":"user","content":"hello"}]}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer "+test.key)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status=%d, expected=%d", response.Code, test.status)
			}
		})
	}
	if called != 2 {
		t.Fatal("invalid queries or credentials reached upstream")
	}
}

func TestClaudeContextModelSyntaxIsMessagesOnly(t *testing.T) {
	for _, model := range []string{"deepseek-flash[1m]", "deepseek-flash[2m]", "deepseek-flash[1m][1m]", "[1m]", strings.Repeat("x", 125) + "[1m]"} {
		body := []byte(`{"model":"` + model + `","max_tokens":1,"messages":[{"role":"user","content":"hello"}],"input":"hello"}`)
		_, err := ParseProtocolRequest(body, "/v1/messages")
		if (err == nil) != (model == "deepseek-flash[1m]") {
			t.Fatal("incorrect Messages model syntax", model, err)
		}
		if _, err := ParseProtocolRequest(body, "/v1/responses"); err == nil {
			t.Fatal("Claude context suffix accepted for Responses")
		}
		if _, err := ParseRequest(body); err == nil {
			t.Fatal("Claude context suffix accepted for Chat Completions")
		}
	}
}

func TestMessagesBodyLimitRemainsBounded(t *testing.T) {
	body := `{"model":"deepseek-flash","max_tokens":1,"messages":[{"role":"user","content":"hello"}]}`
	within := []byte(body + strings.Repeat(" ", MaxMessagesRequestBytes-len(body)))
	if _, err := ParseProtocolRequest(within, "/v1/messages"); err != nil {
		t.Fatal("valid Messages request at the limit rejected", err)
	}
	if _, err := ParseProtocolRequest(within, "/v1/responses"); err == nil {
		t.Fatal("Messages capacity leaked into Responses")
	}
	server := New(Options{})
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:0/v1/messages?beta=true", strings.NewReader(string(within)+" "))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatal("oversized Messages body was accepted", response.Code)
	}
	if _, err := ParseProtocolRequest(append(within, ' '), "/v1/messages"); err == nil {
		t.Fatal("oversized Messages JSON was accepted")
	}
}
