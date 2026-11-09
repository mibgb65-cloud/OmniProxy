// Package openai 转发 OpenAI 按量 API Key 账号的原生 Responses 请求。
package openai

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

// DefaultBaseURL 是 OpenAI 官方 API 根地址；账号可使用已保存的提供商地址。
const DefaultBaseURL = "https://api.openai.com/v1"

// Client 在凭证回调内完成请求，禁止重定向和客户端头部透传。
type Client struct{ http *http.Client }

// New 构造隔离传输；可注入 transport 进行无真实凭证的契约测试。
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

// Forward 保留原生 Responses 正文与流式响应；压缩请求走独立无状态端点。
func (c *Client) Forward(ctx context.Context, baseURL string, key []byte, request proxy.Request, consume func(proxy.Response) error) (result error) {
	if request.Protocol != "responses" {
		return proxy.Failure("unsupported_protocol", 400)
	}
	if len(key) == 0 || len(key) > 8192 || consume == nil {
		return proxy.Failure("invalid_credential", 503)
	}
	for _, char := range key {
		if char < 33 || char > 126 {
			return proxy.Failure("invalid_credential", 503)
		}
	}
	path := "/responses"
	if request.Compaction {
		path = "/responses/compact"
	}
	endpoint, err := endpointURL(baseURL, path)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(request.Body))
	if err != nil {
		return proxy.UpstreamFailure(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+string(key))
	defer req.Header.Del("Authorization")
	if request.Compaction || !request.Stream {
		req.Header.Set("Accept", "application/json")
	} else {
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
	return consume(proxy.Response{
		StatusCode:  response.StatusCode,
		ContentType: response.Header.Get("Content-Type"),
		RetryAfter:  response.Header.Get("Retry-After"),
		Body:        response.Body,
	})
}

// endpointURL 只接受 HTTPS 或本机回环 HTTP，并把固定端点拼到已保存基址后。
func endpointURL(baseURL, path string) (string, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if len(baseURL) > 2048 {
		return "", proxy.Failure("invalid_configuration", 503)
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Hostname() == "" || u.Opaque != "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.ForceQuery {
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
