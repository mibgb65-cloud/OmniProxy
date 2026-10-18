package proxy

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// 显式启用的离线契约测试：只运行本机客户端解析器，模型和认证全部由 loopback 假数据替代。
func TestCodexCLIIsolatedWebSocket(t *testing.T) {
	executable := os.Getenv("OMNIPROXY_CODEX_TEST_EXECUTABLE")
	if executable == "" {
		t.Skip("set OMNIPROXY_CODEX_TEST_EXECUTABLE to run isolated local CLI contract")
	}
	if !filepath.IsAbs(executable) {
		t.Fatal("Codex executable must be an absolute path")
	}
	var calls atomic.Int32
	finished := make(chan RequestEvent, 2)
	server := New(Options{Port: 0, Models: []string{"gpt-test"}, Observer: firstTokenObserver{finished}, Upstream: upstreamFunc(func(_ context.Context, r Request, consume func(Response) error) error {
		calls.Add(1)
		if strings.Contains(string(r.Body), `"generate"`) || strings.Contains(string(r.Body), "previous_response_id") {
			t.Error("transport fields reached HTTP upstream")
		}
		body := "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_isolated\",\"object\":\"response\",\"status\":\"completed\",\"output\":[],\"usage\":{\"input_tokens\":1,\"output_tokens\":0,\"total_tokens\":1}}}\n\n"
		return consume(Response{StatusCode: http.StatusOK, ContentType: "text/event-stream", Body: strings.NewReader(body)})
	})})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := server.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	home, work := t.TempDir(), t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	provider := fmt.Sprintf(`model_providers.mock={name="Isolated mock",base_url="%s/v1",wire_api="responses",requires_openai_auth=false,env_key="OMNIPROXY_TEST_KEY",supports_websockets=true,request_max_retries=0,stream_max_retries=0}`, server.Status().Address)
	command := exec.CommandContext(ctx, executable, "exec", "--ignore-user-config", "--ignore-rules", "--ephemeral", "--skip-git-repo-check", "--sandbox", "read-only", "--json", "-C", work, "-c", `model_provider="mock"`, "-m", "gpt-test", "-c", provider, "Reply OK without tools.")
	command.Dir = work
	// 不继承 API Key、Agent 配置或登录环境；请求目标与 HTTP 代理均指向 loopback。
	for _, name := range []string{"PATH", "SystemRoot", "WINDIR", "COMSPEC", "PATHEXT"} {
		if value := os.Getenv(name); value != "" {
			command.Env = append(command.Env, name+"="+value)
		}
	}
	command.Env = append(command.Env, "CODEX_HOME="+home, "TEMP="+home, "TMP="+home, "OMNIPROXY_TEST_KEY=isolated-placeholder", "NO_PROXY=127.0.0.1,localhost", "HTTP_PROXY="+server.Status().Address, "HTTPS_PROXY="+server.Status().Address, "ALL_PROXY="+server.Status().Address)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("isolated CLI failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), `"type":"turn.completed"`) || calls.Load() != 1 {
		t.Fatalf("expected one real request after local warmup, calls=%d\n%s", calls.Load(), output)
	}
	select {
	case event := <-finished:
		if event.State != "success" || event.Transport != "websocket" {
			t.Fatal("unexpected CLI request result", event)
		}
	case <-ctx.Done():
		t.Fatal("CLI request was not recorded")
	}
	if len(finished) != 0 {
		t.Fatal("warmup was recorded as model usage")
	}
}
