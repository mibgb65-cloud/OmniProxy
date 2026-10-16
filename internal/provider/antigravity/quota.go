package antigravity

import (
	"context"
	"math"
	"time"
	"unicode"
	"unicode/utf8"
)

// Quota 保留官方额度分组，不按模型或跨组相加。
type Quota struct {
	Groups    []QuotaGroup `json:"groups"`
	FetchedAt string       `json:"fetchedAt"`
}

// QuotaGroup 是官方返回的独立模型共享额度组。
type QuotaGroup struct {
	Name    string        `json:"name"`
	Buckets []QuotaBucket `json:"buckets"`
}

// QuotaBucket 的 nil 比例表示未返回，不代表耗尽或满额。
type QuotaBucket struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Window           string   `json:"window"`
	RemainingPercent *float64 `json:"remainingPercent"`
	ResetsAt         string   `json:"resetsAt,omitempty"`
}

type rawQuotaBucket struct {
	ID       string   `json:"bucketId"`
	Name     string   `json:"displayName"`
	Window   string   `json:"window"`
	Fraction *float64 `json:"remainingFraction"`
	Reset    string   `json:"resetTime"`
}

func quotaText(value string, maximum int, optional bool) bool {
	return (optional || value != "") && len(value) <= maximum && utf8.ValidString(value) && stringsNoControl(value)
}

func stringsNoControl(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func normalizeQuotaBucket(raw rawQuotaBucket) (QuotaBucket, error) {
	if !safeString(raw.ID, 128) || !quotaText(raw.Name, 128, true) || !quotaText(raw.Window, 32, true) {
		return QuotaBucket{}, Failure("invalid_response")
	}
	bucket := QuotaBucket{ID: raw.ID, Name: raw.Name, Window: raw.Window}
	if raw.Fraction != nil {
		fraction := *raw.Fraction
		if math.IsNaN(fraction) || math.IsInf(fraction, 0) || fraction < 0 || fraction > 1 {
			return QuotaBucket{}, Failure("invalid_response")
		}
		// 保留原始精度，防止很小的正余额被四舍五入为“耗尽”。
		percent := fraction * 100
		bucket.RemainingPercent = &percent
	}
	if raw.Reset != "" {
		reset, err := time.Parse(time.RFC3339Nano, raw.Reset)
		if len(raw.Reset) > 64 || err != nil {
			return QuotaBucket{}, Failure("invalid_response")
		}
		bucket.ResetsAt = reset.UTC().Format(time.RFC3339Nano)
	}
	return bucket, nil
}

// ReadQuota 只调用官方 summary；模型目录的 quotaInfo 不能替代每周额度。
func (c *Client) ReadQuota(ctx context.Context, cred Credentials) (Quota, error) {
	var raw struct {
		Groups *[]struct {
			Name    string           `json:"displayName"`
			Buckets []rawQuotaBucket `json:"buckets"`
		} `json:"groups"`
	}
	if !cred.Valid() {
		return Quota{}, Failure("reauth_required")
	}
	if err := c.post(ctx, inferenceURL+"/v1internal:retrieveUserQuotaSummary", cred.AccessToken, map[string]string{"project": cred.Project}, &raw); err != nil {
		return Quota{}, err
	}
	if raw.Groups == nil || len(*raw.Groups) > 16 {
		return Quota{}, Failure("invalid_response")
	}
	result := Quota{Groups: []QuotaGroup{}, FetchedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	groupNames, bucketIDs := map[string]bool{}, map[string]bool{}
	for _, group := range *raw.Groups {
		if !quotaText(group.Name, 128, false) || groupNames[group.Name] || len(group.Buckets) > 32 {
			return Quota{}, Failure("invalid_response")
		}
		groupNames[group.Name] = true
		normalized := QuotaGroup{Name: group.Name, Buckets: []QuotaBucket{}}
		for _, value := range group.Buckets {
			bucket, err := normalizeQuotaBucket(value)
			if err != nil || bucketIDs[value.ID] || len(bucketIDs) >= 256 {
				return Quota{}, Failure("invalid_response")
			}
			bucketIDs[value.ID] = true
			normalized.Buckets = append(normalized.Buckets, bucket)
		}
		result.Groups = append(result.Groups, normalized)
	}
	return result, nil
}
