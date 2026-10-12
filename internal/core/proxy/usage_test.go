package proxy

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type testObserver struct {
	events []RequestEvent
	fail   bool
}

func (o *testObserver) Begin(context.Context, RequestEvent) error { return nil }
func (o *testObserver) Update(_ context.Context, e RequestEvent) error {
	if o.fail {
		return errors.New("private storage error")
	}
	o.events = append(o.events, e)
	return nil
}
func (o *testObserver) Finish(_ context.Context, e RequestEvent) error {
	o.events = append(o.events, e)
	return nil
}
func TestUsageProtocolContracts(t *testing.T) {
	for _, tc := range []struct {
		protocol, body string
		stream         bool
		input, output  int64
		complete       bool
	}{
		{"chat", `{"usage":{"prompt_tokens":12,"completion_tokens":5,"prompt_cache_hit_tokens":8}}`, false, 12, 5, true},
		{"responses", `{"type":"response.completed","response":{"usage":{"input_tokens":20,"output_tokens":7,"output_tokens_details":{"reasoning_tokens":3}}}}`, true, 20, 7, true},
		{"messages", `{"usage":{"input_tokens":10,"cache_creation_input_tokens":3,"cache_read_input_tokens":5,"output_tokens":4}}`, false, 18, 4, true},
		{"chat", `{"usage":{"prompt_tokens":0,"completion_tokens":0}}`, false, 0, 0, true},
		{"chat", `{"usage":{"prompt_tokens":1.2,"completion_tokens":-1}}`, false, 0, 0, false},
		{"chat", `{"choices":[{}]}`, false, 0, 0, false},
	} {
		u := ParseUsage([]byte(tc.body), tc.protocol, tc.stream)
		if u.Input != tc.input || u.Output != tc.output || u.Complete != tc.complete {
			t.Fatalf("usage contract: %s %+v", tc.protocol, u)
		}
	}
}
func TestStreamingUsageObservedBeforeForwardingAndRepeatedSnapshots(t *testing.T) {
	observer := &testObserver{}
	trace := &Trace{observer: observer}
	body := "data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":10,\"output_tokens\":1}}}\n\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":7}}\n\n"
	response := ObserveResponse(context.Background(), trace, Response{StatusCode: 200, ContentType: "text/event-stream", Body: strings.NewReader(body)}, "messages")
	output, err := io.ReadAll(response.Body)
	if err != nil || string(output) != body {
		t.Fatal("stream modified", err)
	}
	if !trace.event.Usage.Complete || trace.event.Usage.Input != 10 || trace.event.Usage.Output != 7 {
		t.Fatal("incremental snapshots not combined")
	}
	observer.fail = true
	response = ObserveResponse(context.Background(), trace, Response{StatusCode: 200, ContentType: "application/json", Body: strings.NewReader(`{"usage":{"prompt_tokens":4,"completion_tokens":2}}`)}, "chat")
	if _, err = io.ReadAll(response.Body); err == nil {
		t.Fatal("storage failure silently ignored")
	}
}

func TestFinalUsageWithoutTrailingNewline(t *testing.T) {
	observer := &testObserver{}
	trace := &Trace{observer: observer}
	response := ObserveResponse(context.Background(), trace, Response{StatusCode: 200, ContentType: "text/event-stream", Body: strings.NewReader(`data: {"type":"response.completed","response":{"usage":{"input_tokens":8,"output_tokens":3}}}`)}, "responses")
	if _, err := io.ReadAll(response.Body); err != nil {
		t.Fatal(err)
	}
	if !trace.event.Usage.Complete || trace.event.Usage.Input != 8 || trace.event.Usage.Output != 3 {
		t.Fatal("final usage was lost")
	}
}
