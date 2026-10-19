package quotaactivation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/subscription"
)

type quotaWindow struct {
	kind  string
	value *subscription.QuotaWindow
}

func windows(quota subscription.Quota) []quotaWindow {
	result := []quotaWindow{{"weekly", quota.Weekly}}
	if quota.FiveHourEnabled {
		result = append(result, quotaWindow{"five_hour", quota.FiveHour})
	}
	return result
}
func identityVersion(item account.Item) string {
	sum := sha256.Sum256([]byte(item.AuthMethod + "\x00" + item.ProviderID + "\x00" + item.AuthAccountID + "\x00" + strings.ToLower(strings.TrimSpace(item.AuthEmail))))
	return hex.EncodeToString(sum[:])
}
func notStarted(window *subscription.QuotaWindow) bool {
	return window != nil && window.RemainingPercent == 100 && window.ResetsAt == ""
}

func usableQuota(quota subscription.Quota) bool {
	for _, window := range windows(quota) {
		if window.value != nil && (window.value.RemainingPercent <= 0 || window.value.RemainingPercent > 100) {
			return false
		}
		if window.value != nil && window.value.ResetsAt != "" {
			reset, err := time.Parse(time.RFC3339, window.value.ResetsAt)
			if err != nil || reset.Unix() <= 0 {
				return false
			}
		}
	}
	return true
}

// NeedsActivation 保留“明确无重置目标”的即时条件；滑动目标由持久观测另行确认。
func NeedsActivation(quota subscription.Quota) bool {
	if !usableQuota(quota) {
		return false
	}
	for _, window := range windows(quota) {
		if notStarted(window.value) {
			return true
		}
	}
	return false
}

func activationConfirmed(before, after subscription.Quota, now time.Time) bool {
	for _, window := range windows(before) {
		if !notStarted(window.value) {
			continue
		}
		var fresh *subscription.QuotaWindow
		if window.kind == "weekly" {
			fresh = after.Weekly
		} else {
			fresh = after.FiveHour
		}
		if fresh == nil || fresh.ResetsAt == "" {
			return false
		}
		if reset, err := time.Parse(time.RFC3339, fresh.ResetsAt); err != nil || !reset.After(now) {
			return false
		}
	}
	return true
}

func windowsActive(quota subscription.Quota, now time.Time) bool {
	for _, window := range windows(quota) {
		if window.value == nil {
			return false
		}
		reset, err := time.Parse(time.RFC3339, window.value.ResetsAt)
		if err != nil || !reset.After(now) {
			return false
		}
	}
	return true
}

func (s *Service) reportQuota(item account.Item, quota subscription.Quota) error {
	// 只比较窗口，不含 FetchedAt；否则每次后台读取都会触发页面刷新与再次检测。
	body, err := json.Marshal(struct {
		Identity   string
		Five, Week *subscription.QuotaWindow
	}{identityVersion(item), quota.FiveHour, quota.Weekly})
	if err != nil {
		return failure("quota_unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.observations[item.ID] != string(body) {
		s.changed = true
		s.observations[item.ID] = string(body)
	}
	return nil
}

// NeedsObservation 额度读取也可唤醒滑动目标检测，但仍受开关和后台限频约束。
func NeedsObservation(quota subscription.Quota) bool {
	if !usableQuota(quota) {
		return false
	}
	if NeedsActivation(quota) {
		return true
	}
	at, err := time.Parse(time.RFC3339, quota.FetchedAt)
	if err != nil {
		return false
	}
	for _, window := range windows(quota) {
		if window.value == nil || window.value.RemainingPercent != 100 {
			continue
		}
		reset, err := time.Parse(time.RFC3339, window.value.ResetsAt)
		if err == nil && nearFullWindow(reset, at, window.kind) {
			return true
		}
	}
	return false
}
