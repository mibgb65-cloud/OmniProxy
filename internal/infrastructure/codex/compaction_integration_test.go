package codex

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/provider/chatgpt"
)

type compactQATransport func(*http.Request) (*http.Response, error)

func (f compactQATransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type compactQAUpstream struct{ client *chatgpt.Client }

func (s compactQAUpstream) Forward(ctx context.Context, r proxy.Request, c func(proxy.Response) error) error {
	return s.client.Forward(ctx, []byte("qa-upstream"), "qa-account", r, c)
}

// 用真实本机核心、真实代理和固定假上游验证 V2；不继承用户登录或把请求发送到外网。
func TestInstalledCodexRemoteCompactionV2ThroughProxy(t *testing.T) {
	executable := os.Getenv("OMNIPROXY_CODEX_TEST_EXECUTABLE")
	if os.Getenv("OMNIPROXY_CODEX_COMPACT_QA") != "1" || executable == "" {
		t.Skip("显式启用本机离线压缩契约验收")
	}
	if !filepath.IsAbs(executable) {
		t.Fatal("executable must be absolute")
	}
	home, work := t.TempDir(), t.TempDir()
	var compactCalls atomic.Int32
	adapter := chatgpt.New(compactQATransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://chatgpt.com/backend-api/codex/responses" || r.Header.Get("Authorization") != "Bearer qa-upstream" {
			t.Error("native compaction escaped the fixed subscription adapter")
		}
		var metadata struct {
			Compaction struct {
				Implementation string `json:"implementation"`
			} `json:"compaction"`
		}
		json.Unmarshal([]byte(r.Header.Get("X-Codex-Turn-Metadata")), &metadata)
		compact := metadata.Compaction.Implementation == "responses_compaction_v2" && r.Header.Get("X-Codex-Beta-Features") == "remote_compaction_v2"
		item := map[string]any{"type": "message", "id": "msg_qa", "role": "assistant", "status": "completed", "content": []any{map[string]any{"type": "output_text", "text": "QA done"}}}
		if compact {
			compactCalls.Add(1)
			item = map[string]any{"type": "compaction", "id": "cmp_qa", "encrypted_content": "opaque-qa"}
		}
		var data strings.Builder
		for _, event := range []map[string]any{
			{"type": "response.created", "response": map[string]any{"id": "resp_qa", "status": "in_progress", "output": []any{}}},
			{"type": "response.output_item.done", "output_index": 0, "item": item},
			{"type": "response.completed", "response": map[string]any{"id": "resp_qa", "status": "completed", "output": []any{item}, "usage": map[string]int{"input_tokens": 10, "output_tokens": 1, "total_tokens": 11}}},
		} {
			raw, _ := json.Marshal(event)
			data.WriteString("data: " + string(raw) + "\n\n")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(data.String()))}, nil
	}))
	server := proxy.New(proxy.Options{Port: 0, Models: []string{"gpt-qa"}, Authorize: func(_ context.Context, key string) (string, bool) { return "", key == "isolated-placeholder" }, Upstream: compactQAUpstream{adapter}})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop(context.Background())
	models, err := catalog.CodexJSON([]catalog.Entry{{ID: "gpt-qa", Name: "QA", ProviderKind: "openai"}})
	if err != nil {
		t.Fatal(err)
	}
	catalogPath := filepath.Join(home, "models.json")
	if os.WriteFile(catalogPath, models, 0600) != nil {
		t.Fatal("catalog fixture unavailable")
	}
	config := "model='gpt-qa'\nmodel_provider='openai'\nopenai_base_url=" + strconv.Quote(server.Status().Address+"/v1") + "\nmodel_catalog_json=" + strconv.Quote(filepath.ToSlash(catalogPath)) + "\n[features]\napi_key_model_discovery=false\nresponses_websockets=false\nresponses_websockets_v2=false\nenable_request_compression=false\n"
	if os.WriteFile(filepath.Join(home, "config.toml"), []byte(config), 0600) != nil {
		t.Fatal("config fixture unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "app-server", "--listen", "stdio://", "-c", `cli_auth_credentials_store="file"`)
	hideWindow(command)
	command.Dir = work
	for _, name := range []string{"PATH", "SystemRoot", "WINDIR", "COMSPEC", "PATHEXT"} {
		if value := os.Getenv(name); value != "" {
			command.Env = append(command.Env, name+"="+value)
		}
	}
	command.Env = append(command.Env, "CODEX_HOME="+home, "USERPROFILE="+home, "HOME="+home, "APPDATA="+home, "LOCALAPPDATA="+home, "TEMP="+home, "TMP="+home, "NO_PROXY=127.0.0.1,localhost", "HTTP_PROXY="+server.Status().Address, "HTTPS_PROXY="+server.Status().Address, "ALL_PROXY="+server.Status().Address)
	command.Stderr = io.Discard
	input, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = command.Start(); err != nil {
		t.Fatal(err)
	}
	client := &rpcClient{cmd: command, input: input, pending: map[uint64]chan wireResponse{}, notices: make(chan notification, 32), done: make(chan struct{})}
	go client.read(output)
	defer client.Close()
	var ignored struct{}
	if client.call(ctx, "initialize", map[string]any{"clientInfo": map[string]string{"name": "omniproxy_compact_qa", "version": "0.1.0"}, "capabilities": map[string]bool{"experimentalApi": true}}, &ignored) != nil || client.write(map[string]any{"method": "initialized"}) != nil {
		t.Fatal("native core initialization failed")
	}
	if client.call(ctx, "account/login/start", map[string]string{"type": "apiKey", "apiKey": "isolated-placeholder"}, &ignored) != nil {
		t.Fatal("isolated mock auth failed")
	}
	var started struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if client.call(ctx, "thread/start", map[string]any{"cwd": work, "approvalPolicy": "never", "sandbox": "read-only"}, &started) != nil || started.Thread.ID == "" {
		t.Fatal("isolated thread failed")
	}
	if client.call(ctx, "turn/start", map[string]any{"threadId": started.Thread.ID, "input": []any{map[string]string{"type": "text", "text": "Offline QA only"}}}, &ignored) != nil {
		t.Fatal("isolated turn failed")
	}
	waitQAThread(ctx, t, client, started.Thread.ID, false)
	if err = client.call(ctx, "thread/compact/start", map[string]string{"threadId": started.Thread.ID}, &ignored); err != nil {
		t.Fatal("native compaction request rejected")
	}
	waitQAThread(ctx, t, client, started.Thread.ID, true)
	if compactCalls.Load() != 1 {
		t.Fatal("native V2 compaction was not forwarded exactly once")
	}
	t.Log("native Responses V2 compaction and completed contextCompaction item verified through real proxy")
}

func waitQAThread(ctx context.Context, t *testing.T, client *rpcClient, id string, compact bool) {
	t.Helper()
	for range 1000 {
		var result struct {
			Thread struct {
				Turns []struct {
					Status string `json:"status"`
					Items  []struct {
						Type string `json:"type"`
					} `json:"items"`
				} `json:"turns"`
			} `json:"thread"`
		}
		if err := client.call(ctx, "thread/read", map[string]any{"threadId": id, "includeTurns": true}, &result); err != nil {
			// 新线程的 rollout 尚未物化时读取可能暂不可用；有界等待，不能当作压缩已经成功。
			if err.Error() != "subscription: request_failed" || ctx.Err() != nil {
				t.Fatal("QA thread state unavailable")
			}
		}
		if turns := result.Thread.Turns; len(turns) > 0 {
			last := turns[len(turns)-1]
			if last.Status == "failed" {
				t.Fatal("isolated native compaction turn failed")
			}
			if last.Status == "completed" {
				if !compact {
					return
				}
				for _, item := range last.Items {
					if item.Type == "contextCompaction" {
						return
					}
				}
			}
		}
		select {
		case <-ctx.Done():
			t.Fatal("QA thread timed out")
		case <-time.After(20 * time.Millisecond):
		}
	}
	t.Fatal("QA thread did not reach expected completed state")
}
