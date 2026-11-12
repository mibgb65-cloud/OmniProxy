package gateway

import (
	"errors"
	"time"

	"omniproxy/internal/core/proxy"
)

// refusedBefore 是「该账号 6 小时内拒绝过同一 system prompt」的跳过信号：
// 有后续候选时不再发送，也不写账号冷却（Google 订阅通道）。
func refusedBefore(err error) (*proxy.Error, bool) {
	var failure *proxy.Error
	if errors.As(err, &failure) && failure.Code == "prompt_refused_before" {
		return failure, true
	}
	return nil, false
}

// quotaExhausted 是 Google 订阅的容量耗尽 429：上游给出了模型级重置时间。
func quotaExhausted(err error) (*proxy.Error, bool) {
	var failure *proxy.Error
	if errors.As(err, &failure) && failure.Code == "upstream_quota_exhausted" {
		return failure, true
	}
	return nil, false
}

// quotaRetryDelay 是休息本账号的时长：优先采用上游的重置秒数，异常时退回 30 秒。
func quotaRetryDelay(failure *proxy.Error, now time.Time) time.Duration {
	if delay, ok := retryDelay(failure.RetryAfter, now); ok && delay <= maxQuotaBlockWindow {
		return delay
	}
	return 30 * time.Second
}

// hasLaterCandidate 报告后面是否还有未尝试的候选。
func hasLaterCandidate(later []proxy.AccountCandidate, seen map[string]bool) bool {
	for _, candidate := range later {
		if !seen[candidate.AccountID] {
			return true
		}
	}
	return false
}
