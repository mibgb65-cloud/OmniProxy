package chatgpt

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

const responseEvents = "event: response.created\r\ndata: {\"type\":\"response.created\",\"response\":{\"id\":\"resp-test\"}}\r\n\r\ndata: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp-test\",\"output\":[]}}\n\n"

func TestMissingContentTypeAcceptsValidatedResponsesSSE(t *testing.T) {
	client := New(transportFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(responseEvents))}, nil
	}))
	err := client.Forward(context.Background(), []byte("placeholder"), "workspace", proxy.Request{Body: []byte(`{"model":"gpt-test","stream":true}`)}, func(resp proxy.Response) error {
		if resp.ContentType != "text/event-stream" {
			t.Fatalf("missing SSE compatibility: %q", resp.ContentType)
		}
		raw, e := io.ReadAll(resp.Body)
		if e != nil || string(raw) != responseEvents {
			t.Fatal("inference lost or duplicated stream bytes", e)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
