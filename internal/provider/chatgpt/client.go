// Package chatgpt 转发隔离订阅账号的 Responses 请求，不接触用户默认登录。
package chatgpt

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"omniproxy/internal/core/proxy"
	"time"
)

type Client struct{ http *http.Client }

func New(transport http.RoundTripper) *Client {
	if transport == nil {
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.ResponseHeaderTimeout = 45 * time.Second
		t.TLSHandshakeTimeout = 10 * time.Second
		transport = t
	}
	return &Client{&http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

// Forward 只访问固定官方 Codex Responses 端点；本地鉴权头不透传。
func (c *Client) Forward(ctx context.Context, key []byte, accountID string, request proxy.Request, consume func(proxy.Response) error) error {
	var body map[string]json.RawMessage
	if json.Unmarshal(request.Body, &body) != nil {
		return proxy.Failure("invalid_request", 400)
	}
	body["store"] = json.RawMessage("false")
	body["stream"] = json.RawMessage("true")
	if _, ok := body["instructions"]; !ok {
		body["instructions"] = json.RawMessage(`""`)
	}
	// 订阅端点没有公开 API 的持久化与输出预算参数，工具与上下文原样保留。
	delete(body, "max_output_tokens")
	delete(body, "temperature")
	delete(body, "top_p")
	raw, err := json.Marshal(body)
	if err != nil {
		return proxy.Failure("invalid_request", 400)
	}
	defer clear(raw)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://chatgpt.com/backend-api/codex/responses", bytes.NewReader(raw))
	if err != nil {
		return proxy.Failure("upstream_unavailable", 502)
	}
	req.Header.Set("Authorization", "Bearer "+string(key))
	req.Header.Set("ChatGPT-Account-ID", accountID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("OpenAI-Beta", "responses=experimental")
	req.Header.Set("Originator", "codex_cli_rs")
	defer req.Header.Del("Authorization")
	defer req.Header.Del("ChatGPT-Account-ID")
	resp, err := c.http.Do(req)
	if err != nil {
		return proxy.Failure("upstream_unavailable", 502)
	}
	defer resp.Body.Close()
	return consume(proxy.Response{StatusCode: resp.StatusCode, ContentType: resp.Header.Get("Content-Type"), RetryAfter: resp.Header.Get("Retry-After"), Body: resp.Body})
}
