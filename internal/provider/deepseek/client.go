// Package deepseek 实现 DeepSeek Chat Completions 的有界 HTTP 适配器。
package deepseek

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"omniproxy/internal/core/proxy"
)

// DefaultBaseURL 是官方 HTTPS 端点，不包含具体协议路径。
const DefaultBaseURL = "https://api.deepseek.com"

// Models 返回本适配器支持的官方模型 ID，不代表账号权限或实时模型发现。
func Models() []string { return []string{"deepseek-flash", "deepseek-v4-pro"} }

// Client 仅转发协议正文，禁止携带本地客户端的头部或自动跟随重定向。
type Client struct{ http *http.Client }

// New 构造隔离传输；自定义 transport 用于提供商契约测试。
func New(transport http.RoundTripper) *Client {
	if transport == nil {
		standard := http.DefaultTransport.(*http.Transport).Clone()
		standard.ResponseHeaderTimeout = 45 * time.Second
		standard.TLSHandshakeTimeout = 10 * time.Second
		standard.MaxIdleConnsPerHost = 16
		transport = standard
	}
	return &Client{http: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

// Forward 在受控密钥回调内发送请求；响应关闭后才允许凭证释放。
func (c *Client) Forward(ctx context.Context, baseURL string, key []byte, request proxy.Request, consume func(proxy.Response) error) (result error) {
	endpoint, err := completionURL(baseURL)
	if err != nil {
		return err
	}
	if len(key) == 0 || len(key) > 8192 || consume == nil {
		return proxy.Failure("invalid_credential", 503)
	}
	for _, char := range key {
		if char < 33 || char > 126 {
			return proxy.Failure("invalid_credential", 503)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(request.Body))
	if err != nil {
		return proxy.Failure("upstream_unavailable", 502)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+string(key))
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

func completionURL(baseURL string) (string, error) {
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
	u.Path = strings.TrimRight(u.Path, "/") + "/chat/completions"
	u.RawPath = ""
	return u.String(), nil
}
