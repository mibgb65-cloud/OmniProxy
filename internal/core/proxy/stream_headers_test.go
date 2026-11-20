package proxy

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// silentThenReader 在 delay 之前不返回任何字节，模拟首个可见 token 之前的思考期。
type silentThenReader struct {
	delay time.Duration
	body  string
	once  sync.Once
}

func (r *silentThenReader) Read(p []byte) (int, error) {
	r.once.Do(func() { time.Sleep(r.delay) })
	if r.body == "" {
		return 0, io.EOF
	}
	n := copy(p, r.body)
	r.body = r.body[n:]
	return n, nil
}

type scriptedRead struct {
	chunk string
	delay time.Duration
}

// scriptedReader 按脚本分块返回字节，用来制造「事件尚未结束」的停顿。
type scriptedReader struct {
	steps []scriptedRead
	index int
}

func (r *scriptedReader) Read(p []byte) (int, error) {
	if r.index >= len(r.steps) {
		return 0, io.EOF
	}
	step := r.steps[r.index]
	r.index++
	if step.delay > 0 {
		time.Sleep(step.delay)
	}
	return copy(p, step.chunk), nil
}

// swapStreamTimings 缩短等待，避免测试真的等 3 秒／10 秒；返回恢复函数。
func swapStreamTimings(grace, heartbeat time.Duration) func() {
	previousGrace, previousHeartbeat := streamHeaderGraceDelay, streamHeartbeatPeriod
	streamHeaderGraceDelay, streamHeartbeatPeriod = grace, heartbeat
	return func() {
		streamHeaderGraceDelay, streamHeartbeatPeriod = previousGrace, previousHeartbeat
	}
}

func TestStreamHeadersArriveBeforeFirstToken(t *testing.T) {
	defer swapStreamTimings(30*time.Millisecond, 20*time.Millisecond)()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		committed := false
		body := &silentThenReader{delay: 400 * time.Millisecond, body: "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\ndata: [DONE]\n\n"}
		if err := copyStream(w, body, &committed); err != nil {
			t.Errorf("copyStream: %v", err)
		}
	}))
	defer server.Close()

	// 修复前响应头要等到首个 data 行（400ms），这里 200ms 就会超时。
	client := &http.Client{Transport: &http.Transport{ResponseHeaderTimeout: 200 * time.Millisecond}}
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("headers not delivered before first token: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("unexpected response: %d %q", response.StatusCode, response.Header.Get("Content-Type"))
	}
	raw, err := io.ReadAll(response.Body)
	if err != nil || !strings.Contains(string(raw), "\"content\":\"hi\"") {
		t.Fatalf("stream body incomplete: %v %q", err, raw)
	}
}

func TestStreamHeartbeatKeepsSilentStreamOpen(t *testing.T) {
	defer swapStreamTimings(10*time.Millisecond, 25*time.Millisecond)()

	committed := make(chan bool, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		sent := false
		body := &silentThenReader{delay: 200 * time.Millisecond, body: "data: {\"choices\":[]}\n\ndata: [DONE]\n\n"}
		if err := copyStream(w, body, &sent); err != nil {
			t.Errorf("copyStream: %v", err)
		}
		committed <- sent
	}))
	defer server.Close()

	response, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), streamHeartbeatFrame) {
		t.Fatalf("silent stream sent no heartbeat: %q", raw)
	}
	if !strings.Contains(string(raw), "data: [DONE]") {
		t.Fatalf("terminal event lost: %q", raw)
	}
	if !<-committed {
		t.Fatal("committed flag not set after headers were sent")
	}
}

// 心跳不能插进正在拼装的事件：注释行自带的空行会提前终结该事件。
// 事件之间的注释行是合法 SSE，不影响客户端解析，因此只断言事件本身完整。
func TestStreamHeartbeatDoesNotSplitEvent(t *testing.T) {
	defer swapStreamTimings(10*time.Millisecond, 20*time.Millisecond)()

	recorder := httptest.NewRecorder()
	committed := false
	body := &scriptedReader{steps: []scriptedRead{
		{chunk: "data: {\"choices\":[]}\n"},
		{chunk: "\n", delay: 80 * time.Millisecond},
		{chunk: "data: [DONE]\n\n"},
	}}
	if err := copyStream(recorder, body, &committed); err != nil {
		t.Fatal(err)
	}
	got := recorder.Body.String()
	if !strings.Contains(got, "data: {\"choices\":[]}\n\n") {
		t.Fatalf("heartbeat split the event: %q", got)
	}
	if !strings.HasSuffix(got, "data: [DONE]\n\n") {
		t.Fatalf("terminal event lost: %q", got)
	}
}

