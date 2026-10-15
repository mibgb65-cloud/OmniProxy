package minimax

import (
	"encoding/json"
	"time"
)

type quotaWire struct {
	Model         string   `json:"model_name"`
	End           *int64   `json:"end_time"`
	WeeklyEnd     *int64   `json:"weekly_end_time"`
	Total         *int64   `json:"current_interval_total_count"`
	Count         *int64   `json:"current_interval_usage_count"`
	Percent       *float64 `json:"current_interval_remaining_percent"`
	Status        *int     `json:"current_interval_status"`
	WeeklyTotal   *int64   `json:"current_weekly_total_count"`
	WeeklyCount   *int64   `json:"current_weekly_usage_count"`
	WeeklyPercent *float64 `json:"current_weekly_remaining_percent"`
	WeeklyStatus  *int     `json:"current_weekly_status"`
	Boost         *int64   `json:"weekly_boost_permille"`
}

func parseQuota(raw []byte, now time.Time) (Quota, error) {
	var page struct {
		Base *struct {
			Code *int `json:"status_code"`
		} `json:"base_resp"`
		Models *[]quotaWire `json:"model_remains"`
	}
	if json.Unmarshal(raw, &page) != nil || page.Models == nil || len(*page.Models) > 128 || page.Base == nil || page.Base.Code == nil || *page.Base.Code != 0 {
		return Quota{}, quotaFailure("invalid_response")
	}
	result := Quota{Buckets: []QuotaBucket{}, FetchedAt: now.Format(time.RFC3339)}
	seen := map[string]bool{}
	for _, row := range *page.Models {
		if !validResourceName(row.Model) || seen[row.Model] {
			return Quota{}, quotaFailure("invalid_response")
		}
		seen[row.Model] = true
		interval, err := quotaWindow(row.Percent, row.Total, row.Count, row.Status, row.End, nil)
		if err != nil {
			return Quota{}, err
		}
		weekly, err := quotaWindow(row.WeeklyPercent, row.WeeklyTotal, row.WeeklyCount, row.WeeklyStatus, row.WeeklyEnd, row.Boost)
		if err != nil {
			return Quota{}, err
		}
		result.Buckets = append(result.Buckets, QuotaBucket{Model: row.Model, Interval: interval, Weekly: weekly})
	}
	return result, nil
}

func validResourceName(s string) bool {
	if len(s) == 0 || len(s) > 128 {
		return false
	}
	for _, r := range s {
		if r < 33 || r > 126 {
			return false
		}
	}
	return true
}

func quotaWindow(percent *float64, total, count *int64, status *int, end, boost *int64) (QuotaWindow, error) {
	result := QuotaWindow{Status: "unknown"}
	if percent != nil && (*percent < 0 || *percent > 100) {
		return result, quotaFailure("invalid_response")
	}
	for _, n := range []*int64{total, count} {
		if n != nil && (*n < 0 || *n > 9007199254740991) {
			return result, quotaFailure("invalid_response")
		}
	}
	if total != nil && count != nil && *total > 0 && *count > *total {
		return result, quotaFailure("invalid_response")
	}
	if boost != nil && (*boost < 0 || *boost > 100000) {
		return result, quotaFailure("invalid_response")
	}
	if end != nil && *end != 0 {
		if *end < 0 || *end > 253402300799000 {
			return result, quotaFailure("invalid_response")
		}
		// 官方字段为 Unix 毫秒；不用 remains_time 倒推时间，防止误当剩余 Token。
		result.ResetsAt = time.UnixMilli(*end).UTC().Format(time.RFC3339)
	}
	if status != nil && (*status < 0 || *status > 100) {
		return result, quotaFailure("invalid_response")
	}
	if status != nil && *status == 3 {
		result.Status = "unlimited"
		return result, nil
	}
	if percent == nil && total != nil && count != nil && *total > 0 {
		// 沿用官方 CLI 的旧版口径：usage_count 是剩余次数；有明确百分比时绝不再次相减。
		p := float64(*count) / float64(*total) * 100
		percent = &p
	}
	result.RemainingPercent = percent
	result.BoostPermille = boost
	if percent != nil {
		result.Status = "known"
	}
	return result, nil
}
