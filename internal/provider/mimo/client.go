package mimo

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"omniproxy/internal/core/netproxy"
	"omniproxy/internal/core/proxy"
)

// Client 不保存密钥，原生响应在受保护凭证回调内消费。
type Client struct{ http *http.Client }

// New 创建禁止重定向的独立 HTTP 传输；测试可注入上游。
func New(transport http.RoundTripper) *Client {
	if transport == nil {
		standard := http.DefaultTransport.(*http.Transport).Clone()
		standard.ResponseHeaderTimeout = 45 * time.Second
		standard.TLSHandshakeTimeout = 10 * time.Second
		standard.MaxIdleConnsPerHost = 16
		transport = netproxy.Wrap(standard)
	}
	return &Client{http: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

// Forward 不转换三种原生协议，保留思考块、工具调用和增量事件。
func (c *Client) Forward(ctx context.Context, base string, key []byte, request proxy.Request, consume func(proxy.Response) error) (result error) {
	path, ok := protocolPath(request.Protocol)
	if !ok {
		return proxy.Failure("unsupported_protocol", 400)
	}
	address, err := endpoint(base, path)
	if err != nil {
		return err
	}
	if !keyMatchesBase(address, key) || consume == nil {
		return proxy.Failure("invalid_credential", 503)
	}
	if planBase(address) && !SupportsAccountModel(Options{BillingMode: "token_plan", Region: "cn"}, request.Model) {
		return proxy.Failure("unsupported_model", 400)
	}
	body, err := validatedBody(request)
	if err != nil {
		return err
	}
	defer clear(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, address, bytes.NewReader(body))
	if err != nil {
		return proxy.UpstreamFailure(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", string(key))
	defer req.Header.Del("Api-Key")
	if request.Stream {
		req.Header.Set("Accept", "text/event-stream")
	}
	response, err := c.http.Do(req)
	if err != nil {
		return proxy.UpstreamFailure(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil && result == nil {
			result = proxy.Failure("upstream_unavailable", 502)
		}
	}()
	return consume(proxy.Response{StatusCode: response.StatusCode, ContentType: response.Header.Get("Content-Type"), RetryAfter: response.Header.Get("Retry-After"), Body: response.Body})
}

func protocolPath(protocol string) (string, bool) {
	switch protocol {
	case "chat":
		return "/v1/chat/completions", true
	case "responses":
		return "/v1/responses", true
	case "messages":
		return "/anthropic/v1/messages", true
	default:
		return "", false
	}
}
