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
	body, err := json.Marshal(map[string]any{
		"model": model, "stream": stream, "max_output_tokens": 16,
		"reasoning": map[string]string{"effort": "high"},
		"input":     []map[string]any{{"role": "user", "content": []map[string]string{{"type": "input_text", "text": "Reply with OK only."}}}},
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
	reply, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if err != nil {
		t.Fatal("cannot read gateway response")
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
