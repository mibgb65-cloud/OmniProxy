package gateway

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"omniproxy/internal/application/activity"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/provider/chatgpt"
)

type brokenStreamReader struct{}

func (brokenStreamReader) Read([]byte) (int, error) {
	return 0, errors.New("private-stream-reader-detail")
}

func TestCodexStreamIOFailuresKeepSafeClassificationAndSingleAttempt(t *testing.T) {
	for _, transport := range []string{"http", "websocket"} {
		for _, scenario := range []string{"read-error", "event-limit"} {
			t.Run(transport+"/"+scenario, func(t *testing.T) {
				s, _, observer, _ := subscriptionQuotaFixture(t)
				var calls atomic.Int32
				s.chatgpt = chatgpt.New(roundTripFunc(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					var tail io.Reader = brokenStreamReader{}
					if scenario == "event-limit" {
						tail = strings.NewReader("data: " + strings.Repeat("x", proxy.MaxResponsesEventBytes))
					}
					head := "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_partial\"}}\n\n"
					return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(io.MultiReader(strings.NewReader(head), tail))}, nil
				}))
				completed, _, raw := requestQuotaGateway(t, s, transport, "gpt-test")
				if completed || strings.Contains(raw, "private-stream-reader-detail") || strings.Contains(raw, "credential_unavailable") {
					t.Fatal("stream failure leaked details or became a credential failure")
				}
				page, err := observer.Requests(context.Background(), activity.RequestQuery{Limit: 10})
				if err != nil || len(page.Items) != 1 {
					t.Fatal("stream failure record missing", err)
				}
				item, err := observer.Request(context.Background(), page.Items[0].ID)
				want := "upstream_stream_failed"
				if scenario == "event-limit" {
					want = "response_too_large"
				}
				status := 502
				if transport == "http" {
					// HTTP 已送出 SSE 响应头，保留真实 200，同时用失败状态与错误码表达中断。
					status = 200
				}
				if err != nil || item.StatusCode != status || item.State != "failed" || item.ErrorCode != want || item.AttemptCount != 1 || calls.Load() != 1 {
					t.Fatalf("stream failure classification or replay changed: request=%+v calls=%d err=%v", item, calls.Load(), err)
				}
				if len(item.Attempts) != 1 || item.Attempts[0].ErrorCode != want || item.Attempts[0].StatusCode != 200 || item.Attempts[0].State != "failed" {
					t.Fatal("upstream status or failure classification lost", item.Attempts)
				}
			})
		}
	}
}
