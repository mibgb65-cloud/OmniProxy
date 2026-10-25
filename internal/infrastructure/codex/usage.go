package codex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"time"
)

// usageEndpoint 是官方客户端读取订阅额度的固定地址；不随配置变化，也不接受调用方拼接。
const usageEndpoint = "https://chatgpt.com/backend-api/wham/usage"

// usageTimeout 只约束一次额度读取；比标准操作的 30 秒预算更短，留出回退官方组件的余量。
const usageTimeout = 15 * time.Second

const maxUsageBytes = 256 << 10

var (
	// ErrUsageUnauthorized 表示官方明确拒绝本次授权，属于确定性失败，不再回退子进程。
	ErrUsageUnauthorized = errors.New("subscription: quota_not_authenticated")
	// ErrUsageTransient 表示网络、超时、限流或上游 5xx，调用方可回退既有读取链路。
	ErrUsageTransient = errors.New("subscription: quota_transient")
)

// UsageClient 通过官方 HTTP 额度接口读取订阅窗口，不启动官方子进程。
type UsageClient struct {
	http     *http.Client
	endpoint string
}

// NewUsageClient 复用调用方传输；为空时使用仅约束本接口的默认客户端。
func NewUsageClient(client *http.Client) *UsageClient {
	if client == nil {
		client = &http.Client{Timeout: usageTimeout}
	}
	return &UsageClient{http: client, endpoint: usageEndpoint}
}

type usageWindow struct {
	UsedPercent        *float64 `json:"used_percent"`
	LimitWindowSeconds *int64   `json:"limit_window_seconds"`
	ResetAt            *int64   `json:"reset_at"`
}

type usageSnapshot struct {
	PlanType  string `json:"plan_type"`
	RateLimit *struct {
		PrimaryWindow   *usageWindow `json:"primary_window"`
		SecondaryWindow *usageWindow `json:"secondary_window"`
	} `json:"rate_limit"`
}

// Read 只返回公开的套餐、窗口百分比与重置时间，不保留令牌或原始响应。
func (c *UsageClient) Read(ctx context.Context, accessToken []byte, accountID string) (RateLimits, error) {
	if len(accessToken) == 0 || accountID == "" {
		return RateLimits{}, ErrProtocol
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.target(), nil)
	if err != nil {
		return RateLimits{}, ErrProtocol
	}
	request.Header.Set("Authorization", "Bearer "+string(accessToken))
	request.Header.Set("ChatGPT-Account-Id", accountID)
	request.Header.Set("User-Agent", "codex-cli")
	request.Header.Set("Accept", "application/json")
	defer request.Header.Del("Authorization")
	response, err := c.http.Do(request)
	if err != nil {
		// 传输层失败（含超时与取消）由调用方决定是否回退，不在此改变账号状态。
		return RateLimits{}, ErrUsageTransient
	}
	defer func() { _ = response.Body.Close() }()
	switch {
	case response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden:
		return RateLimits{}, ErrUsageUnauthorized
	case response.StatusCode != http.StatusOK:
		return RateLimits{}, ErrUsageTransient
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxUsageBytes))
	if err != nil {
		return RateLimits{}, ErrUsageTransient
	}
	var body usageSnapshot
	if err := json.Unmarshal(raw, &body); err != nil {
		return RateLimits{}, ErrProtocol
	}
	result, ok := body.rateLimits(accountID)
	if !ok {
		return RateLimits{}, ErrProtocol
	}
	return result, nil
}

// target 固定官方地址；零值客户端也退回该地址，避免空白实现访问未预期主机。
func (c *UsageClient) target() string {
	if c.endpoint == "" {
		return usageEndpoint
	}
	return c.endpoint
}

// rateLimits 把官方 HTTP 字段映射到与 stdio 协议一致的快照；没有可用窗口时返回 false。
func (body usageSnapshot) rateLimits(accountID string) (RateLimits, bool) {
	if body.RateLimit == nil {
		return RateLimits{}, false
	}
	snapshot := &RateLimitSnapshot{
		LimitID: "codex",
		Plan:    body.PlanType,
		Primary: body.RateLimit.PrimaryWindow.window(),
	}
	snapshot.Secondary = body.RateLimit.SecondaryWindow.window()
	if snapshot.Primary == nil && snapshot.Secondary == nil {
		return RateLimits{}, false
	}
	if !validMetadata(snapshot.LimitID, 128) || !validMetadata(snapshot.Plan, 80) || !validRateSnapshot(snapshot) {
		return RateLimits{}, false
	}
	return RateLimits{AccountID: accountID, Snapshot: snapshot}, true
}

// window 只在官方同时给出使用比例与窗口时长时才成立，不凭位置猜测额度周期。
func (window *usageWindow) window() *RateLimitWindow {
	if window == nil || window.UsedPercent == nil || window.LimitWindowSeconds == nil {
		return nil
	}
	seconds := *window.LimitWindowSeconds
	if seconds <= 0 || seconds/60 <= 0 {
		return nil
	}
	used := int32(math.Round(*window.UsedPercent))
	used = max(0, min(100, used))
	minutes := seconds / 60
	return &RateLimitWindow{UsedPercent: &used, DurationMins: &minutes, ResetsAt: window.ResetAt}
}
