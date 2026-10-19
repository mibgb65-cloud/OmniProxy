package proxy

import (
	"strconv"
	"time"
)

// HTTP 与 WebSocket 必须保留相同的状态分类，不能把请求错误与账号限流掩盖为 502。
func upstreamFailure(response Response) *Error {
	if response.PreExecutionQuotaRejected {
		failure := &Error{Code: "upstream_quota_exhausted", StatusCode: 429}
		delay := time.Until(response.QuotaResetAt)
		if delay > 0 && delay <= 8*24*time.Hour {
			failure.RetryAfter = strconv.Itoa(int((delay + time.Second - 1) / time.Second))
		}
		return failure
	}
	failure := &Error{StatusCode: response.StatusCode, Code: "upstream_rejected_request"}
	switch response.StatusCode {
	case 401, 403:
		failure.Code = "upstream_authentication_failed"
	case 402:
		failure.Code = "upstream_insufficient_balance"
	case 429:
		failure.Code = "upstream_rate_limited"
	default:
		if response.StatusCode < 400 || response.StatusCode >= 500 {
			failure.StatusCode, failure.Code = 502, "upstream_unavailable"
		}
	}
	failure.RetryAfter = validatedRetryAfter(response.RetryAfter)
	return failure
}

func validatedRetryAfter(value string) string {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 && seconds <= 86400 {
		return strconv.Itoa(seconds)
	}
	return ""
}
