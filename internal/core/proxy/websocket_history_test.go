package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestSocketLongHistoryReconnectAfterPartialStream(t *testing.T) {
	initial, partial := responseHistory(510), responseHistory(6)
	replay := append(append([]json.RawMessage{}, initial...), partial...)
	output := []json.RawMessage{json.RawMessage(`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"recovered"}]}`)}
	next := append(append(append([]json.RawMessage{}, replay...), output...), json.RawMessage(`{"role":"user","content":"continue"}`))
	expected := [][]json.RawMessage{initial, replay, next}
	var calls atomic.Int32
	server := New(Options{Models: []string{"gpt-test"}, Upstream: upstreamFunc(func(_ context.Context, request Request, consume func(Response) error) error {
		index := int(calls.Add(1)) - 1
		if index >= len(expected) {
			t.Error("interrupted generation was replayed implicitly")
			return Failure("unexpected_call", 502)
		}
		var fields struct{ Input json.RawMessage }
		want, err := json.Marshal(expected[index])
		if err != nil || json.Unmarshal(request.Body, &fields) != nil || !bytes.Equal(fields.Input, want) {
			t.Error("reconnect lost, duplicated or truncated history")
		}
		if index == 0 {
			var frames strings.Builder
			for _, item := range partial {
				event, marshalErr := json.Marshal(map[string]any{"type": "response.output_item.done", "item": item})
				if marshalErr != nil {
					t.Error(marshalErr)
					return Failure("test_encoding_failed", 502)
				}
				frames.WriteString("data: " + string(event) + "\n\n")
			}
			return consume(Response{StatusCode: 200, ContentType: "text/event-stream", Body: io.MultiReader(strings.NewReader(frames.String()), streamFailureReader{})})
		}
		event, err := json.Marshal(map[string]any{"type": "response.completed", "response": map[string]any{"id": "resp_recovered", "output": output}})
		if err != nil {
			t.Error(err)
			return Failure("test_encoding_failed", 502)
		}
		return consume(Response{StatusCode: 200, ContentType: "text/event-stream", Body: strings.NewReader("data: " + string(event) + "\n\n")})
	})})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := server.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	connect := func() *websocket.Conn {
		socket, _, err := websocket.Dial(ctx, strings.Replace(server.Status().Address, "http:", "ws:", 1)+"/v1/responses", nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { socket.CloseNow() })
		return socket
	}
	socket := connect()
	if err := socket.Write(ctx, websocket.MessageText, historyRequest(t, initial, true)); err != nil {
		t.Fatal(err)
	}
	for range partial {
		_, raw, err := socket.Read(ctx)
		if err != nil || !strings.Contains(string(raw), "response.output_item.done") {
			t.Fatal("partial stream was not delivered", err)
		}
	}
	_, raw, err := socket.Read(ctx)
	if err != nil || !strings.Contains(string(raw), "upstream_stream_failed") || strings.Contains(string(raw), "private-reader-detail") {
		t.Fatal("interrupted stream lost its safe failure code", err)
	}
	socket.CloseNow()
	socket = connect()
	event := sendSocket(t, ctx, socket, string(historyRequest(t, replay, true)))
	if string(event["type"]) != `"response.completed"` {
		t.Fatal("516-item full replay was rejected")
	}
	event = sendSocket(t, ctx, socket, `{"type":"response.create","model":"gpt-test","previous_response_id":"resp_recovered","input":[{"role":"user","content":"continue"}]}`)
	if string(event["type"]) != `"response.completed"` || calls.Load() != 3 {
		t.Fatal("incremental continuation after reconnect failed")
	}
}

func TestSocketValidationPreservesSafeCodeAndLogsRejection(t *testing.T) {
	for _, tc := range []struct{ model, input, code string }{
		{`1`, `[]`, "invalid_model"},
		{`"gpt-test","model":"gpt-test"`, `[]`, "invalid_request"},
		{`"gpt-test"`, strings.Repeat("[", 33) + `0` + strings.Repeat("]", 33), "invalid_request"},
		{`"gpt-test"`, `[{"role":"user","role":"assistant","content":"private-input-detail"}]`, "invalid_request"},
	} {
		t.Run(tc.code+tc.model, func(t *testing.T) {
			var logs bytes.Buffer
			socket, ctx := regressionSocket(t, Options{Logger: slog.New(slog.NewJSONHandler(&logs, nil)), Upstream: upstreamFunc(func(context.Context, Request, func(Response) error) error {
				t.Error("invalid request reached upstream")
				return nil
			})})
			event := sendSocket(t, ctx, socket, `{"type":"response.create","model":`+tc.model+`,"input":`+tc.input+`}`)
			var failure struct{ Code string }
			if json.Unmarshal(event["error"], &failure) != nil || failure.Code != tc.code || string(event["status"]) != "400" {
				t.Fatal("socket changed the parser's safe failure")
			}
			if !strings.Contains(logs.String(), `"msg":"gateway request rejected"`) || !strings.Contains(logs.String(), `"code":"`+tc.code+`"`) || strings.Contains(logs.String(), "private-input-detail") {
				t.Fatal("rejection diagnostic missing or contains request content")
			}
		})
	}
}
