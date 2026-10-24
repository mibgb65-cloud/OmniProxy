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
	if status >= 500 && status <= 599 {
		return true
	}
	var failure *proxy.Error
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
