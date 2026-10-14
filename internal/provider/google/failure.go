package google

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"
)

// Failure 只传递经过验证的资源分类，不保存上游正文或错误消息。
type Failure struct {
	Category     string `json:"category"`
	Metric       string `json:"metric,omitempty"`
	Limit        *int64 `json:"limit,omitempty"`
	RetrySeconds int64  `json:"retrySeconds,omitempty"`
}

func durationSeconds(value int64) string { return strconv.FormatInt(value, 10) }

func ParseFailure(status int, retry string, raw []byte) Failure {
	result := Failure{}
	switch status {
	case 401:
		result.Category = "authentication_failed"
	case 403:
		result.Category = "permission_denied"
	case 402:
		result.Category = "payment_required"
	case 429:
		result.Category = "rate_limited"
	default:
		return result
	}
	if seconds, err := strconv.ParseInt(retry, 10, 64); err == nil && seconds > 0 && seconds <= 86400 {
		result.RetrySeconds = seconds
	}
	var envelope struct {
		Error struct {
			Type    string `json:"type"`
			Status  string `json:"status"`
			Details []struct {
				Type       string `json:"@type"`
				Delay      string `json:"retryDelay"`
				Violations []struct {
					Metric string `json:"quotaMetric"`
					ID     string `json:"quotaId"`
					Value  string `json:"quotaValue"`
				} `json:"violations"`
			} `json:"details"`
		} `json:"error"`
	}
	if len(raw) > 65536 || json.Unmarshal(raw, &envelope) != nil {
		return result
	}
	for _, detail := range envelope.Error.Details {
		if detail.Type == "type.googleapis.com/google.rpc.RetryInfo" && result.RetrySeconds == 0 && len(detail.Delay) < 30 {
			if duration, err := time.ParseDuration(detail.Delay); err == nil && duration > 0 && duration <= 24*time.Hour {
				result.RetrySeconds = int64(math.Ceil(duration.Seconds()))
			}
		}
		if status != 429 || detail.Type != "type.googleapis.com/google.rpc.QuotaFailure" || len(detail.Violations) > 16 {
			continue
		}
		for _, violation := range detail.Violations {
			if !validMetric(violation.Metric) {
				continue
			}
			if result.Metric == "" {
				result.Metric = violation.Metric
			}
			if result.Metric != violation.Metric {
				continue
			}
			if value, err := strconv.ParseInt(violation.Value, 10, 64); err == nil && value >= 0 && value <= 1e12 {
				result.Limit = &value
			}
			if len(violation.ID) < 256 && strings.Contains(strings.ToLower(violation.ID), "perday") {
				result.Category = "daily_exhausted"
			}
		}
	}
	return result
}

func validMetric(value string) bool { return value != "" && ValidMetric(value) }

// ValidMetric 允许空值或 Google 官方指标名，禁止错误正文进入本地资源记录。
func ValidMetric(value string) bool {
	if value == "" {
		return true
	}
	if len(value) > 256 || !strings.HasPrefix(value, "generativelanguage.googleapis.com/") {
		return false
	}
	for _, c := range value {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("/._-", c)) {
			return false
		}
	}
	return true
}