func TestStreamImmediateFailureStaysUncommitted(t *testing.T) {
	committed := false
	recorder := httptest.NewRecorder()
	body := strings.NewReader("data: {\"type\":\"response.failed\",\"response\":{\"error\":{\"message\":\"private\"}}}\n\n")
	err := copyStream(recorder, body, &committed, "responses")
	var failure *Error
	if !errors.As(err, &failure) || failure.Code != "upstream_error_event" || failure.StatusCode != http.StatusBadGateway {
		t.Fatalf("unexpected failure: %v", err)
	}
	if committed || recorder.Body.Len() != 0 {
		t.Fatalf("failure committed response: committed=%v body=%q", committed, recorder.Body.String())
	}
}

// 等上游响应头期间就要下发响应头并保活，之后由 copyStream 接管且不重复写状态码。
func TestStreamHeaderKeeperHandsOffToCopyStream(t *testing.T) {
	defer swapStreamTimings(30*time.Millisecond, 20*time.Millisecond)()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		committed := false
		stop := startStreamHeaderKeeper(w, &committed)
		// 上游响应头尚未到达：桌面端在这段时间里只可能等到响应头超时。
		time.Sleep(120 * time.Millisecond)
		stop()
		stop() // 停止必须可重复调用。
		if err := copyStream(w, strings.NewReader("data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\ndata: [DONE]\n\n"), &committed); err != nil {
			t.Errorf("copyStream: %v", err)
		}
	}))
	defer server.Close()

	client := &http.Client{Transport: &http.Transport{ResponseHeaderTimeout: 80 * time.Millisecond}}
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("headers not delivered while upstream headers were pending: %v", err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("unexpected response: %d %q", response.StatusCode, response.Header.Get("Content-Type"))
	}
	if !strings.Contains(body, streamHeartbeatFrame) {
		t.Fatalf("waiting period sent no heartbeat: %q", body)
	}
	if strings.Count(body, "data: [DONE]") != 1 || !strings.HasSuffix(body, "data: [DONE]\n\n") {
		t.Fatalf("stream body incomplete or replayed: %q", body)
	}
}

func TestEarlyStreamHeadersOnlyForAntigravityStreams(t *testing.T) {
	for _, tc := range []struct {
		client string
		stream bool
		want   bool
	}{
		{"antigravity", true, true},
		{"antigravity", false, false},
		{"codex", true, false},
		{"harness-desktop", true, false},
		{"", true, false},
	} {
		if got := needsEarlyStreamHeaders(tc.client, tc.stream); got != tc.want {
			t.Errorf("needsEarlyStreamHeaders(%q,%v) = %v, want %v", tc.client, tc.stream, got, tc.want)
		}
	}
}

// earlyHeaderServer 模拟「上游迟迟不返回响应头」：Forward 在 delay 之后才回调 consume。
func earlyHeaderServer(t *testing.T, client string, delay time.Duration) *Server {
	t.Helper()
	server := New(Options{
		Port:   0,
		Models: []string{"deepseek-flash"},
		Authorize: func(context.Context, string) (string, bool) {
			return client, true
		},
		Upstream: upstreamFunc(func(_ context.Context, _ Request, consume func(Response) error) error {
			time.Sleep(delay)
			return consume(Response{StatusCode: http.StatusOK, ContentType: "text/event-stream", Body: strings.NewReader("data: {\"choices\":[{\"delta\":{\"content\":\"OK\"}}]}\n\ndata: [DONE]\n\n")})
		}),
	})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if _, err := server.Stop(ctx); err != nil {
			t.Error(err)
		}
	})
	return server
}

func streamRequest(t *testing.T, server *Server, headerTimeout time.Duration) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, server.Status().Address+"/v1/chat/completions", strings.NewReader(strings.TrimSuffix(validBody, "}")+`,"stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Transport: &http.Transport{ResponseHeaderTimeout: headerTimeout}, Timeout: 3 * time.Second}
	return client.Do(req)
}

func TestAntigravityStreamHeadersPrecedeUpstreamHeaders(t *testing.T) {
	defer swapStreamTimings(30*time.Millisecond, 20*time.Millisecond)()
	server := earlyHeaderServer(t, "antigravity", 300*time.Millisecond)

	// 桌面端的响应头预算在这里被压缩成 100ms，修复前必然超时。
	response, err := streamRequest(t, server, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Antigravity waited for upstream headers: %v", err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(raw), streamHeartbeatFrame) || !strings.Contains(string(raw), "data: [DONE]") {
		t.Fatalf("unexpected stream: %d %q", response.StatusCode, raw)
	}
}

func TestOtherClientsStillWaitForUpstreamHeaders(t *testing.T) {
	defer swapStreamTimings(30*time.Millisecond, 20*time.Millisecond)()
	server := earlyHeaderServer(t, "codex", 300*time.Millisecond)

	if _, err := streamRequest(t, server, 100*time.Millisecond); err == nil {
		t.Fatal("early headers changed another client's error semantics")
	}
}
