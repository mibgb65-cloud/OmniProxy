package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// policyBinding 是带透明判定的宿主桩：能否透明转发由用例决定。
type policyBinding struct {
	session     *stubSession
	transparent bool
}

func (b policyBinding) OpenPassthrough(context.Context, PassthroughRequest) (PassthroughSession, error) {
	return b.session, nil
}

func (b policyBinding) Transparent(context.Context, PassthroughRequest) bool { return b.transparent }

// TestPassthroughFallsBackToProtocolWhenNotTransparent 断言宿主判定不能按字节转发时，
// 同一个请求改走协议入口：正文被解析、模型交给上游，透明会话一次都没有取号。
func TestPassthroughFallsBackToProtocolWhenNotTransparent(t *testing.T) {
	session := &stubSession{attempts: []PassthroughAttempt{{BaseURL: "https://upstream.invalid"}}}
	var gotModel, gotProtocol string
	upstream := upstreamFunc(func(_ context.Context, req Request, consume func(Response) error) error {
		gotModel, gotProtocol = req.Model, req.Protocol
		return consume(Response{StatusCode: 200, ContentType: "application/json", Body: strings.NewReader(`{"id":"resp_1","output":[]}`)})
	})
	server := New(Options{Port: 0, Passthrough: policyBinding{session: session}, Upstream: upstream, Models: []string{"deepseek-flash"}})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = server.Stop(context.Background()) }()

	request, err := http.NewRequest(http.MethodPost, server.Status().Address+"/codex/v1/responses",
		strings.NewReader(`{"model":"deepseek-flash","input":[{"role":"user","content":"Hi"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.StatusCode, body)
	}
	if gotModel != "deepseek-flash" || gotProtocol != "responses" {
		t.Fatalf("protocol upstream got model=%q protocol=%q", gotModel, gotProtocol)
	}
	if len(session.done) != 0 || session.index != 0 {
		t.Fatalf("transparent session was used: attempts=%d", session.index)
	}
}

// TestPassthroughStaysTransparentWhenAllowed 断言判定为可透明转发时正文与路径原样送出。
func TestPassthroughStaysTransparentWhenAllowed(t *testing.T) {
	var gotPath, gotBody string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotPath, gotBody = r.URL.Path, string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_1"}`))
	}))
	defer upstream.Close()

	session := &stubSession{attempts: []PassthroughAttempt{{
		BaseURL: upstream.URL + "/backend-api/codex",
		Header:  http.Header{"Authorization": {"Bearer pool-token"}},
	}}}
	server := New(Options{Port: 0, Passthrough: policyBinding{session: session, transparent: true}, Upstream: unusedUpstream(t)})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = server.Stop(context.Background()) }()

	const body = `{"model":"gpt-5.5","stream":true}`
	response, err := http.Post(server.Status().Address+"/codex/v1/responses", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
	if gotPath != "/backend-api/codex/responses" || gotBody != body {
		t.Fatalf("unexpected upstream request: path=%q body=%q", gotPath, gotBody)
	}
}

// TestPassthroughKeepsNonProtocolPathsTransparent 断言只有协议端点参与判定：
// 模型目录、分析事件这类没有可路由模型的路径仍按原样透明转发。
func TestPassthroughKeepsNonProtocolPathsTransparent(t *testing.T) {
	var gotPath, gotBody string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotPath, gotBody = r.URL.Path, string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	session := &stubSession{attempts: []PassthroughAttempt{{
		BaseURL: upstream.URL,
		Header:  http.Header{"Authorization": {"Bearer pool-token"}},
	}}}
	server := New(Options{Port: 0, Passthrough: policyBinding{session: session, transparent: false}, Upstream: unusedUpstream(t)})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = server.Stop(context.Background()) }()

	const body = `{"model":"deepseek-flash"}`
	response, err := http.Post(server.Status().Address+"/codex/v1/analytics-events/events", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
	if gotPath != "/analytics-events/events" || gotBody != body {
		t.Fatalf("unexpected upstream request: path=%q body=%q", gotPath, gotBody)
	}
}
