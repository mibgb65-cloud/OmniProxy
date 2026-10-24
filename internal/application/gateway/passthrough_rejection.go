package gateway

import (
	"time"
)

// maxQuotaBlockWindow 与订阅额度观察的上限一致：更远的重置时间不作为阻断依据。
const maxQuotaBlockWindow = 8 * 24 * time.Hour

// passthroughRetryableStatus 与 v1 的可重试集合一致，另外把 401 视为账号级失效。
func passthroughRetryableStatus(status int) bool {
	switch status {
	case 401, 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

// passthroughRejection 只依据明确拒绝分类写入冷却，不猜泛化 5xx 之外的语义；
// 上游给出合法重试提示时优先采用，避免把长窗口限流当成 30 秒抖动。
func passthroughRejection(status int, retryAfter string, now time.Time) *retryReason {
	rejected := passthroughRejectionCode(status)
	if rejected == nil {
		return nil
	}
	if delay, ok := retryDelay(retryAfter, now); ok {
		rejected.delay = delay
	}
	return rejected
}

func passthroughRejectionCode(status int) *retryReason {
	switch status {
	case 401:
		return &retryReason{code: "upstream_authentication_failed", delay: 5 * time.Minute}
	case 429:
		return &retryReason{code: "upstream_rate_limited", delay: 30 * time.Second}
	case 500, 502, 503, 504:
		return &retryReason{code: "upstream_unavailable", delay: 30 * time.Second}
	default:
		return nil
	}
}

// quotaRetryReason 按真实重置时间换算执行前额度拒绝的阻断窗口；没有可信重置时间时
// 只短期跳过，不假装额度已经恢复。
func quotaRetryReason(reset time.Time, now time.Time) *retryReason {
	delay := 30 * time.Second
	if reset.After(now) && reset.Sub(now) <= maxQuotaBlockWindow {
		delay = reset.Sub(now)
	}
	return &retryReason{code: "upstream_quota_exhausted", delay: delay}
}
