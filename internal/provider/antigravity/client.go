// Package antigravity 通过独立 OAuth 授权接入 Google 订阅客户端协议。
package antigravity

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"omniproxy/internal/core/netproxy"
	"omniproxy/internal/core/proxy"
	"runtime"
	"strings"
	"sync"
	"time"
)

const controlURL = "https://cloudcode-pa.googleapis.com"
const inferenceURL = "https://daily-cloudcode-pa.googleapis.com"
const tokenURL = "https://oauth2.googleapis.com/token"
const authURL = "https://accounts.google.com/o/oauth2/v2/auth"
const userURL = "https://www.googleapis.com/oauth2/v2/userinfo"
const maxResponse = 8 << 20

// maxClassifiedBody 是 429 正文的分类上限：只用于识别固定措辞，不保存、不返回。
const maxClassifiedBody = 8 << 10

// protocolVersionURL 是 Antigravity 更新清单；只读取其中公开的 version 行。
// 声明为变量以便契约测试指向本地服务。
var protocolVersionURL = "https://antigravity-hub-auto-updater-974169037036.us-central1.run.app/manifest/latest-arm64-mac.yml"

const (
	protocolVersionFallback = "2.23.0"
	protocolVersionTTL      = 6 * time.Hour
)

var protocolVersionState struct {
	sync.Mutex
	value    string
	at       time.Time
	fetching bool
}

// Client 只访问固定官方端点；测试通过 RoundTripper 注入契约服务。
type Client struct {
	http       *http.Client
	signatures signatureCache
	refusals   refusalMarks
	// tiers 记录被拒绝的 entitlement 套餐，24 小时内不再重复尝试。
	tiers struct {
		sync.Mutex
		refused map[string]time.Time
	}
}

// responseHeaderTimeout 是等待上游响应头的上限。
//
// 订阅的思考型模型（Gemini 4 Argon、Claude thinking 等）在第一个 token 之前可能先想
// 几十秒到几分钟，流式请求也要等第一个分片才有响应头；30 秒会把正常请求当成网络失败。
// 连接与 TLS 握手仍有独立上限，真正不可达的主机会更早失败。
const responseHeaderTimeout = 5 * time.Minute

// New 创建拒绝重定向且具有有限握手等待的客户端。
func New(transport http.RoundTripper) *Client {
	if transport == nil {
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.ResponseHeaderTimeout = responseHeaderTimeout
		t.TLSHandshakeTimeout = 10 * time.Second
		transport = netproxy.Wrap(t)
	}
	return &Client{http: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

// Failure 返回稳定错误码，禁止原始上游正文或网络错误携带授权信息。
func Failure(code string) error { return errors.New("google_subscription: " + code) }

// ProxyError 保留可操作的认证、权限及限流状态，不暴露上游正文。
func ProxyError(err error) error {
	code, status := ErrorCode(err), 502
	switch code {
	case "invalid_request", codePromptTurnedAway, codePromptRefusedOnce:
		status = 400
	case "reauth_required":
		status = 401
	case "access_denied":
		status = 403
	case "rate_limited", codeQuotaExhausted:
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

func (c *Client) request(ctx context.Context, endpoint string, body []byte, token, contentType string, headers map[string]string) (*http.Response, error) {
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
	// 采用官方客户端的协议形态；末尾保留 OmniProxy 身份，不宣称已安装 agy 的发行版本。
	req.Header.Set("User-Agent", userAgent())
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, proxy.UpstreamFailure(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusTooManyRequests {
			// 429 的两种语义（内容拒绝 / 容量耗尽）只能从正文措辞区分；
			// 只做有界读取与固定分类，正文随后关闭且不进入错误、日志或界面。
			body, _ := io.ReadAll(io.LimitReader(resp.Body, maxClassifiedBody))
			resp.Body.Close()
			return nil, classifyFailure(resp.StatusCode, body)
		}
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
		default:
			return nil, Failure("upstream_unavailable")
		}
	}
	return resp, nil
}

// userAgent 是官方 Antigravity 客户端的 User-Agent 形态，版本取公开更新清单。
func userAgent() string {
	return "antigravity/hub/" + protocolVersion() + " " + runtime.GOOS + "/" + runtime.GOARCH + " OmniProxy"
}

// protocolVersion 是公开清单里的当前客户端版本，缓存 6 小时；读不到时回退到兼容下限。
// 清单在后台刷新，请求路径不等网络。
func protocolVersion() string {
	protocolVersionState.Lock()
	value := protocolVersionState.value
	if value == "" {
		value = protocolVersionFallback
	}
	stale := time.Since(protocolVersionState.at) >= protocolVersionTTL
	if stale && !protocolVersionState.fetching {
		protocolVersionState.fetching = true
		protocolVersionState.at = time.Now()
		go refreshProtocolVersion()
	}
	protocolVersionState.Unlock()
	return value
}

// refreshProtocolVersion 读取清单并缓存版本；失败保留原值（或回退值）。
func refreshProtocolVersion() {
	value := fetchProtocolVersion()
	protocolVersionState.Lock()
	if value != "" {
		protocolVersionState.value = value
	}
	protocolVersionState.fetching = false
	protocolVersionState.Unlock()
}

func fetchProtocolVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, protocolVersionURL, nil)
	if err != nil {
		return ""
	}
	request.Header.Set("User-Agent", "electron-builder")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return ""
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if err != nil {
		return ""
	}
	return parseManifestVersion(string(data))
}

// parseManifestVersion 取清单里的 version 行；不是纯数字与点的值不采用。
func parseManifestVersion(manifest string) string {
	scanner := bufio.NewScanner(strings.NewReader(manifest))
	for scanner.Scan() {
		value, ok := strings.CutPrefix(strings.TrimSpace(scanner.Text()), "version:")
		if !ok {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `'"`)
		if value != "" && strings.Trim(value, "0123456789.") == "" {
			return value
		}
		break
	}
	return ""
}

func (c *Client) read(ctx context.Context, endpoint string, body []byte, token, contentType string, result any, headers map[string]string) error {
	resp, err := c.request(ctx, endpoint, body, token, contentType, headers)
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
	return c.postWith(ctx, endpoint, token, payload, result, nil)
}

// postWith 与 post 相同，额外发送少量固定头（`onboardUser` 的客户端标识）。
func (c *Client) postWith(ctx context.Context, endpoint, token string, payload, result any, headers map[string]string) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return Failure("invalid_request")
	}
	defer clear(data)
	return c.read(ctx, endpoint, data, token, "application/json", result, headers)
}

// ErrorCode 只允许已知错误进入 UI，不返回动态正文。
func ErrorCode(err error) string {
	var upstream *upstreamError
	if errors.As(err, &upstream) {
		if upstream.reset > 0 {
			return codeQuotaExhausted
		}
		return codeRateLimited
	}
	for _, code := range []string{"agy_unavailable", "client_incompatible", "invalid_input", "invalid_request", "busy", "cancelled", "login_expired", "identity_changed", "reauth_required", "access_denied", "rate_limited", "project_unavailable", "invalid_response", "upstream_unavailable", "credential_unavailable", "browser_failed", "cleanup_failed", "quota_changed", "quota_cache_unavailable", codePromptTurnedAway, codePromptRefusedOnce, codeQuotaExhausted} {
		if err != nil && strings.Contains(err.Error(), "google_subscription: "+code) {
			return code
		}
	}
	return "operation_failed"
}
