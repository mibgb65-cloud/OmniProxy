package antigravity

import "time"

// ValidateQuotaSnapshot 与官方响应共用边界，不允许存储层恢复任意 JSON 或非法比例。
func ValidateQuotaSnapshot(value Quota) bool {
	if value.Groups == nil || len(value.FetchedAt) > 64 || len(value.Groups) > 16 {
		return false
	}
	if _, err := time.Parse(time.RFC3339Nano, value.FetchedAt); err != nil {
		return false
	}
	names, ids := map[string]bool{}, map[string]bool{}
	for _, group := range value.Groups {
		if !quotaText(group.Name, 128, false) || names[group.Name] || len(group.Buckets) > 32 {
			return false
		}
		names[group.Name] = true
		for _, bucket := range group.Buckets {
			var fraction *float64
			if bucket.RemainingPercent != nil {
				number := *bucket.RemainingPercent / 100
				fraction = &number
			}
			if _, err := normalizeQuotaBucket(rawQuotaBucket{
				ID: bucket.ID, Name: bucket.Name, Window: bucket.Window, Fraction: fraction, Reset: bucket.ResetsAt,
			}); err != nil || ids[bucket.ID] || len(ids) >= 256 {
				return false
			}
			ids[bucket.ID] = true
		}
	}
	return true
}
