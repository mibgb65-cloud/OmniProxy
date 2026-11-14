package proxy

import (
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
	want := "data: {\"choices\":[]}\n\ndata: [DONE]\n\n"
	if got := recorder.Body.String(); got != want {
		t.Fatalf("heartbeat split the event: %q", got)
	}
}

func TestStreamImmediateFailureStaysUncommitted(t *testing.T) {
	committed := false
	recorder := httptest.NewRecorder()
	body := strings.NewReader("data: {\"type\":\"response.failed\",\"response\":{\"error\":{\"message\":\"private\"}}}\n\n")
	err := copyStream(recorder, body, &committed, "responses")
	var failure *Error
	if !errors.As(err, &failure) || failure.Code != "upstream_stream_failed" || failure.StatusCode != http.StatusBadGateway {
		t.Fatalf("unexpected failure: %v", err)
	}
	if committed || recorder.Body.Len() != 0 {
		t.Fatalf("failure committed response: committed=%v body=%q", committed, recorder.Body.String())
	}
}
