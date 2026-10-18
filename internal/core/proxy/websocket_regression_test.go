package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func regressionSocket(t *testing.T, options Options) (*websocket.Conn, context.Context) {
	t.Helper()
	options.Port = 0
	options.Models = []string{"gpt-test"}
	server := New(options)
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := server.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	socket, _, err := websocket.Dial(ctx, strings.Replace(server.Status().Address, "http:", "ws:", 1)+"/v1/responses", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { socket.CloseNow() })
	return socket, ctx
}

func sendSocket(t *testing.T, ctx context.Context, socket *websocket.Conn, body string) map[string]json.RawMessage {
	t.Helper()
	if err := socket.Write(ctx, websocket.MessageText, []byte(body)); err != nil {
		t.Fatal(err)
	}
	_, data, err := socket.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var event map[string]json.RawMessage
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatal(err)
	}
	return event
}

func TestSocketUpstreamErrorsPreserveStatusAndCode(t *testing.T) {
	for _, tc := range []struct {
		status, want int
		code         string
	}{
		{400, 400, "upstream_rejected_request"}, {401, 401, "upstream_authentication_failed"},
		{402, 402, "upstream_insufficient_balance"}, {403, 403, "upstream_authentication_failed"},
		{404, 404, "upstream_rejected_request"}, {429, 429, "upstream_rate_limited"},
		{503, 502, "upstream_unavailable"}, {302, 502, "upstream_unavailable"},
	} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			finished := make(chan RequestEvent, 1)
			socket, ctx := regressionSocket(t, Options{Observer: firstTokenObserver{finished}, Upstream: upstreamFunc(func(_ context.Context, _ Request, consume func(Response) error) error {
				return consume(Response{StatusCode: tc.status, RetryAfter: "17", Body: strings.NewReader(`{"error":{"message":"private-upstream-body"}}`)})
			})})
			event := sendSocket(t, ctx, socket, `{"type":"response.create","model":"gpt-test","input":[]}`)
			var status int
			var kind string
			var failure struct{ Code, Message string }
			var headers map[string]string
			json.Unmarshal(event["status"], &status)
			json.Unmarshal(event["type"], &kind)
			json.Unmarshal(event["error"], &failure)
			json.Unmarshal(event["headers"], &headers)
			if status != tc.want || kind != "error" || failure.Code != tc.code || failure.Message != tc.code || headers["retry-after"] != "17" {
				t.Fatalf("incorrect safe socket error: status=%d code=%s headers=%v", status, failure.Code, headers)
			}
			select {
			case record := <-finished:
				if record.StatusCode != tc.want || record.ErrorCode != tc.code {
					t.Fatal("request record lost upstream failure", record)
				}
			case <-ctx.Done():
				t.Fatal("request not recorded")
			}
		})
	}
}

func TestSocketWarmupDoesNotCallUpstreamAndPreservesHistory(t *testing.T) {
	var calls atomic.Int32
	finished := make(chan RequestEvent, 2)
	socket, ctx := regressionSocket(t, Options{Observer: firstTokenObserver{finished}, Upstream: upstreamFunc(func(_ context.Context, r Request, consume func(Response) error) error {
		calls.Add(1)
		if strings.Contains(string(r.Body), `"generate"`) || strings.Contains(string(r.Body), "previous_response_id") || !strings.Contains(string(r.Body), "warmup-context") || !strings.Contains(string(r.Body), "actual-question") {
			t.Error("warmup metadata leaked or history was lost")
		}
		return consume(Response{StatusCode: 200, ContentType: "text/event-stream", Body: strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"actual\",\"output\":[]}}\n\n")})
	})})
	event := sendSocket(t, ctx, socket, `{"type":"response.create","model":"gpt-test","generate":false,"input":[{"role":"system","content":"warmup-context"}]}`)
	if string(event["type"]) != `"response.created"` {
		t.Fatal("warmup missing created event")
	}
	_, body, err := socket.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var completed struct {
		Type     string
		Response struct{ ID string }
	}
	if json.Unmarshal(body, &completed) != nil || completed.Type != "response.completed" || completed.Response.ID == "" {
		t.Fatal("invalid warmup completion")
	}
	if calls.Load() != 0 || len(finished) != 0 {
		t.Fatal("warmup consumed model or accounting")
	}
	event = sendSocket(t, ctx, socket, `{"type":"response.create","model":"gpt-test","generate":true,"previous_response_id":"`+completed.Response.ID+`","input":[{"role":"user","content":"actual-question"}]}`)
	if string(event["type"]) != `"response.completed"` || calls.Load() != 1 {
		t.Fatal("real request not completed exactly once")
	}
	select {
	case <-finished:
	case <-ctx.Done():
		t.Fatal("real request not recorded")
	}
}

func TestSocketGenerateRequiresBoolean(t *testing.T) {
	for _, value := range []string{`null`, `"false"`, `0`, `{}`} {
		t.Run(value, func(t *testing.T) {
			socket, ctx := regressionSocket(t, Options{Upstream: upstreamFunc(func(context.Context, Request, func(Response) error) error {
				t.Error("invalid generate reached upstream")
				return nil
			})})
			event := sendSocket(t, ctx, socket, `{"type":"response.create","model":"gpt-test","generate":`+value+`,"input":[]}`)
			if !strings.Contains(string(event["error"]), "invalid_request") {
				t.Fatal("invalid generate accepted")
			}
		})
	}
}
