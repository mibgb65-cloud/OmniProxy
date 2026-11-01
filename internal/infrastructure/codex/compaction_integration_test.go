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
	for _, automatic := range []bool{false, true} {
		name := "manual"
		if automatic {
			name = "automatic_from_catalog_capacity"
		}
		t.Run(name, func(t *testing.T) { installedCodexCompaction(t, automatic) })
	}
}

func installedCodexCompaction(t *testing.T, automatic bool) {
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
		if r.Method == http.MethodGet && r.URL.Path == "/backend-api/codex/models" {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"models":[{"slug":"gpt-qa","context_window":100000,"max_context_window":1000000}]}`))}, nil
		}
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
		inputTokens := 10
		if automatic && !compact && compactCalls.Load() == 0 {
			// 只有合成 usage 越过 90% 默认阈值，不生成或发送真实大上下文。
			inputTokens = 90001
		}
		var data strings.Builder
		for _, event := range []map[string]any{
			{"type": "response.created", "response": map[string]any{"id": "resp_qa", "status": "in_progress", "output": []any{}}},
			{"type": "response.output_item.done", "output_index": 0, "item": item},
			{"type": "response.completed", "response": map[string]any{"id": "resp_qa", "status": "completed", "output": []any{item}, "usage": map[string]int{"input_tokens": inputTokens, "output_tokens": 1, "total_tokens": inputTokens + 1}}},
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
	windows, err := adapter.ModelContextWindows(context.Background(), []byte("qa-upstream"), "qa-account", "0.163.0")
	if err != nil {
		t.Fatal("native model capacity fixture failed")
	}
	window := windows["gpt-qa"]
	models, err := catalog.CodexJSON([]catalog.Entry{{ID: "gpt-qa", Name: "QA", ProviderKind: "openai", Metadata: catalog.Metadata{ContextWindow: &window}}})
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
	method, params := "thread/compact/start", map[string]any{"threadId": started.Thread.ID}
	if automatic {
		method = "turn/start"
		params["input"] = []any{map[string]string{"type": "text", "text": "Continue offline QA"}}
	}
	if err = client.call(ctx, method, params, &ignored); err != nil {
		t.Fatal("native compaction request rejected")
	}
	waitQAThread(ctx, t, client, started.Thread.ID, true)
	if compactCalls.Load() != 1 {
		t.Fatal("native V2 compaction was not forwarded exactly once")
	}
	t.Log("native Responses V2 compaction and completed contextCompaction item verified through real proxy")
}
