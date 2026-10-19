package subscription

import "time"

// QuotaBalanceWeight 只读取同版本、完整且一分钟内的额度观察，绝不主动查询授权或上游。
// 较紧张的窗口决定可用份额；重置后旧百分比不再作为当前额度使用。
func (s *Service) QuotaBalanceWeight(id, version string, now time.Time) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.quotaObservations[id]
	if !ok || value.version != version {
		return 0, false
	}
	fetched, err := time.Parse(time.RFC3339, value.quota.FetchedAt)
	if err != nil || fetched.After(now.Add(time.Minute)) || now.Sub(fetched) > time.Minute {
		return 0, false
	}
	windows := []*QuotaWindow{value.quota.Weekly}
	if value.quota.FiveHourEnabled {
		windows = append(windows, value.quota.FiveHour)
	}
	weight := 100
	for _, window := range windows {
		if window == nil || window.RemainingPercent < 0 || window.RemainingPercent > 100 {
			return 0, false
		}
		if window.ResetsAt != "" {
			reset, err := time.Parse(time.RFC3339, window.ResetsAt)
			if err != nil || !reset.After(now) {
				return 0, false
			}
		}
		weight = min(weight, window.RemainingPercent)
	}
	return weight, true
}
