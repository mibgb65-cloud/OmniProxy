package proxy

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

type streamFailureReader struct{}

func (streamFailureReader) Read([]byte) (int, error) {
	return 0, errors.New("private-reader-detail")
}

func requireSafeStreamFailure(t *testing.T, err error, code string) {
	t.Helper()
	var failure *Error
	if !errors.As(err, &failure) || failure.Code != code || failure.StatusCode != 502 || strings.Contains(err.Error(), "private-") {
		t.Fatalf("stream error was not safely classified: %v", err)
	}
}

func TestSocketReadFailuresAreTyped(t *testing.T) {
	for _, tc := range []struct {
		name, code string
		body       io.Reader
	}{
		{"read", "upstream_stream_failed", streamFailureReader{}},
		{"event-limit", "response_too_large", strings.NewReader("data: " + strings.Repeat("x", MaxResponsesEventBytes))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := copySocketResponse(context.Background(), nil, Response{StatusCode: 200, ContentType: "text/event-stream", Body: tc.body}, nil)
			requireSafeStreamFailure(t, err, tc.code)
		})
	}
}

func TestSocketWriteFailureIsTyped(t *testing.T) {
	socket, ctx := regressionSocket(t, Options{Upstream: upstreamFunc(func(context.Context, Request, func(Response) error) error {
		t.Error("closed socket unexpectedly reached upstream")
		return nil
	})})
	socket.CloseNow()
	_, err := copySocketResponse(ctx, socket, Response{StatusCode: 200, ContentType: "text/event-stream", Body: strings.NewReader("data: {\"type\":\"response.created\"}\n\n")}, nil)
	requireSafeStreamFailure(t, err, "client_write_failed")
}

type failingResponseWriter struct {
	*httptest.ResponseRecorder
	failWrite bool
}

func (w failingResponseWriter) Write(body []byte) (int, error) {
	if w.failWrite {
		return 0, errors.New("private-writer-detail")
	}
	return w.ResponseRecorder.Write(body)
}

func (w failingResponseWriter) WriteString(body string) (int, error) {
	return w.Write([]byte(body))
}

func (w failingResponseWriter) FlushError() error {
	return errors.New("private-flush-detail")
}

func TestHTTPWriteFailuresAreTyped(t *testing.T) {
	for _, tc := range []struct {
		name      string
		stream    bool
		failWrite bool
	}{
		{"json-write", false, true},
		{"sse-write", true, true},
		{"sse-flush", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writer := failingResponseWriter{httptest.NewRecorder(), tc.failWrite}
			kind, body := "application/json", `{"output":[]}`
			if tc.stream {
				kind, body = "text/event-stream", "data: {\"type\":\"response.created\"}\n\n"
			}
			committed := false
			err := copyResponse(writer, Response{StatusCode: 200, ContentType: kind, Body: strings.NewReader(body)}, tc.stream, &committed, "responses")
			requireSafeStreamFailure(t, err, "client_write_failed")
		})
	}
}

func TestHTTPReadFailuresAreTypedAndBounded(t *testing.T) {
	for _, tc := range []struct {
		name, code string
		body       io.Reader
	}{
		{"read", "upstream_stream_failed", streamFailureReader{}},
		{"event-limit", "response_too_large", strings.NewReader("data: " + strings.Repeat("x", MaxResponsesEventBytes))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			committed := false
			err := copyResponse(httptest.NewRecorder(), Response{StatusCode: 200, ContentType: "text/event-stream", Body: tc.body}, true, &committed, "responses")
			requireSafeStreamFailure(t, err, tc.code)
		})
	}
}
