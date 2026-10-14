// Package google 接入 Gemini 的官方模型发现和 Chat 兼容接口。
package google

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"omniproxy/internal/core/proxy"
)

const DefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

type Client struct{ http *http.Client }

func New(transport http.RoundTripper) *Client {
	if transport == nil {
		standard := http.DefaultTransport.(*http.Transport).Clone()
		standard.ResponseHeaderTimeout = 45 * time.Second
		standard.TLSHandshakeTimeout = 10 * time.Second
		transport = standard
	}
	return &Client{http: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func endpoint(base, suffix string) (string, error) {
	if base == "" {
		base = DefaultBaseURL
	}
	u, err := url.Parse(base)
	if err != nil || len(base) > 2048 || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || u.ForceQuery {
		return "", proxy.Failure("invalid_configuration", 503)
	}
	ip := net.ParseIP(u.Hostname())
	if port := u.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return "", proxy.Failure("invalid_configuration", 503)
		}
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && (strings.EqualFold(u.Hostname(), "localhost") || ip != nil && ip.IsLoopback())) {
		return "", proxy.Failure("invalid_configuration", 503)
	}
	u.Path = strings.TrimRight(u.Path, "/") + suffix
	u.RawPath = ""
	return u.String(), nil
}

func validKey(key []byte) bool {
	if len(key) == 0 || len(key) > 8192 {
		return false
	}
	for _, c := range key {
		if c < 33 || c > 126 {
			return false
		}
	}
	return true
}

// Forward 原生透传 Chat 和 SSE，不转换或丢弃 Gemini 工具签名扩展。
func (c *Client) Forward(ctx context.Context, base string, key []byte, request proxy.Request, observe func(Failure) error, consume func(proxy.Response) error) (resultError error) {
	if !validKey(key) || consume == nil {
		return proxy.Failure("invalid_credential", 503)
	}
	if request.Protocol != "chat" {
		return proxy.Failure("unsupported_protocol", 400)
	}
	body, err := chatBody(request)
	if err != nil {
		return err
	}
	if request.Stream {
		defer clear(body)
	}
	address, err := endpoint(base, "/openai/chat/completions")
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, address, bytes.NewReader(body))
	if err != nil {
		return proxy.Failure("invalid_request", 400)
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
		if err := response.Body.Close(); err != nil && resultError == nil {
			resultError = proxy.Failure("upstream_unavailable", 502)
		}
	}()
	result := proxy.Response{StatusCode: response.StatusCode, ContentType: response.Header.Get("Content-Type"), RetryAfter: response.Header.Get("Retry-After"), Body: response.Body}
	if response.StatusCode >= 400 {
		raw, err := io.ReadAll(io.LimitReader(response.Body, 65537))
		if err != nil || len(raw) > 65536 {
			return proxy.Failure("invalid_upstream_response", 502)
		}
		defer clear(raw)
		failure := ParseFailure(response.StatusCode, result.RetryAfter, raw)
		if observe != nil && failure.Category != "" {
			if err := observe(failure); err != nil {
				return err
			}
		}
		if failure.RetrySeconds > 0 {
			result.RetryAfter = durationSeconds(failure.RetrySeconds)
		}
		result.Body = bytes.NewReader(raw)
	}
	return consume(result)
}
