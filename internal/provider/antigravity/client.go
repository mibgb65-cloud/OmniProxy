// Package antigravity 通过独立 OAuth 授权接入 Google 订阅客户端协议。
package antigravity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"omniproxy/internal/core/proxy"
	"strings"
	"time"
)

const controlURL = "https://cloudcode-pa.googleapis.com"
const inferenceURL = "https://daily-cloudcode-pa.googleapis.com"
const tokenURL = "https://oauth2.googleapis.com/token"
const authURL = "https://accounts.google.com/o/oauth2/v2/auth"
const userURL = "https://www.googleapis.com/oauth2/v2/userinfo"
const maxResponse = 8 << 20

// Client 只访问固定官方端点；测试通过 RoundTripper 注入契约服务。
type Client struct {
	http       *http.Client
	signatures signatureCache
}

// New 创建拒绝重定向且具有有限握手等待的客户端。
func New(transport http.RoundTripper) *Client {
	if transport == nil {
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.ResponseHeaderTimeout = 30 * time.Second
		t.TLSHandshakeTimeout = 10 * time.Second
		transport = t
	}
	return &Client{http: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

// Failure 返回稳定错误码，禁止原始上游正文或网络错误携带授权信息。
func Failure(code string) error { return errors.New("google_subscription: " + code) }

// ProxyError 保留可操作的认证、权限及限流状态，不暴露上游正文。
func ProxyError(err error) error {
	code, status := ErrorCode(err), 502
	switch code {
	case "invalid_request":
		status = 400
	case "reauth_required":
		status = 401
	case "access_denied":
		status = 403
	case "rate_limited":
		status = 429
	}
	return proxy.Failure(code, status)
}

func safeString(value string, max int) bool {
	if len(value) == 0 || len(value) > max {
		return false
	}
	for _, r := range value {
		if r < 33 || r > 126 {
			return false
		}
	}
	return true
}

func (c *Client) request(ctx context.Context, endpoint string, body []byte, token, contentType string) (*http.Response, error) {
	method := http.MethodPost
	if body == nil {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, Failure("invalid_request")
	}
	if token != "" {
		if !safeString(token, 16384) {
			return nil, Failure("invalid_credential")
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", contentType)
	// 内部协议存在最低兼容版本；保留 OmniProxy 身份，不宣称本机 agy 的发行版本。
	req.Header.Set("User-Agent", "antigravity/2.23.0 OmniProxy")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, Failure("upstream_unavailable")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		switch resp.StatusCode {
		case 400:
			// OAuth 令牌拒绝通常用 400；推理／控制端点的 400 不是凭据失效，
			// 否则下游会把请求参数或工具上下文错误误报成 API Key 被撤销。
			if endpoint == tokenURL {
				return nil, Failure("reauth_required")
			}
			return nil, Failure("invalid_request")
		case 401:
			return nil, Failure("reauth_required")
		case 403:
			return nil, Failure("access_denied")
		case 429:
			return nil, Failure("rate_limited")
		default:
			return nil, Failure("upstream_unavailable")
		}
	}
	return resp, nil
}

func (c *Client) read(ctx context.Context, endpoint string, body []byte, token, contentType string, result any) error {
	resp, err := c.request(ctx, endpoint, body, token, contentType)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	defer clear(data)
	if err != nil || len(data) > maxResponse || json.Unmarshal(data, result) != nil {
		return Failure("invalid_response")
	}
	return nil
}

func (c *Client) post(ctx context.Context, endpoint, token string, payload, result any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return Failure("invalid_request")
	}
	defer clear(data)
	return c.read(ctx, endpoint, data, token, "application/json", result)
}

// ErrorCode 只允许已知错误进入 UI，不返回动态正文。
func ErrorCode(err error) string {
	for _, code := range []string{"agy_unavailable", "client_incompatible", "invalid_input", "invalid_request", "busy", "cancelled", "login_expired", "identity_changed", "reauth_required", "access_denied", "rate_limited", "project_unavailable", "invalid_response", "upstream_unavailable", "credential_unavailable", "browser_failed", "cleanup_failed", "quota_changed", "quota_cache_unavailable"} {
		if err != nil && strings.Contains(err.Error(), "google_subscription: "+code) {
			return code
		}
	}
	return "operation_failed"
}
