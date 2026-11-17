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

// TestRealChatAgainstWorkBuddy 用真实账号跑一次最小对话，验证出站形态（补齐 system）、
// 流式回包与 SSE 解析。只在显式开启时运行：
//
//	$env:WORKBUDDY_REAL_CREDENTIAL = 'C:\path\to\cred.json'
//	go test -tags workbuddyreal ./internal/provider/workbuddyai -run TestRealChatAgainstWorkBuddy -v
//
// cred.json 形如 {"accessToken":"…","refreshToken":"…","uid":"…","domain":"www.workbuddy.ai"}，
// 由 DPAPI 解出的受保护凭据转换而来；测试不写入、不打印令牌。会产生真实积分消耗。
func TestRealChatAgainstWorkBuddy(t *testing.T) {
	path := os.Getenv("WORKBUDDY_REAL_CREDENTIAL")
	if path == "" {
		t.Skip("WORKBUDDY_REAL_CREDENTIAL 未设置：跳过真实账号验证")
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
	body := []byte(`{"model":"glm-5.3-flash","messages":[{"role":"user","content":"Reply with exactly one word: OK"}],"stream":true,"stream_options":{"include_usage":true},"max_tokens":32}`)
	request := proxy.Request{Protocol: ChatProtocol, Stream: true, Model: "glm-5.3-flash", Body: body}
	client := New(nil)
	var status int
	var content strings.Builder
	if err := client.Forward(context.Background(), cred, request, func(response proxy.Response) error {
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
