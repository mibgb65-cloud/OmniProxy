package minimax

import (
	"bytes"
	"context"
	"net/http"
	"omniproxy/internal/core/proxy"
	"strings"
	"time"
)

// Client 使用独立传输，在凭证回调内消费原生 HTTP／SSE。
type Client struct{ http *http.Client }

// New 禁止重定向及客户端头部透传，可注入传输验证官方端点。
func New(transport http.RoundTripper) *Client {
	if transport == nil {
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.ResponseHeaderTimeout = 45 * time.Second
		t.TLSHandshakeTimeout = 10 * time.Second
		t.MaxIdleConnsPerHost = 16
		transport = t
	}
	return &Client{http: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

// ValidAPIKey 限制认证头格式，错误密钥不会进入 HTTP 头部。
func ValidAPIKey(key []byte) bool {
	if len(key) == 0 || len(key) > 8192 {
		return false
	}
	for _, ch := range key {
		if ch < 33 || ch > 126 {
			return false
		}
	}
	return true
}

func endpoint(o Options, protocol string) (string, error) {
	base, err := o.BaseURL()
	if err != nil {
		return "", proxy.Failure("invalid_configuration", 503)
	}
	switch protocol {
	case "chat":
		return base + "/chat/completions", nil
	case "responses":
		return base + "/responses", nil
	case "messages":
		return strings.TrimSuffix(base, "/v1") + "/anthropic/v1/messages", nil
	default:
		return "", proxy.Failure("unsupported_protocol", 400)
	}
}

// Forward 原生透传协议、工具关联、推理历史和增量事件，不收集完整 SSE 后伪造流。
func (c *Client) Forward(ctx context.Context, o Options, key []byte, request proxy.Request, consume func(proxy.Response) error) (result error) {
	address, err := endpoint(o, request.Protocol)
	if err != nil {
		return err
	}
	if !ValidKeyForOptions(o, key) || consume == nil {
		return proxy.Failure("invalid_credential", 503)
	}
	body, err := validatedBody(o, request)
	if err != nil {
		return err
	}
	defer clear(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, address, bytes.NewReader(body))
	if err != nil {
		return proxy.Failure("upstream_unavailable", 502)
	}
	req.Header.Set("Authorization", "Bearer "+string(key))
	req.Header.Set("Content-Type", "application/json")
	defer req.Header.Del("Authorization")
	if request.Stream {
		req.Header.Set("Accept", "text/event-stream")
	}
	response, err := c.http.Do(req)
	if err != nil {
		return proxy.Failure("upstream_unavailable", 502)
	}
	defer func() {
		if err := response.Body.Close(); err != nil && result == nil {
			result = proxy.Failure("upstream_unavailable", 502)
		}
	}()
	return consume(proxy.Response{StatusCode: response.StatusCode, ContentType: response.Header.Get("Content-Type"), RetryAfter: response.Header.Get("Retry-After"), Body: response.Body})
}
