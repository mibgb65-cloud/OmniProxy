package dumate

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"omniproxy/internal/core/netproxy"
	"omniproxy/internal/core/proxy"
)

const (
	// chatPath 相对网关版本基址追加，不能重复 /gateway/apis/v1。
	chatPath = "/chat/completions"
	// ChatProtocol 是上游唯一可确认的原生协议；其余协议由网关转换。
	ChatProtocol = "chat"
)

// Client 只向固定网关转发 Chat Completions，禁止重定向与客户端头部透传。
type Client struct {
	http *http.Client
	base string
}

// New 构造隔离传输；可注入 transport 进行无真实凭据的契约测试。
func New(transport http.RoundTripper) *Client {
	if transport == nil {
		standard := http.DefaultTransport.(*http.Transport).Clone()
		standard.ResponseHeaderTimeout = GatewayResponseHeaderTimeout
		standard.TLSHandshakeTimeout = TLShandshakeTimeout
		standard.MaxIdleConnsPerHost = 16
		transport = netproxy.Wrap(standard)
	}
	return &Client{http: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, base: DefaultBaseURL}
}

// NewClientAt 用指定网关基址构造客户端；非 HTTPS 基址只允许回环地址，
// 供隔离集成测试使用，生产路径始终使用固定网关。
func NewClientAt(base string, transport http.RoundTripper) (*Client, error) {
	if _, _, err := consoleURL(base, chatPath); err != nil {
		return nil, err
	}
	client := New(transport)
	client.base = strings.TrimRight(base, "/")
	return client, nil
}

// Forward 在凭证回调内完成一次 Chat 请求，响应由 consume 关闭。
func (c *Client) Forward(ctx context.Context, token string, request proxy.Request, consume func(proxy.Response) error) (result error) {
	if request.Protocol != ChatProtocol {
		return proxy.Failure("unsupported_protocol", 400)
	}
	if len(token) == 0 || len(token) > 8192 || !printable(token) || consume == nil {
		return proxy.Failure("invalid_credential", 503)
	}
	body, err := requestBody(request)
	if err != nil {
		return err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+chatPath, bytes.NewReader(body))
	if err != nil {
		return proxy.UpstreamFailure(err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("User-Agent", UserAgent)
	if request.Stream {
		httpRequest.Header.Set("Accept", "text/event-stream")
	}
	httpRequest.Header.Set("Authorization", "Bearer "+token)
	defer httpRequest.Header.Del("Authorization")
	response, err := c.http.Do(httpRequest)
	if err != nil {
		return proxy.UpstreamFailure(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil && result == nil {
			result = proxy.Failure("upstream_unavailable", 502)
		}
	}()
	if response.StatusCode == http.StatusUnauthorized {
		// 只说明 gateway token 失效；Cookie 是否失效由控制台判定。
		return proxy.Failure(CodeReauthRequired, http.StatusUnauthorized)
	}
	return consume(proxy.Response{StatusCode: response.StatusCode, ContentType: response.Header.Get("Content-Type"), RetryAfter: response.Header.Get("Retry-After"), Body: response.Body})
}

// requestBody 为流式请求补上标准 include_usage，保留客户端其它字段。
func requestBody(request proxy.Request) ([]byte, error) {
	if len(request.Body) == 0 || len(request.Body) > proxy.MaxRequestBytes {
		return nil, proxy.Failure("invalid_request", 400)
	}
	if !request.Stream {
		return request.Body, nil
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(request.Body, &fields) != nil || fields == nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	options := map[string]json.RawMessage{}
	if raw, exists := fields["stream_options"]; exists && string(raw) != "null" {
		if json.Unmarshal(raw, &options) != nil || options == nil {
			return nil, proxy.Failure("invalid_request", 400)
		}
	}
	options["include_usage"] = json.RawMessage("true")
	fields["stream_options"], _ = json.Marshal(options)
	body, err := json.Marshal(fields)
	if err != nil || len(body) > proxy.MaxRequestBytes {
		return nil, proxy.Failure("request_too_large", 413)
	}
	return body, nil
}

// TokenCacheKey 由账号与凭据版本共同决定，重新导入后不会命中旧 token。
func TokenCacheKey(accountID, credentialVersion string) string {
	return strings.TrimSpace(accountID) + ":" + strings.TrimSpace(credentialVersion)
}
