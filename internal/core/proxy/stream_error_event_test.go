package proxy

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 单条数据行的分类必须可核实：上游主动报错与转发中断不能共用一个结论。
func TestInspectStreamDataClassifiesFailures(t *testing.T) {
	for _, tc := range []struct {
		name         string
		data         string
		failed       bool
		code         string
		kind         streamDataKind
		eventType    string
		upstreamCode string
	}{
		{name: "created", data: `{"type":"response.created","response":{"id":"r1"}}`},
		{name: "nested-null-error", data: `{"type":"response.in_progress","response":{"error":null}}`},
		{name: "delta", data: `{"type":"response.output_text.delta","delta":"hi","error":null}`},
		{
			name: "response-failed", data: `{"type":"response.failed","response":{"error":{"code":"server_error","message":"private-detail"}}}`,
			failed: true, code: "upstream_error_event", kind: streamDataUpstreamError, eventType: "response.failed", upstreamCode: "server_error",
		},
		{
			name: "error-event", data: `{"type":"error","error":{"type":"overloaded_error"}}`,
			failed: true, code: "upstream_error_event", kind: streamDataUpstreamError, eventType: "error", upstreamCode: "overloaded_error",
		},
		{
			name: "string-error", data: `{"error":"private text"}`,
			failed: true, code: "upstream_error_event", kind: streamDataUpstreamError,
		},
		{
			name: "invalid-json", data: `{"type":"response.created"`,
			failed: true, code: "upstream_stream_failed", kind: streamDataInvalidJSON,
		},
		{
			name: "array-shape", data: `[]`,
			failed: true, code: "upstream_stream_failed", kind: streamDataUnexpectedShape,
		},
		{
			name: "string-shape", data: `"ping"`,
			failed: true, code: "upstream_stream_failed", kind: streamDataUnexpectedShape,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			failure, failed := inspectStreamData([]byte(tc.data))
			if failed != tc.failed {
				t.Fatalf("failed=%v want %v", failed, tc.failed)
			}
			if !tc.failed {
				return
			}
			if failure.code != tc.code || failure.kind != tc.kind {
				t.Fatalf("unexpected classification: %+v", failure)
			}
			if failure.eventType != tc.eventType || failure.upstreamCode != tc.upstreamCode {
				t.Fatalf("unexpected signature: %+v", failure)
			}
			if strings.Contains(tc.data, "private-detail") && (strings.Contains(failure.eventType, "private") || strings.Contains(failure.upstreamCode, "private")) {
				t.Fatal("upstream text leaked into the classification")
			}
		})
	}
}

