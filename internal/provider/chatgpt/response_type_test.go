package chatgpt

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"omniproxy/internal/core/proxy"
)

func TestResponseTypeInferenceRejectsUntrustedOrIncompleteData(t *testing.T) {
	cases := []struct {
		name, kind, body string
		status           int
	}{
		{"html", "", "<!doctype html><html>private</html>", 200},
		{"json", "", `{"error":{"message":"private"}}`, 200},
		{"empty", "", "", 200},
		{"invalid-json", "", "data: {broken}\n\n", 200},
		{"unknown-event", "", "data: {\"type\":\"unrelated\",\"response\":{\"id\":\"one\"}}\n\n", 200},
		{"missing-id", "", "data: {\"type\":\"response.created\",\"response\":{}}\n\n", 200},
		{"not-terminated", "", "data: {\"type\":\"response.created\",\"response\":{\"id\":\"one\"}}\n", 200},
		{"done-only", "", "data: [DONE]\n\n", 200},
		{"mismatched-event", "", "event: error\ndata: {\"type\":\"response.created\",\"response\":{\"id\":\"one\"}}\n\n", 200},
		{"multiple-data", "", "data: {}\ndata: {}\n\n", 200},
		{"explicit-json", "application/json", responseEvents, 200},
		{"explicit-html", "text/html", responseEvents, 200},
		{"explicit-plain", "text/plain", responseEvents, 200},
		{"malformed-type", "invalid; header", responseEvents, 200},
		{"status-429", "", responseEvents, 429},
		{"status-401", "", responseEvents, 401},
		{"status-500", "", responseEvents, 500},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := inferResponseType(proxy.Response{StatusCode: tt.status, ContentType: tt.kind, Body: strings.NewReader(tt.body)})
			if r.ContentTypeInferred || r.ContentType != tt.kind || r.StatusCode != tt.status {
				t.Fatal("untrusted response accepted", r.ContentType)
			}
			raw, e := io.ReadAll(r.Body)
			if e != nil || string(raw) != tt.body {
				t.Fatal("body was changed", e)
			}
		})
	}
	if r := inferResponseType(proxy.Response{StatusCode: 200}); r.ContentTypeInferred {
		t.Fatal("nil body accepted")
	}
}

func TestInferencePreservesPrefixAndDoesNotWaitForTail(t *testing.T) {
	first := "\n: keepalive\nevent: response.created\ndata: {\"type\":\"response.created\",\"response\":{\"id\":\"one\"}}\n\n"
	reader := &gatedReader{first: strings.NewReader(first), tail: strings.NewReader(responseEvents)}
	response := inferResponseType(proxy.Response{StatusCode: 200, Body: reader})
	if !response.ContentTypeInferred || reader.premature {
		t.Fatal("inference waited for the response tail")
	}
	reader.allowed = true
	raw, e := io.ReadAll(response.Body)
	if e != nil || string(raw) != first+responseEvents {
		t.Fatal("prefetched bytes changed", e)
	}
	explicit := inferResponseType(proxy.Response{StatusCode: 200, ContentType: "text/event-stream; charset=utf-8", Body: strings.NewReader(responseEvents)})
	if explicit.ContentTypeInferred || explicit.ContentType != "text/event-stream; charset=utf-8" {
		t.Fatal("explicit SSE changed")
	}
}

type gatedReader struct {
	first, tail        *strings.Reader
	allowed, premature bool
}

func (r *gatedReader) Read(p []byte) (int, error) {
	if r.first.Len() > 0 {
		return r.first.Read(p)
	}
	if !r.allowed {
		r.premature = true
		return 0, errors.New("tail is still open")
	}
	return r.tail.Read(p)
}

type countedReader struct {
	io.Reader
	n int
}

func (r *countedReader) Read(p []byte) (int, error) { n, e := r.Reader.Read(p); r.n += n; return n, e }
func TestInferencePrefixBoundAndReadError(t *testing.T) {
	reader := &countedReader{Reader: strings.NewReader(":" + strings.Repeat("x", maxSSEPrefixBytes*2))}
	response := inferResponseType(proxy.Response{StatusCode: 200, Body: reader})
	if response.ContentTypeInferred || reader.n > maxSSEPrefixBytes {
		t.Fatal("unbounded prefix", reader.n)
	}
	broken := &gatedReader{first: strings.NewReader("data: {}"), tail: strings.NewReader("")}
	if r := inferResponseType(proxy.Response{StatusCode: 200, Body: broken}); r.ContentTypeInferred {
		t.Fatal("read failure accepted")
	}
}

