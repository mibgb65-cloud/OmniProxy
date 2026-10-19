package subscription

import (
	"time"
)

const maxQuotaObservations = 4096
const maxQuotaBlock = 8 * 24 * time.Hour

type quotaObservation struct {
	version       string
	quota         Quota
	rejectedUntil time.Time
}

// rememberQuota 保存已通过账号身份校验的快照，不在推理热路径额外启动官方子进程。
func (s *Service) rememberQuota(id, version string, quota Quota) {
	if id == "" || version == "" {
		return
	}
	at, err := time.Parse(time.RFC3339, quota.FetchedAt)
	if err != nil || at.After(time.Now().Add(time.Minute)) {
		return
	}
	for _, window := range []*QuotaWindow{quota.FiveHour, quota.Weekly} {
		if window != nil && (window.RemainingPercent < 0 || window.RemainingPercent > 100) {
			return
		}
	}
	if quota.FiveHour != nil {
		value := *quota.FiveHour
		quota.FiveHour = &value
	}
	if quota.Weekly != nil {
		value := *quota.Weekly
		quota.Weekly = &value
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureQuotaSpace(id)
	previous := s.quotaObservations[id]
	if previous.version == version {
		if previousAt, err := time.Parse(time.RFC3339, previous.quota.FetchedAt); err == nil && at.Before(previousAt) {
			return
		}
		quota = retainMissingExhaustedWindows(previous.quota, quota, time.Now())
	}
	current := quotaObservation{version: version, quota: quota}
	if previous.version == version && (quota.Weekly == nil || quota.FiveHourEnabled && quota.FiveHour == nil) {
		current.rejectedUntil = previous.rejectedUntil
	}
	s.quotaObservations[id] = current
}

func (s *Service) ensureQuotaSpace(id string) {
	if s.quotaObservations == nil {
		s.quotaObservations = map[string]quotaObservation{}
	}
	if _, exists := s.quotaObservations[id]; exists || len(s.quotaObservations) < maxQuotaObservations {
		return
	}
	var oldest string
	var oldestAt string
	for key, value := range s.quotaObservations {
		if oldest == "" || value.quota.FetchedAt < oldestAt {
			oldest, oldestAt = key, value.quota.FetchedAt
		}
	}
	delete(s.quotaObservations, oldest)
}

// QuotaBlockedUntil 任一共享窗口耗尽即跳过账号；凭据版本变化或重置后允许正常业务请求重试。
func (s *Service) QuotaBlockedUntil(id, version string, now time.Time) (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	observation, ok := s.quotaObservations[id]
	if !ok || observation.version != version {
		return time.Time{}, false
	}
	until := observation.rejectedUntil
	windows := []*QuotaWindow{observation.quota.Weekly}
	if observation.quota.FiveHourEnabled {
		windows = append(windows, observation.quota.FiveHour)
	}
	fetched, err := time.Parse(time.RFC3339, observation.quota.FetchedAt)
	if err == nil && !fetched.After(now.Add(time.Minute)) {
		for _, window := range windows {
			if window == nil || window.RemainingPercent != 0 {
				continue
			}
			reset, parseErr := time.Parse(time.RFC3339, window.ResetsAt)
			if parseErr != nil {
				reset = fetched.Add(time.Minute)
			}
			if reset.After(now) && !reset.After(now.Add(maxQuotaBlock)) && reset.After(until) {
				until = reset
			}
		}
	}
	return until, until.After(now)
}

// RecordQuotaRejection 仅由固定订阅端点已确认的执行前额度拒绝调用，不把一般流失败当作额度不足。
func (s *Service) RecordQuotaRejection(id, version string, until time.Time) {
	now := time.Now()
	if id == "" || version == "" || !until.After(now) || until.After(now.Add(maxQuotaBlock)) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureQuotaSpace(id)
	observation := s.quotaObservations[id]
	if observation.version != version {
		observation = quotaObservation{version: version}
	}
	if until.After(observation.rejectedUntil) {
		observation.rejectedUntil = until
	}
	s.quotaObservations[id] = observation
}

// 缺失窗口不是额度恢复的证据；只保留重置时间仍可信的旧耗尽窗口。
func retainMissingExhaustedWindows(previous, current Quota, now time.Time) Quota {
	valid := func(window *QuotaWindow) bool {
		if window == nil || window.RemainingPercent != 0 {
			return false
		}
		reset, err := time.Parse(time.RFC3339, window.ResetsAt)
		return err == nil && reset.After(now) && !reset.After(now.Add(maxQuotaBlock))
	}
	if current.Weekly == nil && valid(previous.Weekly) {
		current.Weekly = previous.Weekly
	}
	if current.FiveHourEnabled && current.FiveHour == nil && valid(previous.FiveHour) {
		current.FiveHour = previous.FiveHour
	}
	return current
}
