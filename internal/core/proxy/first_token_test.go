package proxy

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFirstTokenRecognizesContentInsteadOfMetadata(t *testing.T) {
	for _, tc := range []struct {
		protocol, body string
		want           bool
	}{
		{"chat", `{"choices":[{"delta":{"role":"assistant","content":""}}]}`, false},
		{"chat", `{"choices":[{"delta":{"content":"hello"}}]}`, true},
		{"chat", `{"choices":[{"delta":{"content":[{"type":"text","text":"hello"}]}}]}`, true},
		{"chat", `{"choices":[{"delta":{"reasoning_content":"think"}}]}`, true},
		{"chat", `{"choices":[{"delta":{"tool_calls":[{"function":{"name":"tool"}}]}}]}`, true},
		{"chat", `{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{}"}}]}}]}`, true},
		{"chat", `{"choices":[],"usage":{"completion_tokens":3}}`, false},
		{"responses", `{"type":"response.created","response":{"id":"r"}}`, false},
		{"responses", `{"type":"response.output_text.delta","delta":""}`, false},
		{"responses", `{"type":"response.output_text.delta","delta":"hello"}`, true},
		{"responses", `{"type":"response.output_text.delta","delta":[{"type":"text","text":"invalid delta type"}]}`, false},
		{"responses", `{"type":"response.content_part.added","part":{"type":"output_text","text":"hello"}}`, true},
		{"responses", `{"type":"response.content_part.added","part":{"type":"metadata","text":"metadata"}}`, false},
		{"responses", `{"type":"response.reasoning_summary_text.delta","delta":"think"}`, true},
		{"responses", `{"type":"response.function_call_arguments.delta","delta":"{}"}`, true},
		{"responses", `{"type":"response.output_item.added","item":{"type":"message","content":[]}}`, false},
		{"responses", `{"type":"response.output_item.added","item":{"type":"function_call","name":"tool"}}`, true},
		{"responses", `{"type":"response.completed","response":{"usage":{"output_tokens":1}}}`, false},
		{"messages", `{"type":"ping"}`, false},
		{"messages", `{"type":"message_start","message":{"content":[]}}`, false},
		{"messages", `{"type":"content_block_start","content_block":{"type":"text","text":""}}`, false},
		{"messages", `{"type":"content_block_start","content_block":{"type":"metadata","text":"metadata"}}`, false},
		{"messages", `{"type":"content_block_start","content_block":{"type":"tool_use","name":"tool"}}`, true},
		{"messages", `{"type":"content_block_delta","delta":{"type":"text_delta","text":"hi"}}`, true},
		{"messages", `{"type":"content_block_delta","delta":{"type":"thinking_delta","thinking":"think"}}`, true},
		{"messages", `{"type":"content_block_delta","delta":{"type":"input_json_delta","partial_json":"{}"}}`, true},
		{"messages", `{"type":"content_block_delta","delta":{"type":"signature_delta","signature":"opaque"}}`, false},
		{"unknown", `{"delta":"hello"}`, false},
		{"chat", `invalid`, false},
	} {
		if got := hasModelOutput(tc.protocol, []byte(tc.body)); got != tc.want {
			t.Fatalf("%s %s: %v", tc.protocol, tc.body, got)
		}
	}
}

type firstTokenFlushFailure struct{ *httptest.ResponseRecorder }

func (w firstTokenFlushFailure) FlushError() error { return errors.New("write failed") }

