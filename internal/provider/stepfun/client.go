package stepfun

import (
	"bytes"
	"context"
	"net/http"
	"omniproxy/internal/core/proxy"
	"time"
)

// Client 不保存密钥，原生 HTTP／SSE 在凭证回调内消费。
type Client struct{ http *http.Client }

// New 使用独立传输并禁止重定向；测试可注入传输检查四类官方地址。
func New(transport http.RoundTripper) *Client {
	if transport == nil {
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.ResponseHeaderTimeout, t.TLSHandshakeTimeout = 45*time.Second, 10*time.Second
		t.MaxIdleConnsPerHost = 16
		transport = t
	}
	return &Client{http: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

// Forward 仅改写已验证的请求参数，不收集 SSE，不丢失工具及思考上下文。
func (c *Client) Forward(ctx context.Context, o Options, key []byte, request proxy.Request, consume func(proxy.Response) error) (result error) {
	base, err := o.BaseURL()
	if err != nil {
		return proxy.Failure("invalid_configuration", 503)
	}
	if !validKey(key) || consume == nil {
		return proxy.Failure("invalid_credential", 503)
	}
	path := "/" + request.Protocol
	if request.Protocol == "chat" {
		path = "/chat/completions"
	}
	body, err := validatedBody(o, request)
	if err != nil {
		return err
	}
	defer clear(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, bytes.NewReader(body))
	if err != nil {
		return proxy.Failure("upstream_unavailable", 502)
	}
	req.Header.Set("Authorization", "Bearer "+string(key))
	req.Header.Set("Content-Type", "application/json")
	defer req.Header.Del("Authorization")
	if request.Protocol == "messages" {
		req.Header.Set("Anthropic-Version", "2023-06-01")
	}
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
	if err := resourceFailure(response.StatusCode, response.Body); err != nil {
		return err
	}
	return consume(proxy.Response{StatusCode: response.StatusCode, ContentType: response.Header.Get("Content-Type"), RetryAfter: response.Header.Get("Retry-After"), Body: response.Body})
}

// ValidAPIKey 校验认证头允许的 ASCII 字节和长度，拒绝空格及控制字符。
func ValidAPIKey(key []byte) bool {
	if len(key) == 0 || len(key) > 8192 {
		return false
	}
	for _, char := range key {
		if char < 33 || char > 126 {
			return false
		}
	}
	return true
}

func validKey(key []byte) bool { return ValidAPIKey(key) }
