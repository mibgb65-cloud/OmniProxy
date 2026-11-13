package dumate

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	maxConsoleBytes = 256 << 10
	maxURLBytes     = 2048
)

// Console 调用固定百度控制台路径；禁止重定向，避免把 Cookie 带到其它主机。
type Console struct {
	http *http.Client
	base string
}

// NewConsole 构造隔离传输；测试可注入 transport。
func NewConsole(transport http.RoundTripper) *Console {
	return newConsoleAt(ConsoleBase, transport, ConsoleTimeout)
}

// NewConsoleAt 用指定基址构造控制台客户端；非 HTTPS 基址只允许回环地址，
// 供隔离集成测试使用，生产路径始终使用固定控制台。
func NewConsoleAt(base string, transport http.RoundTripper) (*Console, error) {
	if _, _, err := consoleURL(base, "/probe"); err != nil {
		return nil, err
	}
	return newConsoleAt(base, transport, ConsoleTimeout), nil
}

func newConsoleAt(base string, transport http.RoundTripper, timeout time.Duration) *Console {
	if transport == nil {
		standard := http.DefaultTransport.(*http.Transport).Clone()
		standard.ResponseHeaderTimeout = ConsoleTimeout
		standard.TLSHandshakeTimeout = TLShandshakeTimeout
		transport = standard
	}
	return &Console{
		http: &http.Client{Transport: transport, Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		base: strings.TrimRight(base, "/"),
	}
}

type envelope struct {
	Success *bool           `json:"success"`
	Code    textValue       `json:"code"`
	Message json.RawMessage `json:"message"`
	Result  json.RawMessage `json:"result"`
}

// do 发送一次控制台请求并返回 result；所有失败都映射为固定错误码。
func (c *Console) do(ctx context.Context, method, path string, cookies []Cookie, payload []byte) (json.RawMessage, error) {
	endpoint, host, err := consoleURL(c.base, path)
	if err != nil {
		return nil, err
	}
	header := HeaderFor(cookies, host)
	if header == "" {
		return nil, Failure(CodeCredentialUnavailable)
	}
	var body io.Reader
	if method == http.MethodPost {
		body = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, Failure(CodeConsoleUnavailable)
	}
	request.Header.Set("Cookie", header)
	request.Header.Set("User-Agent", UserAgent)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Referer", c.base+"/")
	if method == http.MethodPost {
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Requested-With", "XMLHttpRequest")
		request.Header.Set("Origin", c.base)
		if token := CSRF(cookies, host); token != "" {
			request.Header.Set("csrftoken", token)
		}
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, Failure(CodeConsoleUnavailable)
	}
	defer func() { _ = response.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxConsoleBytes+1))
	if err != nil || len(raw) > maxConsoleBytes {
		return nil, Failure(CodeConsoleUnavailable)
	}
	return consoleResult(response.StatusCode, response.Header.Get("Content-Type"), raw)
}

// consoleResult 把 HTTP 状态、登录页与业务码统一映射为固定结果。
func consoleResult(status int, contentType string, raw []byte) (json.RawMessage, error) {
	switch {
	case status >= 300 && status < 400:
		// 控制台用重定向把过期会话送到登录页；不跟随，直接判定登录失效。
		return nil, SignInFailure()
	case status == http.StatusTooManyRequests:
		return nil, Failure(CodeRateLimited)
	case status >= 500:
		return nil, Failure(CodeConsoleUnavailable)
	case status < 200 || status >= 300:
		if looksLikeLoginPage(contentType, raw) {
			return nil, SignInFailure()
		}
		return nil, Failure(CodeConsoleUnavailable)
	}
	if looksLikeLoginPage(contentType, raw) {
		return nil, SignInFailure()
	}
	var payload envelope
	if json.Unmarshal(raw, &payload) != nil {
		return nil, Failure(CodeInvalidResponse)
	}
	if payload.Success != nil && !*payload.Success || payload.Code.string() != "" && payload.Code.string() != "0" {
		if messageMentionsSignIn(payload.Message) {
			return nil, SignInFailure()
		}
		return nil, Failure(CodeRejected)
	}
	if len(payload.Result) == 0 || string(payload.Result) == "null" {
		return nil, Failure(CodeInvalidResponse)
	}
	return payload.Result, nil
}

func looksLikeLoginPage(contentType string, raw []byte) bool {
	if strings.Contains(strings.ToLower(contentType), "text/html") {
		return true
	}
	prefix := strings.ToLower(string(raw[:min(len(raw), 512)]))
	return strings.Contains(prefix, "<!doctype") || strings.Contains(prefix, "<html")
}

// messageMentionsSignIn 只在业务码失败时判断是否属于登录态问题；
// 上游原文不进入返回值或日志。
func messageMentionsSignIn(raw json.RawMessage) bool {
	text := strings.ToLower(string(raw))
	for _, marker := range []string{"登录凭证", "重新登录", "登录失效", "login", "sign in", "unauthorized"} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

// consoleURL 拼接固定路径，只接受 HTTPS 或回环 HTTP。
func consoleURL(base, path string) (string, string, error) {
	if base == "" || len(base) > maxURLBytes || !strings.HasPrefix(path, "/") {
		return "", "", Failure(CodeConsoleUnavailable)
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Hostname() == "" || parsed.Opaque != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", Failure(CodeConsoleUnavailable)
	}
	if parsed.Scheme != "https" {
		ip := net.ParseIP(parsed.Hostname())
		if parsed.Scheme != "http" || !(strings.EqualFold(parsed.Hostname(), "localhost") || ip != nil && ip.IsLoopback()) {
			return "", "", Failure(CodeConsoleUnavailable)
		}
	}
	if port := parsed.Port(); port != "" {
		if value, err := strconv.Atoi(port); err != nil || value < 1 || value > 65535 {
			return "", "", Failure(CodeConsoleUnavailable)
		}
	}
	target, query, _ := strings.Cut(path, "?")
	parsed.Path = strings.TrimRight(parsed.Path, "/") + target
	parsed.RawQuery = query
	parsed.RawPath = ""
	return parsed.String(), parsed.Hostname(), nil
}
