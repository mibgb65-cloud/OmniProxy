package oxionai

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"omniproxy/internal/core/netproxy"
	"omniproxy/internal/core/proxy"
)

// Client 只转发 Chat Completions 正文，禁止携带本地客户端头部或自动跟随重定向。
type Client struct{ http *http.Client }

// New 构造隔离传输；自定义 transport 用于提供商契约测试。
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

// Forward 在受控密钥回调内发送 Chat Completions 请求；响应关闭后才释放凭证。
// Responses／Messages 由网关先转换，本适配器不伪造原生协议。
func (c *Client) Forward(ctx context.Context, baseURL string, key []byte, request proxy.Request, consume func(proxy.Response) error) (result error) {
	if request.Protocol != "" && request.Protocol != "chat" {
		return proxy.Failure("unsupported_protocol", 400)
	}
	endpoint, err := completionsURL(baseURL)
	if err != nil {
		return err
	}
	if !validKey(key) || consume == nil {
		return proxy.Failure("invalid_credential", 503)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(request.Body))
	if err != nil {
		return proxy.UpstreamFailure(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+string(key))
	defer req.Header.Del("Authorization")
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

func validKey(key []byte) bool {
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

// endpointURL 校验基址后拼接固定路径；只允许 HTTPS 或本机回环 HTTP。
func endpointURL(baseURL, path string) (string, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	u, err := url.Parse(baseURL)
	if err != nil || len(baseURL) > 2048 || u.Hostname() == "" || u.Opaque != "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.ForceQuery {
		return "", proxy.Failure("invalid_configuration", 503)
	}
	if u.Scheme != "https" {
		ip := net.ParseIP(u.Hostname())
		if u.Scheme != "http" || !(strings.EqualFold(u.Hostname(), "localhost") || ip != nil && ip.IsLoopback()) {
			return "", proxy.Failure("invalid_configuration", 503)
		}
	}
	if port := u.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return "", proxy.Failure("invalid_configuration", 503)
		}
	}
	u.Path = strings.TrimRight(u.Path, "/") + path
	u.RawPath = ""
	return u.String(), nil
}

func completionsURL(baseURL string) (string, error) { return endpointURL(baseURL, "/chat/completions") }
