package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/credential"
	"omniproxy/internal/provider/river"
)

// TestLiveCodexRiverModel 用现有数据库与真实 River Key 验证 Codex 入口：
// 选择 River 公共模型时网关必须回到协议入口，把模型与档位还原后完成一次真实请求。
// 只读数据库、不写请求记录；未显式设置 OMNIPROXY_LIVE_RIVER=1 时跳过。
func TestLiveCodexRiverModel(t *testing.T) {
	prof, db := liveProfile(t, "OMNIPROXY_LIVE_RIVER")
	store, err := credential.New(prof)
	if err != nil {
		t.Fatal("cannot initialize protected credential access")
	}
	providers, accounts := provider.NewService(db), account.NewService(db, store)
	service := New(proxy.Options{}, providers, accounts, nil)
	service.SetRouter(routes.New(db, providers, accounts))
	if _, err := service.Start(); err != nil {
		t.Fatal("cannot start isolated test listener")
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if _, err := service.Stop(ctx); err != nil {
			t.Error("cannot stop isolated test listener")
		}
	})

	for _, stream := range []bool{false, true} {
		started := time.Now()
		status, body := codexEntryReply(t, service, river.PublicModel(river.DeepSeekFlash1017), stream)
		t.Logf("entry=/codex/v1/responses model=%s stream=%t status=%d latency_ms=%d body=%s",
			river.PublicModel(river.DeepSeekFlash1017), stream, status, time.Since(started).Milliseconds(), truncateForLog(body))
		if status != http.StatusOK {
			t.Fatalf("live codex request failed: status=%d body=%s", status, truncateForLog(body))
		}
		if !responsesReplyHasContent(body, stream) {
			t.Fatal("live codex request returned no complete reply")
		}
	}
}

// codexEntryReply 用 Codex 客户端的入口与正文形状发一条最短 Responses 请求。
func codexEntryReply(t *testing.T, service *Service, model string, stream bool) (int, []byte) {
	t.Helper()
	// 工具集与真实 Codex 一致：函数工具 + 一个 MCP 命名空间分组。命名空间由网关按
	// 目标服务商移除，函数工具原样保留；这样验证的就是真实客户端正文形状。
	// 历史里同时带一对合法工具调用／结果与一条没有 call_id 的桌面端 Companion Page
	// 快照：后者没有对应的工具调用，River 会先返回 200 再中断连接，必须由网关移除。
	body, err := json.Marshal(map[string]any{
		"model": model, "stream": stream, "max_output_tokens": 16,
		"reasoning": map[string]string{"effort": "high"},
		// Codex 0.169 起默认请求加密输出；第三方服务商无法表达，必须由网关移除。
		"include": []string{"reasoning.encrypted_content", "output.encrypted_content"},
		"input": []map[string]any{
			{"role": "user", "content": []map[string]string{{"type": "input_text", "text": "Reply with OK only."}}},
			{"type": "function_call", "call_id": "call_live_1", "name": "exec_command", "arguments": "{}"},
			{"type": "function_call_output", "call_id": "call_live_1", "output": "ok"},
			{"type": "function_call_output", "name": "codex_apps_open_page", "namespace": "agent", "output": `{"page_id":null}`},
		},
		"tools": []any{
			map[string]any{"type": "function", "name": "exec_command", "parameters": map[string]any{"type": "object", "properties": map[string]any{}}},
			map[string]any{"type": "namespace", "name": "mcp__codex_apps__pets", "tools": []any{
				map[string]any{"type": "function", "name": "_adopt", "parameters": map[string]any{"type": "object", "properties": map[string]any{}}},
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, service.Info().Runtime.Address+"/codex/v1/responses", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("originator", "codex_desktop")
	client := &http.Client{
		Timeout: 45 * time.Second, Transport: &http.Transport{Proxy: nil},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	defer client.CloseIdleConnections()
	response, err := client.Do(req)
	if err != nil {
		t.Fatal("local codex request failed")
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error("cannot close gateway response")
		}
	}()
	reply, readErr := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if readErr != nil {
		// 流式失败时网关会中断连接，这里保留已收到的字节供断言与排查。
		t.Logf("gateway response stopped early: %v after %d bytes", readErr, len(reply))
	}
	return response.StatusCode, reply
}

// responsesReplyHasContent 只确认 Responses 回复里出现了文本内容或终结事件。
func responsesReplyHasContent(body []byte, stream bool) bool {
	if !stream {
		var reply struct {
			Output []struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"output"`
		}
		if json.Unmarshal(body, &reply) != nil {
			return false
		}
		for _, item := range reply.Output {
			for _, part := range item.Content {
				if strings.TrimSpace(part.Text) != "" {
					return true
				}
			}
		}
		return false
	}
	return strings.Contains(string(body), "response.completed") || strings.Contains(string(body), "response.incomplete")
}

func truncateForLog(body []byte) string {
	const limit = 400
	if len(body) <= limit {
		return string(body)
	}
	return string(body[:limit]) + "…"
}
