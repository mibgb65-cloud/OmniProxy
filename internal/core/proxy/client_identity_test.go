package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientIdentityBoundaries(t *testing.T) {
	for _, v := range []string{"", "client/1.0 (windows)", strings.Repeat("x", 1024)} {
		if !ValidClientUserAgent(v) {
			t.Fatal("valid identity rejected")
		}
	}
	for _, v := range []string{"client\r\nAuthorization: private", "client\x00", "非ASCII", strings.Repeat("x", 1025)} {
		if ValidClientUserAgent(v) {
			t.Fatal("unsafe identity accepted")
		}
	}
}

type beginObserver struct {
	begun RequestEvent
	calls int
}

func (o *beginObserver) Begin(_ context.Context, e RequestEvent) error {
	o.begun, o.calls = e, o.calls+1
	return nil
}
func (o *beginObserver) Update(context.Context, RequestEvent) error { return nil }
func (o *beginObserver) Finish(context.Context, RequestEvent) error { return nil }

// 接入 Key 识别出的客户端必须随首条记录写入，否则请求无法按 Agent 归属。
func TestServeHTTPPassesAuthorizedClientToAccounting(t *testing.T) {
	observer := &beginObserver{}
	server := New(Options{Port: 0, Models: []string{"gpt-test"}, Observer: observer,
		Authorize: func(_ context.Context, key string) (string, bool) { return "codex", key == "agent-key" },
		Upstream:  upstreamFunc(func(_ context.Context, _ Request, _ func(Response) error) error { return nil })})
	body := strings.NewReader(`{"model":"gpt-test","input":[{"role":"user","content":"hi"}]}`)
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:0/v1/responses", body)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Api-Key", "agent-key")
	server.ServeHTTP(httptest.NewRecorder(), request)
	if observer.begun.Client != "codex" {
		t.Fatal("authorized client not recorded", observer.begun.Client)
	}
	// 未通过鉴权的请求直接拒绝，不能产生归属错误或空白记录。
	denied := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:0/v1/responses", strings.NewReader(`{"model":"gpt-test","input":[]}`))
	denied.Header.Set("Content-Type", "application/json")
	denied.Header.Set("X-Api-Key", "wrong-key")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, denied)
	if recorder.Code != http.StatusUnauthorized || observer.calls != 1 {
		t.Fatal("unauthorized request recorded", recorder.Code, observer.calls)
	}
}