func TestFirstTokenWaitsForCompleteAndFlushedSSEFrame(t *testing.T) {
	content := "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n"
	for _, tc := range []struct {
		name, body       string
		failFlush, known bool
	}{
		{"metadata", ": heartbeat\n\ndata: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\ndata: [DONE]\n\n", false, false},
		{"unterminated", content, false, false},
		{"flushed", content + "\ndata: [DONE]\n\n", false, true},
		{"client-failure", content + "\ndata: [DONE]\n\n", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			trace := &Trace{event: RequestEvent{StartedAt: time.Now().Add(-100 * time.Millisecond), Protocol: "chat", Stream: true}}
			recorder := httptest.NewRecorder()
			writer := &trackedWriter{ResponseWriter: recorder, trace: trace}
			if tc.failFlush {
				writer.ResponseWriter = firstTokenFlushFailure{recorder}
			}
			committed := false
			_ = copyStream(writer, strings.NewReader(tc.body), &committed)
			if (trace.event.FirstTokenMS != nil) != tc.known {
				t.Fatal("incorrect first output", trace.event.FirstTokenMS)
			}
			if tc.known {
				before := *trace.event.FirstTokenMS
				if before < 100 {
					t.Fatal("wrong request start")
				}
				trace.markFirstToken()
				if *trace.event.FirstTokenMS != before {
					t.Fatal("first output overwritten")
				}
			}
		})
	}
}

func TestHTTPFirstTokenIsRecordedOnlyForStreamingContent(t *testing.T) {
	for _, tc := range []struct {
		protocol, path, input, output string
		stream                        bool
	}{
		{"chat", "chat/completions", validBody, `{"choices":[{"message":{"content":"hi"}}]}`, false},
		{"chat", "chat/completions", strings.TrimSuffix(validBody, "}") + `,"stream":true}`, "data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\ndata: [DONE]\n\n", true},
		{"responses", "responses", `{"model":"deepseek-flash","input":"hi","stream":true}`, "data: {\"type\":\"response.created\"}\n\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"hi\"}\n\ndata: {\"type\":\"response.completed\",\"response\":{\"id\":\"r\",\"output\":[]}}\n\n", true},
		{"messages", "messages", strings.TrimSuffix(validBody, "}") + `,"max_tokens":10,"stream":true}`, "data: {\"type\":\"ping\"}\n\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"thinking_delta\",\"thinking\":\"hi\"}}\n\ndata: {\"type\":\"message_stop\"}\n\n", true},
	} {
		t.Run(tc.path+map[bool]string{false: "-json", true: "-stream"}[tc.stream], func(t *testing.T) {
			finished := make(chan RequestEvent, 1)
			server := New(Options{Port: 0, Models: []string{"deepseek-flash"}, Observer: firstTokenObserver{finished}, Upstream: upstreamFunc(func(_ context.Context, _ Request, consume func(Response) error) error {
				contentType := "application/json"
				if tc.stream {
					contentType = "text/event-stream"
				}
				return consume(Response{StatusCode: 200, ContentType: contentType, Body: strings.NewReader(tc.output)})
			})})
			if _, err := server.Start(); err != nil {
				t.Fatal(err)
			}
			defer server.Stop(context.Background())
			client := http.Client{Timeout: 3 * time.Second}
			response, err := client.Post(server.Status().Address+"/v1/"+tc.path, "application/json", strings.NewReader(tc.input))
			if err != nil {
				t.Fatal(err)
			}
			_, err = io.ReadAll(response.Body)
			response.Body.Close()
			if err != nil || response.StatusCode != 200 {
				t.Fatal("forwarding failed", err, response.StatusCode)
			}
			select {
			case event := <-finished:
				if (event.FirstTokenMS != nil) != tc.stream || event.Protocol != tc.protocol || event.State != "success" {
					t.Fatal("incorrect timing", event)
				}
				if tc.stream && (*event.FirstTokenMS < 0 || *event.FirstTokenMS > event.LatencyMS) {
					t.Fatal("timing outside lifecycle")
				}
			case <-time.After(time.Second):
				t.Fatal("request was not recorded")
			}
		})
	}
}

type firstTokenObserver struct{ finished chan RequestEvent }

func (o firstTokenObserver) Begin(context.Context, RequestEvent) error  { return nil }
func (o firstTokenObserver) Update(context.Context, RequestEvent) error { return nil }
func (o firstTokenObserver) Finish(_ context.Context, event RequestEvent) error {
	o.finished <- event
	return nil
}