// 上游错误码只保留可打印短码，换行与超长内容都不能进日志。
func TestSanitizeStreamSignatureBoundsUpstreamText(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{in: "server_error", want: "server_error"},
		{in: "rate_limit.exceeded", want: "rate_limit.exceeded"},
		{in: "bad\r\nprivate-header: x", want: "badprivate-headerx"},
		{in: strings.Repeat("a", 64), want: strings.Repeat("a", 40)},
		{in: "中文错误", want: ""},
	} {
		if got := sanitizeStreamSignature(tc.in); got != tc.want {
			t.Fatalf("sanitizeStreamSignature(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

// 上游报错时留一条可核实的分类；说明文字既不回显也不进日志。
func TestStreamUpstreamErrorEventRecordsSignature(t *testing.T) {
	trace := &Trace{}
	line := `data: {"type":"response.failed","response":{"error":{"code":"server_error","message":"private-detail"}}}`
	done, _, err := streamLineState(line, trace)
	if done {
		t.Fatal("failed event reported as terminal")
	}
	var failure *Error
	if !errors.As(err, &failure) || failure.Code != "upstream_error_event" || failure.StatusCode != http.StatusBadGateway {
		t.Fatalf("unexpected failure: %v", err)
	}
	fields := trace.diagnosticLogFields()
	want := map[string]any{
		"upstream_stream_kind": "upstream_error",
		"upstream_event_type":  "response.failed",
		"upstream_error_code":  "server_error",
	}
	if len(fields) != len(want)*2 {
		t.Fatalf("unexpected log fields: %v", fields)
	}
	for index := 0; index < len(fields); index += 2 {
		key, _ := fields[index].(string)
		if value, ok := want[key]; !ok || fields[index+1] != value {
			t.Fatalf("unexpected log field %v=%v", fields[index], fields[index+1])
		}
	}
	for _, field := range fields[1:] {
		if text, ok := field.(string); ok && strings.Contains(text, "private-detail") {
			t.Fatal("upstream message leaked into the log")
		}
	}
}

// response.incomplete 是 Responses 的收尾事件之一（命中输出上限或内容过滤），
// 上游随后正常关闭流，不能按「提前收流」记失败。
func TestStreamIncompleteEventIsTerminal(t *testing.T) {
	for _, line := range []string{
		`data: {"type":"response.incomplete","response":{"id":"r1","status":"incomplete"}}`,
		`data: {"type":"response.completed","response":{"id":"r1","status":"completed"}}`,
	} {
		trace := &Trace{}
		done, _, err := streamLineState(line, trace)
		if !done || err != nil {
			t.Fatalf("terminal event %q reported done=%t err=%v", line, done, err)
		}
		if fields := trace.diagnosticLogFields(); len(fields) != 0 {
			t.Fatalf("terminal event %q recorded failure fields: %v", line, fields)
		}
	}
	// 整条流以 incomplete 收尾时，转发必须成功结束，而不是断开下游连接。
	committed := false
	recorder := httptest.NewRecorder()
	trace := &Trace{}
	writer := &trackedWriter{ResponseWriter: recorder, trace: trace}
	body := "data: {\"type\":\"response.created\"}\n\n" +
		"data: {\"type\":\"response.incomplete\",\"response\":{\"status\":\"incomplete\"}}\n\n"
	if err := copyStream(writer, strings.NewReader(body), &committed, "responses"); err != nil {
		t.Fatalf("incomplete stream reported failure: %v", err)
	}
	if !committed || !strings.Contains(recorder.Body.String(), "response.incomplete") {
		t.Fatalf("incomplete stream not relayed: committed=%t body=%s", committed, recorder.Body.String())
	}
}

// 上游在终结事件之前结束（干净 EOF 或读错误）时，日志必须能与「数据行不合语法」区分开。
func TestStreamEndsWithoutTerminalEventRecordsIncomplete(t *testing.T) {
	for _, body := range []string{
		"data: {\"type\":\"response.created\"}\n\ndata: {\"type\":\"response.in_progress\"}\n\n",
		"data: {\"type\":\"response.created\"}\n\n",
	} {
		committed := false
		recorder := httptest.NewRecorder()
		trace := &Trace{}
		writer := &trackedWriter{ResponseWriter: recorder, trace: trace}
		err := copyStream(writer, strings.NewReader(body), &committed, "responses")
		var failure *Error
		if !errors.As(err, &failure) || failure.Code != "upstream_stream_failed" || failure.StatusCode != http.StatusBadGateway {
			t.Fatalf("unexpected failure for %q: %v", body, err)
		}
		if !committed {
			t.Fatalf("stream was not committed before upstream ended for %q", body)
		}
		fields := trace.diagnosticLogFields()
		if len(fields) != 2 || fields[0] != "upstream_stream_kind" || fields[1] != "incomplete_stream" {
			t.Fatalf("unexpected log fields for %q: %v", body, fields)
		}
	}
}

// 合法但不是对象的数据行仍按转发中断处理，并记录各自分类。
func TestStreamNonObjectDataStaysUncommittedFailure(t *testing.T) {
	for _, tc := range []struct{ body, kind string }{
		{body: "data: {\"type\":\"response.created\"\n\n", kind: "invalid_json"},
		{body: "data: [1,2,3]\n\n", kind: "unexpected_shape"},
	} {
		committed := false
		recorder := httptest.NewRecorder()
		trace := &Trace{}
		writer := &trackedWriter{ResponseWriter: recorder, trace: trace}
		err := copyStream(writer, strings.NewReader(tc.body), &committed, "responses")
		var failure *Error
		if !errors.As(err, &failure) || failure.Code != "upstream_stream_failed" || failure.StatusCode != http.StatusBadGateway {
			t.Fatalf("unexpected failure for %q: %v", tc.body, err)
		}
		if committed || recorder.Body.Len() != 0 {
			t.Fatalf("failure committed response for %q", tc.body)
		}
		fields := trace.diagnosticLogFields()
		if len(fields) != 2 || fields[0] != "upstream_stream_kind" || fields[1] != tc.kind {
			t.Fatalf("unexpected log fields for %q: %v", tc.body, fields)
		}
	}
}
