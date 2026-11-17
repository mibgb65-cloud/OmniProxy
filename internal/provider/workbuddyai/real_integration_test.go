//go:build workbuddyreal

package workbuddyai

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

// loadRealCredential 读取 DPAPI 解出的受保护凭据副本；测试不写入、不打印令牌。
// cred.json 形如 {"accessToken":"…","refreshToken":"…","uid":"…","domain":"www.workbuddy.ai"}。
func loadRealCredential(t *testing.T, env string) Credentials {
	t.Helper()
	path := os.Getenv(env)
	if path == "" {
		t.Skipf("%s 未设置：跳过真实账号验证", env)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cred Credentials
	if err := json.Unmarshal(raw, &cred); err != nil {
		t.Fatal(err)
	}
	if !cred.Valid() {
		t.Fatal("凭据不完整：需要 accessToken 与 uid")
	}
	return cred
}

// realChat 用给定构建跑一次最小非 system 开头的对话，返回流式回答。
func realChat(t *testing.T, client *Client, credential Credentials, model string) {
	t.Helper()
	body := []byte(`{"model":"` + model + `","messages":[{"role":"user","content":"Reply with exactly one word: OK"}],"stream":true,"stream_options":{"include_usage":true},"max_tokens":32}`)
	request := proxy.Request{Protocol: ChatProtocol, Stream: true, Model: model, Body: body}
	var (
		status  int
		content strings.Builder
	)
	if err := client.Forward(context.Background(), credential, request, func(response proxy.Response) error {
		status = response.StatusCode
		payload, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			return readErr
		}
		for _, line := range strings.Split(string(payload), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				continue
			}
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
				Usage struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					TotalTokens      int `json:"total_tokens"`
				} `json:"usage"`
			}
			if json.Unmarshal([]byte(data), &chunk) != nil {
				continue
			}
			if len(chunk.Choices) > 0 {
				content.WriteString(chunk.Choices[0].Delta.Content)
			}
			if chunk.Usage.TotalTokens > 0 {
				t.Logf("usage: prompt=%d completion=%d total=%d", chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens, chunk.Usage.TotalTokens)
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("forward: %v", err)
	}
	if status != 200 {
		t.Fatalf("status = %d", status)
	}
	answer := strings.TrimSpace(content.String())
	t.Logf("answer: %q", answer)
	if answer == "" {
		t.Fatal("no streamed content")
	}
}

// TestRealChatAgainstWorkBuddy 用真实国际版账号跑一次最小对话，验证出站形态（补齐 system）、
// 流式回包与 SSE 解析。只在显式开启时运行：
//
//	$env:WORKBUDDY_REAL_CREDENTIAL = 'C:\path\to\cred.json'
//	go test -tags workbuddyreal ./internal/provider/workbuddyai -run TestRealChatAgainstWorkBuddy -v
//
// 会产生真实积分消耗。
func TestRealChatAgainstWorkBuddy(t *testing.T) {
	credential := loadRealCredential(t, "WORKBUDDY_REAL_CREDENTIAL")
	realChat(t, New(nil), credential, "glm-5.3-flash")
}

// TestRealChatAgainstWorkBuddyChina 用真实国内版账号验证国内构建的指纹、请求形态与流式回包；
// 国内版的 UA 与 platform 与国际版不同，被上游按 10085 拒绝时这里会直接失败。
//
//	$env:WORKBUDDY_REAL_CN_CREDENTIAL = 'C:\path\to\cred.json'
//	go test -tags workbuddyreal ./internal/provider/workbuddyai -run TestRealChatAgainstWorkBuddyChina -v
func TestRealChatAgainstWorkBuddyChina(t *testing.T) {
	credential := loadRealCredential(t, "WORKBUDDY_REAL_CN_CREDENTIAL")
	model := os.Getenv("WORKBUDDY_REAL_CN_MODEL")
	if model == "" {
		model = "glm-5.3-flash"
	}
	realChat(t, NewForBuild(China, nil), credential, model)
}
