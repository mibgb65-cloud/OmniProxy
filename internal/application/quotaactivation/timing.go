package quotaactivation

import (
	"omniproxy/internal/application/subscription"
	"time"
)

// 兼容秒级误差和页面向上取整到 5h1m 的边界；仍须独立观测，不能只凭接近满周期就发送。
const timerSlack = 90 * time.Second
const timerDrift = 2 * time.Second
const maxObservationGap = 15 * time.Minute

type timerObservation struct{ reset, at string }
type windowAssessment struct {
	value             *subscription.QuotaWindow
	floating, waiting bool
	confirmed         string
}
type quotaAssessment struct {
	quota    subscription.Quota
	floating []string
	waiting  bool
}

func duration(kind string) time.Duration {
	if kind == "five_hour" {
		return 5 * time.Hour
	}
	return 7 * 24 * time.Hour
}
func absolute(value time.Duration) time.Duration {
	if value < 0 {
		return -value
	}
	return value
}
func snapshotTime(value string, now time.Time) (time.Time, bool) {
	at, err := time.Parse(time.RFC3339, value)
	return at, err == nil && at.Unix() > 0 && !at.After(now.Add(5*time.Second)) && now.Sub(at) <= 2*time.Minute
}
func nearFullWindow(reset, at time.Time, kind string) bool {
	return absolute(reset.Sub(at)-duration(kind)) <= timerSlack
}
func copyQuota(value subscription.Quota) subscription.Quota {
	if value.FiveHour != nil {
		window := *value.FiveHour
		value.FiveHour = &window
	}
	if value.Weekly != nil {
		window := *value.Weekly
		value.Weekly = &window
	}
	return value
}
func windowFor(value subscription.Quota, kind string) *subscription.QuotaWindow {
	if kind == "five_hour" {
		return value.FiveHour
	}
	return value.Weekly
}
func setWindow(value *subscription.Quota, kind string, window *subscription.QuotaWindow) {
	if kind == "five_hour" {
		value.FiveHour = window
	} else {
		value.Weekly = window
	}
}

// 百分比可能取整为 100%；未来目标本身不能证明窗口已开始。
func assessWindow(kind string, value *subscription.QuotaWindow, previous timerObservation, fetched string, now time.Time) windowAssessment {
	result := windowAssessment{value: value}
	if value == nil || value.ResetsAt == "" {
		return result
	}
	reset, err := time.Parse(time.RFC3339, value.ResetsAt)
	if err != nil || !reset.After(now) || value.RemainingPercent < 0 || value.RemainingPercent > 100 {
		return windowAssessment{value: value, waiting: true}
	}
	if value.RemainingPercent < 100 {
		result.confirmed = value.ResetsAt
		return result
	}
	at, valid := snapshotTime(fetched, now)
	if !valid {
		return windowAssessment{waiting: true}
	}
	if !nearFullWindow(reset, at, kind) {
		if reset.Sub(at) < duration(kind)-timerSlack {
			result.confirmed = value.ResetsAt
			return result
		}
		return windowAssessment{waiting: true}
	}
	oldAt, e1 := time.Parse(time.RFC3339, previous.at)
	oldReset, e2 := time.Parse(time.RFC3339, previous.reset)
	gap := at.Sub(oldAt)
	if e1 != nil || e2 != nil || gap < wakeDebounce || gap > maxObservationGap || !nearFullWindow(oldReset, oldAt, kind) {
		return windowAssessment{waiting: true}
	}
	delta := reset.Sub(oldReset)
	if absolute(delta) <= timerDrift {
		result.confirmed = value.ResetsAt
		return result
	}
	if absolute(delta-gap) <= timerDrift {
		// 仅在独立间隔观测证明目标随查询滑动后，才按未开始处理。
		clone := *value
		clone.ResetsAt = ""
		result.value = &clone
		result.floating = true
		return result
	}
	return windowAssessment{waiting: true}
}
