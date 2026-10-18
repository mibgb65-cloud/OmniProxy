package proxy

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestSocketCompletesWithoutWaitingForUpstreamEOF(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	finished := make(chan RequestEvent, 1)
	socket, ctx := regressionSocket(t, Options{Observer: firstTokenObserver{finished}, Upstream: upstreamFunc(func(_ context.Context, _ Request, consume func(Response) error) error {
		go func() {
			if _, err := io.WriteString(writer, "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"one\",\"output\":[]}}\n\n"); err != nil {
				return
			}
		}()
		return consume(Response{StatusCode: 200, ContentType: "text/event-stream", Body: reader})
	})})
	event := sendSocket(t, ctx, socket, `{"type":"response.create","model":"gpt-test","input":[]}`)
	if string(event["type"]) != `"response.completed"` {
		t.Fatal("completion missing")
	}
	select {
	case <-finished:
	case <-ctx.Done():
		t.Fatal("request waited for EOF after completed event")
	}
}

func TestSocketInvalidResponseIsSafe(t *testing.T) {
	for _, tc := range []struct {
		name, kind, body string
		nilBody          bool
		code             string
	}{
		{"content-type", "application/json", `{"private":"private-body"}`, false, "invalid_upstream_response"},
		{"nil-body", "text/event-stream", "", true, "invalid_upstream_response"},
		{"nested-error", "text/event-stream", "data: {\"type\":\"response.failed\",\"response\":{\"error\":{\"message\":\"private-body\"}}}\n\n", false, "upstream_stream_failed"},
		{"incomplete", "text/event-stream", "data: {\"type\":\"response.created\"}\n\n", false, "upstream_stream_failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			socket, ctx := regressionSocket(t, Options{Upstream: upstreamFunc(func(_ context.Context, _ Request, consume func(Response) error) error {
				var body io.Reader = strings.NewReader(tc.body)
				if tc.nilBody {
					body = nil
				}
				return consume(Response{StatusCode: 200, ContentType: tc.kind, Body: body})
			})})
			event := sendSocket(t, ctx, socket, `{"type":"response.create","model":"gpt-test","input":[]}`)
			if tc.name == "incomplete" {
				_, raw, err := socket.Read(ctx)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(raw), tc.code) {
					t.Fatal("incomplete stream not failed")
				}
			} else if !strings.Contains(string(event["error"]), tc.code) || strings.Contains(string(event["error"]), "private-body") {
				t.Fatal("unsafe or missing response failure")
			}
		})
	}
}

func TestSocketWarmupAndPreviousIDStillValidateRequests(t *testing.T) {
	for _, body := range []string{
		`{"type":"response.create","model":"unknown-model","generate":false,"input":[]}`,
		`{"type":"response.create","model":"gpt-test","generate":false,"previous_response_id":"unknown","input":[]}`,
		`{"type":"response.create","model":"gpt-test","previous_response_id":1,"input":[]}`,
	} {
		t.Run(body, func(t *testing.T) {
			socket, ctx := regressionSocket(t, Options{Upstream: upstreamFunc(func(context.Context, Request, func(Response) error) error {
				t.Error("invalid request reached upstream")
				return nil
			})})
			event := sendSocket(t, ctx, socket, body)
			if string(event["type"]) != `"error"` {
				t.Fatal("invalid warmup or previous ID was accepted")
			}
		})
	}
}

func TestRetryAfterValidation(t *testing.T) {
	for _, value := range []string{"-1", "86401", "not-a-number", "17\r\nprivate-header: unsafe", strings.Repeat("9", 100)} {
		if failure := upstreamFailure(Response{StatusCode: 429, RetryAfter: value}); failure.RetryAfter != "" {
			t.Fatal("invalid retry-after retained")
		}
	}
}
