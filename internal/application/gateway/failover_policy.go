package gateway

import (
	"context"
	"errors"
	"net/http"
	"omniproxy/internal/core/proxy"
	"strconv"
	"strings"
	"time"
)

const maxFailoverAttempts = 3

// 账号级拒绝的冷却：模型权益不会自己恢复，按较长窗口休息；登录失效等 401 类
// 需要用户重新授权，按与上游 401 相同的 5 分钟窗口处理。
const (
	entitlementCooldown   = 30 * time.Minute
	authorizationCooldown = 5 * time.Minute
)

type retryReason struct {
	code  string
	delay time.Duration
}

// 只依据明确拒绝分类；服务商整体过载、403 和泛化 5xx 不遍历同入口的账号。
func rejectedAccount(t target, response proxy.Response, now time.Time) *retryReason {
	if t.kind == "openai" && response.PreExecutionQuotaRejected {
		delay := 30 * time.Second
		if response.QuotaResetAt.After(now) && response.QuotaResetAt.Sub(now) <= maxQuotaBlockWindow {
			delay = response.QuotaResetAt.Sub(now)
		}
		return &retryReason{code: "upstream_quota_exhausted", delay: delay}
	}
	code := ""
	switch response.StatusCode {
	case 401:
		code = "upstream_authentication_failed"
	case 402:
		if t.kind == "deepseek" || t.kind == "river" || t.kind == "" {
			code = "upstream_insufficient_balance"
		}
	case 429:
		// 普通 Gemini 限额通常属于项目；不同 Key 不代表独立额度。
		if t.kind != "google" {
			code = "upstream_rate_limited"
		}
	}
	if code == "" {
		return nil
	}
	delay := 30 * time.Second
	if response.StatusCode == 401 || response.StatusCode == 402 {
		delay = 5 * time.Minute
	}
	if parsed, ok := retryDelay(response.RetryAfter, now); ok {
		delay = parsed
	}
	return &retryReason{code: code, delay: delay}
}

// accountRejection 把以 error 形态返回的账号级拒绝翻译成换号理由。
//
// 适配器把上游拒绝直接作为错误返回时没有可检查的响应对象，rejectedAccount 看不到；
// 这里补上账号级的两种情况：模型权益（换号有机会成功）与登录／凭据被拒（ADR-015
// 里的上游 401）。服务商级拒绝（渠道、签名、客户端指纹）不在此列：换号只会得到
// 同样的拒绝，不应遍历整个账号池。
func accountRejection(err error) (*retryReason, *proxy.Error) {
	var failure *proxy.Error
	if !errors.As(err, &failure) {
		return nil, nil
	}
	switch failure.Code {
	case "model_not_entitled":
		return &retryReason{code: failure.Code, delay: entitlementCooldown}, failure
	case "upstream_authentication_failed", "sign_in_expired", "refresh_rejected", "unauthorized":
		return &retryReason{code: failure.Code, delay: authorizationCooldown}, failure
	}
	return nil, nil
}

func retryDelay(value string, now time.Time) (time.Duration, bool) {
	if len(value) == 0 || len(value) > 128 {
		return 0, false
	}
	value = strings.TrimSpace(value)
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && strings.Trim(value, "0123456789") == "" && seconds >= 0 && seconds <= 86400 {
		return time.Duration(seconds) * time.Second, true
	}
	date, err := http.ParseTime(value)
	if err != nil || date.Before(now) || date.Sub(now) > 24*time.Hour {
		return 0, false
	}
	return date.Sub(now), true
}

func riverTransientFailure(ctx context.Context, status int, err error) bool {
	if ctx.Err() != nil {
		return false
	}
	var failure *proxy.Error
	if errors.As(err, &failure) && proxy.ProxyLayerFailure(failure.Code) {
		// 代理层故障与账号无关，不能累加成账号失败。
		return false
	}
	if status >= 500 && status <= 599 {
		return true
	}
	return errors.As(err, &failure) && failure.StatusCode >= 500 && strings.HasPrefix(failure.Code, "upstream_")
}

func attemptFailure(ctx context.Context, err error) (int, string) {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return 504, "request_timeout"
	}
	if ctx.Err() != nil {
		return 499, "request_cancelled"
	}
	if err == nil {
		return 200, ""
	}
	var failure *proxy.Error
	if errors.As(err, &failure) {
		return failure.StatusCode, failure.Code
	}
	return 503, "credential_unavailable"
}
