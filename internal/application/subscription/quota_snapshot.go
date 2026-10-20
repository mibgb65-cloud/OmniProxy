package subscription

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"omniproxy/internal/application/account"
)

type quotaCacheScope struct {
	identity, profile string
	workspaceKnown    bool
}

func quotaScope(item account.Item, profile string) quotaCacheScope {
	email := strings.ToLower(strings.TrimSpace(item.AuthEmail))
	hash := func(values []string) string {
		data, _ := json.Marshal(values)
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:])
	}
	return quotaCacheScope{
		identity:       hash([]string{item.ProviderID, email, item.AuthAccountID}),
		profile:        hash([]string{item.ProviderID, email, profile}),
		workspaceKnown: item.AuthAccountID != "",
	}
}

func (s quotaCacheScope) matches(other quotaCacheScope) bool {
	if s.workspaceKnown && other.workspaceKnown {
		return s.identity == other.identity
	}
	// 工作区 ID 暂缺／随后补全时，只允许同一成员、同一隔离配置恢复，不能跨重新授权猜身份。
	return s.profile == other.profile
}

func decodeQuotaSnapshot(data []byte) (Quota, error) {
	var value Quota
	if len(data) == 0 || len(data) > 16384 {
		return value, failure("quota_cache_unavailable")
	}
	reader := json.NewDecoder(bytes.NewReader(data))
	reader.DisallowUnknownFields()
	if reader.Decode(&value) != nil || validateQuotaSnapshot(value) != nil || value.ResetCredits != nil {
		return Quota{}, failure("quota_cache_unavailable")
	}
	if reader.Decode(&struct{}{}) != io.EOF {
		return Quota{}, failure("quota_cache_unavailable")
	}
	return value, nil
}

func validateQuotaSnapshot(value Quota) error {
	fetched, err := time.Parse(time.RFC3339, value.FetchedAt)
	if err != nil || fetched.After(time.Now().Add(time.Minute)) || len(value.Plan) > 80 || !utf8.ValidString(value.Plan) || strings.IndexFunc(value.Plan, unicode.IsControl) >= 0 {
		return failure("quota_cache_unavailable")
	}
	for _, window := range []*QuotaWindow{value.FiveHour, value.Weekly} {
		if window == nil {
			continue
		}
		if window.RemainingPercent < 0 || window.RemainingPercent > 100 || len(window.ResetsAt) > 40 {
			return failure("quota_cache_unavailable")
		}
		if window.UsedPercent != nil && (math.IsNaN(*window.UsedPercent) || math.IsInf(*window.UsedPercent, 0) || *window.UsedPercent < 0 || *window.UsedPercent > 100) {
			return failure("quota_cache_unavailable")
		}
		if window.ObservedAt != "" {
			observed, err := time.Parse(time.RFC3339, window.ObservedAt)
			if err != nil || observed.After(fetched) {
				return failure("quota_cache_unavailable")
			}
		}
		if window.ResetsAt != "" {
			if _, err := time.Parse(time.RFC3339, window.ResetsAt); err != nil {
				return failure("quota_cache_unavailable")
			}
		}
	}
	return nil
}

// 合并只用于展示；旧窗口和旧重置时间不会回灌给内存额度观察或自动激活。
func mergeQuotaSnapshot(previous, next Quota) Quota {
	merge := func(old, current *QuotaWindow) *QuotaWindow {
		if current == nil {
			if old == nil {
				return nil
			}
			next.FromCache = true
			copy := *old
			return &copy
		}
		copy := *current
		if copy.ResetsAt == "" && old != nil && old.ResetsAt != "" {
			copy.ResetsAt, next.FromCache = old.ResetsAt, true
			// 旧重置目标仅供显示；不能据此将新百分比与推断的窗口成本配对。
			copy.ObservedAt = ""
		}
		return &copy
	}
	if next.FiveHour == nil && next.Weekly == nil {
		next.FetchedAt = previous.FetchedAt
	}
	next.FiveHour = merge(previous.FiveHour, next.FiveHour)
	next.Weekly = merge(previous.Weekly, next.Weekly)
	return next
}
