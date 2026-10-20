package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func responseHistory(count int) []json.RawMessage {
	items := make([]json.RawMessage, count)
	for i := range items {
		switch i % 5 {
		case 0:
			items[i] = json.RawMessage(`{"role":"user","content":"history"}`)
		case 1:
			items[i] = json.RawMessage(`{"type":"reasoning","encrypted_content":"opaque-history"}`)
		case 2:
			items[i] = json.RawMessage(fmt.Sprintf(`{"type":"function_call","call_id":"call_%d","name":"lookup","arguments":"{}"}`, i/5))
		case 3:
			items[i] = json.RawMessage(fmt.Sprintf(`{"type":"function_call_output","call_id":"call_%d","output":"result"}`, i/5))
		case 4:
			items[i] = json.RawMessage(`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"done"}]}`)
		}
	}
	return items
}

func historyRequest(t *testing.T, input []json.RawMessage, socket bool) []byte {
	t.Helper()
	fields := map[string]any{"model": "gpt-test", "input": input}
	if socket {
		fields["type"] = "response.create"
	}
	body, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestResponsesLongHistoryHTTPAndCompaction(t *testing.T) {
	for _, count := range []int{512, 513, 516, 517, 2048} {
		for _, path := range []string{"/v1/responses", "/v1/responses/compact"} {
			t.Run(fmt.Sprintf("%s/%d", path, count), func(t *testing.T) {
				body := historyRequest(t, responseHistory(count), false)
				calls := 0
				server := New(Options{Models: []string{"gpt-test"}, Upstream: upstreamFunc(func(_ context.Context, request Request, consume func(Response) error) error {
					calls++
					if !bytes.Equal(request.Body, body) || request.Compaction != strings.HasSuffix(path, "/compact") {
						t.Error("long history or protocol changed")
					}
					result := `{"output":[]}`
					if request.Compaction {
						result = compactResult
					}
					return consume(Response{StatusCode: 200, ContentType: "application/json", Body: strings.NewReader(result)})
				})})
				request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:0"+path, bytes.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				server.ServeHTTP(recorder, request)
				if recorder.Code != 200 || calls != 1 {
					t.Fatalf("long history rejected: status=%d upstream_calls=%d", recorder.Code, calls)
				}
			})
		}
	}
}

func TestResponsesHistoryStillValidatesJSONAndInputShape(t *testing.T) {
	input, err := json.Marshal(responseHistory(517))
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{
		`{"model":"gpt-test","input":` + string(input) + `,"input":[]}`,
		`{"model":"gpt-test","input":[{"role":"user","role":"assistant"}]}`,
		`{"model":"gpt-test","input":` + strings.Repeat("[", 33) + `0` + strings.Repeat("]", 33) + `}`,
		`{"model":"gpt-test","input":{}}`,
		`{"model":"gpt-test","input":true}`,
		`{"model":"gpt-test"}`,
	} {
		if _, err := ParseProtocolRequest([]byte(body), "/v1/responses"); err == nil {
			t.Fatal("long-history support bypassed input validation")
		}
	}
	for _, input := range []string{`[]`, `"text"`, `null`} {
		if _, err := ParseProtocolRequest([]byte(`{"model":"gpt-test","input":`+input+`}`), "/v1/responses"); err != nil {
			t.Fatal("existing Responses input compatibility changed", err)
		}
	}
}

func TestMessageProtocolsRetainItemLimit(t *testing.T) {
	for _, count := range []int{512, 513} {
		messages := `[` + strings.Repeat(`{"role":"user","content":"text"},`, count-1) + `{"role":"user","content":"text"}]`
		body := []byte(`{"model":"gpt-test","max_tokens":1,"messages":` + messages + `}`)
		for _, path := range []string{"/v1/messages", "/v1/chat/completions"} {
			_, err := ParseProtocolRequest(body, path)
			if (err == nil) != (count == 512) {
				t.Fatalf("message limit changed: path=%s count=%d err=%v", path, count, err)
			}
		}
	}
}