func TestInferenceHandlesFragmentedEvents(t *testing.T) {
	for _, body := range []string{responseEvents, "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"one\",\"output\":[]}}\n\n"} {
		response := inferResponseType(proxy.Response{StatusCode: 200, Body: iotest.OneByteReader(strings.NewReader(body))})
		if !response.ContentTypeInferred {
			t.Fatal("fragmented lifecycle event rejected")
		}
		raw, e := io.ReadAll(response.Body)
		if e != nil || string(raw) != body {
			t.Fatal("fragmented bytes changed", e)
		}
	}
}

func TestInferenceDiagnosticsPreserveRejectedBodies(t *testing.T) {
	for _, tt := range []struct{ name, body, reason string }{
		{"empty", "", "empty"},
		{"json", `{"error":{"message":"private diagnostic marker"}}`, "non_sse_json"},
		{"html", "<html>private diagnostic marker</html>", "non_sse_html"},
		{"truncated", "data: {}\n", "incomplete_event"},
		{"unknown-field", "private-field: private diagnostic marker\n", "unexpected_field"},
		{"multi-data", "data: {}\ndata: {}\n\n", "multiple_data"},
		{"invalid-json", "data: broken\n\n", "invalid_json"},
		{"event-mismatch", "event: error\ndata: {\"type\":\"response.created\",\"response\":{\"id\":\"one\"}}\n\n", "event_mismatch"},
		{"unknown-event", "data: {\"type\":\"private diagnostic marker\"}\n\n", "unexpected_event"},
		{"missing-id", "data: {\"type\":\"response.created\",\"response\":{}}\n\n", "invalid_response_id"},
		{"invalid-error", "data: {\"type\":\"error\",\"error\":null}\n\n", "invalid_error"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			response := inferResponseType(proxy.Response{StatusCode: 200, Body: strings.NewReader(tt.body)})
			if response.ContentTypeInferred || response.ContentTypeInferenceReason != tt.reason {
				t.Fatalf("unexpected classification: inferred=%t reason=%s", response.ContentTypeInferred, response.ContentTypeInferenceReason)
			}
			if response.ContentTypeInferenceBytes != len(tt.body) {
				t.Fatal("unexpected prefix size")
			}
			raw, err := io.ReadAll(response.Body)
			if err != nil || string(raw) != tt.body {
				t.Fatal("diagnostics changed response bytes")
			}
		})
	}
}

func TestInferenceDiagnosticsDistinguishLargeFirstEventAndReadFailure(t *testing.T) {
	// response.created 会携带工具定义；边界诊断必须区分完整大事件和截断前缀。
	start := "data: {\"type\":\"response.created\",\"response\":{\"id\":\"one\",\"tools\":[{\"description\":\""
	end := "\"}]}}\n\n"
	for _, size := range []int{maxSSEPrefixBytes, maxSSEPrefixBytes + 1} {
		body := start + strings.Repeat("x", size-len(start)-len(end)) + end
		response := inferResponseType(proxy.Response{StatusCode: 200, Body: strings.NewReader(body)})
		reason := "verified"
		if size > maxSSEPrefixBytes {
			reason = "prefix_limit"
		}
		if response.ContentTypeInferenceReason != reason || response.ContentTypeInferred != (reason == "verified") {
			t.Fatal("first event bound misclassified", response.ContentTypeInferenceReason)
		}
		if response.ContentTypeInferenceBytes != maxSSEPrefixBytes {
			t.Fatal("unbounded diagnostic prefix")
		}
		raw, err := io.ReadAll(response.Body)
		if err != nil || string(raw) != body {
			t.Fatal("large body changed")
		}
	}
	broken := &gatedReader{first: strings.NewReader("data: {}"), tail: strings.NewReader("")}
	response := inferResponseType(proxy.Response{StatusCode: 200, Body: broken})
	if response.ContentTypeInferenceReason != "read_error" || response.ContentTypeInferred {
		t.Fatal("read error misclassified")
	}
	if inferResponseType(proxy.Response{StatusCode: 200}).ContentTypeInferenceReason != "missing_body" {
		t.Fatal("missing body misclassified")
	}
	for _, response := range []proxy.Response{{StatusCode: 429}, {StatusCode: 200, ContentType: "application/json"}} {
		if inferResponseType(response).ContentTypeInferenceReason != "" {
			t.Fatal("explicit response was inferred")
		}
	}
}
