package proxy

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPassthroughUsageFailureDoesNotTruncateResponse(t *testing.T) {
	for _, media := range []string{"", "text/event-stream", "application/json"} {
		t.Run(media, func(t *testing.T) {
			body := `{"usage":{"input_tokens":12,"output_tokens":4}}`
			if media != "application/json" {
				// 最后一行后不带空事件，保证记账错误不会被下一行的空结果覆盖。
				body = "data: {\"type\":\"response.completed\",\"response\":" + body + "}\n"
			}
			trace := &Trace{observer: &testObserver{fail: true}}
			observed := ObservePassthroughResponse(context.Background(), trace, Response{
				StatusCode: http.StatusOK, ContentType: media, Body: io.NopCloser(strings.NewReader(body)),
			}, "responses")
			got, err := io.ReadAll(observed.Body)
			if err != nil || string(got) != body || !trace.accountingFailed {
				t.Fatalf("记账失败破坏透明响应或被隐藏: body=%q err=%v failed=%v", got, err, trace.accountingFailed)
			}
		})
	}
}

type passthroughAccountingFailure struct {
	stage    string
	attempts []AttemptEvent
	finished RequestEvent
}

func (*passthroughAccountingFailure) Begin(context.Context, RequestEvent) error { return nil }
func (o *passthroughAccountingFailure) Update(context.Context, RequestEvent) error {
	if o.stage == "bind" {
		return errors.New("private storage failure")
	}
	return nil
}
func (o *passthroughAccountingFailure) Finish(_ context.Context, event RequestEvent) error {
	o.finished = event
	return nil
}
func (o *passthroughAccountingFailure) RecordAttempt(_ context.Context, event AttemptEvent) error {
	o.attempts = append(o.attempts, event)
	if o.stage == "finish" && event.State != "pending" {
		return errors.New("private storage failure")
	}
	return nil
}

func TestPassthroughAttemptAccountingFailureIsVisible(t *testing.T) {
	for _, stage := range []string{"bind", "finish"} {
		t.Run(stage, func(t *testing.T) {
			body := `{"usage":{"input_tokens":12,"output_tokens":4}}`
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, body)
			}))
			defer upstream.Close()
			observer := &passthroughAccountingFailure{stage: stage}
			session := &stubSession{attempts: []PassthroughAttempt{{
				BaseURL: upstream.URL, Model: "gpt-test",
				Snapshot: TargetSnapshot{AccountID: "a"},
			}}}
			server := New(Options{Passthrough: stubBinding{session}, Observer: observer})
			request := httptest.NewRequest(http.MethodPost, "/codex/v1/responses", strings.NewReader(`{"model":"gpt-test"}`))
			response := httptest.NewRecorder()
			server.servePassthrough(response, request, "/v1/responses", socketAccess{})
			event := observer.finished
			if event.State != "failed" || event.ErrorCode != "accounting_unavailable" || len(observer.attempts) != 2 {
				t.Fatalf("尝试记账失败未收尾或未显式报告: event=%+v attempts=%+v", event, observer.attempts)
			}
			if stage == "bind" {
				if response.Code != http.StatusServiceUnavailable || observer.attempts[1].State != "failed" ||
					observer.attempts[1].StatusCode != http.StatusOK || observer.attempts[1].ErrorCode != "accounting_unavailable" {
					t.Fatalf("绑定失败没有结束尝试: status=%d attempts=%+v", response.Code, observer.attempts)
				}
			} else if response.Code != http.StatusOK || response.Body.String() != body {
				t.Fatalf("收尾失败破坏已交付响应: status=%d body=%q", response.Code, response.Body.String())
			}
			if len(session.done) != 1 || session.index != 1 {
				t.Fatalf("记账错误导致上游重放或会话未归还: done=%d attempts=%d", len(session.done), session.index)
			}
		})
	}
}

func TestPassthroughFinishFailurePreservesStreamError(t *testing.T) {
	for _, code := range []string{"request_cancelled", "upstream_stream_failed"} {
		observer := &passthroughAccountingFailure{stage: "finish"}
		trace := &Trace{observer: observer, attempt: &AttemptEvent{State: "pending"}}
		if got := finishPassthroughAttempt(trace, http.StatusOK, code); got != code || !trace.accountingFailed {
			t.Fatalf("收尾错误覆盖传输原因: got=%s failed=%v", got, trace.accountingFailed)
		}
	}
}
