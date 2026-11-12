package proxy

import (
	"testing"
	"time"
)

func TestClientRequestTimeoutWidensOnlyAntigravity(t *testing.T) {
	server := &Server{opts: Options{RequestTimeout: 120 * time.Second}}
	if got := server.clientRequestTimeout("codex", Request{Protocol: "responses"}); got != 120*time.Second {
		t.Fatalf("codex timeout = %v", got)
	}
	if got := server.clientRequestTimeout("antigravity", Request{Protocol: "chat"}); got != antigravityTimeout {
		t.Fatalf("antigravity timeout = %v", got)
	}
	// 配置本身更长时保持配置值，不被常数压低。
	server = &Server{opts: Options{RequestTimeout: 30 * time.Minute}}
	if got := server.clientRequestTimeout("antigravity", Request{Protocol: "chat"}); got != 30*time.Minute {
		t.Fatalf("configured timeout lowered: %v", got)
	}
	// 压缩请求的既有放宽仍然生效。
	server = &Server{opts: Options{RequestTimeout: 120 * time.Second}}
	if got := server.clientRequestTimeout("codex", Request{Protocol: "responses", Compaction: true}); got != 10*time.Minute {
		t.Fatalf("compaction timeout = %v", got)
	}
}
