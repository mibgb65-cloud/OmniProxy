package river

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

// Client 在凭证回调内完成请求，禁止重定向和客户端头部透传。
type Client struct{ http *http.Client }

// New 构造隔离传输；可注入 transport 进行无真实凭证的契约测试。
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

// Forward 保留原生协议和流式正文，响应关闭后才释放凭证。
func (c *Client) Forward(ctx context.Context, baseURL string, key []byte, request proxy.Request, consume func(proxy.Response) error) (result error) {
	path, ok := protocolPath(request.Protocol)
	if !ok {
		return proxy.Failure("unsupported_protocol", 400)
	}
	endpoint, err := requestURL(baseURL, path)
	if err != nil {
		return err
	}
	if !validKey(key) || consume == nil {
		return proxy.Failure("invalid_credential", 503)
	}
	body, err := validatedBody(request, path)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
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

func protocolPath(protocol string) (string, bool) {
	switch protocol {
	case "chat":
		return "/chat/completions", true
	case "responses":
		return "/responses", true
	default:
		return "", false
	}
}

func requestURL(baseURL, path string) (string, error) {
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
