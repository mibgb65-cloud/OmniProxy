package chatgpt

import (
	"omniproxy/internal/core/proxy"
	"time"
)

// 额度拒绝判定只有一份实现：透明转发入口也要用同一套证据，不能在适配器里另存一套。
func canInspectQuotaResponse(status int, kind string) bool {
	return proxy.InspectableQuotaResponse(status, kind)
}

func quotaRejection(prefix []byte, now time.Time) (bool, time.Time) {
	return proxy.QuotaRejected(prefix, now)
}
