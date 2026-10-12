package gateway

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// 通过本机 HTTP 接口验证实际转发；固定短消息只用于测试，不暴露为产品功能。
func localGatewayRequest(t *testing.T, service *Service, model string, stream bool) (int, []byte) {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"model": model, "stream": stream, "max_tokens": 16,
		"thinking": map[string]string{"type": "disabled"},
		"messages": []map[string]string{{"role": "user", "content": "Reply with OK only."}},
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, service.Info().Runtime.Address+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: 45 * time.Second, Transport: &http.Transport{Proxy: nil},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	defer client.CloseIdleConnections()
	response, err := client.Do(req)
	if err != nil {
		t.Fatal("local gateway request failed")
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

func gatewayResponseError(t *testing.T, body []byte) string {
	t.Helper()
	var failure struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &failure); err != nil {
		t.Fatal("invalid gateway error response")
	}
	return failure.Error.Code
}

func gatewayReplyHasContent(body []byte, stream bool) bool {
	var reply struct {
		Choices []struct {
			Message struct{ Content string } `json:"message"`
			Delta   struct{ Content string } `json:"delta"`
		} `json:"choices"`
	}
	if !stream {
		return json.Unmarshal(body, &reply) == nil && len(reply.Choices) > 0 && strings.TrimSpace(reply.Choices[0].Message.Content) != ""
	}
	content, done := false, false
	for _, event := range strings.Split(string(body), "\n\n") {
		if !strings.HasPrefix(event, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(event, "data:"))
		if data == "[DONE]" {
			done = strings.Contains(string(body), "data: [DONE]\n\n")
			break
		}
		if json.Unmarshal([]byte(data), &reply) != nil {
			return false
		}
		for _, choice := range reply.Choices {
			content = content || choice.Delta.Content != ""
		}
	}
	return content && done
}
