package codex

import (
	"context"
	"unicode"
	"unicode/utf8"
)

// RateLimits 仅解析公开额度字段，不包含令牌或完整官方响应。
type RateLimits struct {
	AccountID    string                        `json:"accountId"`
	Snapshot     *RateLimitSnapshot            `json:"rateLimits"`
	ByLimitID    map[string]*RateLimitSnapshot `json:"rateLimitsByLimitId"`
	ResetCredits *ResetCreditSummary           `json:"rateLimitResetCredits"`
}

type RateLimitSnapshot struct {
	LimitID   string           `json:"limitId"`
	Plan      string           `json:"planType"`
	Primary   *RateLimitWindow `json:"primary"`
	Secondary *RateLimitWindow `json:"secondary"`
}

type RateLimitWindow struct {
	UsedPercent  *int32 `json:"usedPercent"`
	DurationMins *int64 `json:"windowDurationMins"`
	ResetsAt     *int64 `json:"resetsAt"`
}

func validMetadata(value string, max int) bool {
	if len(value) > max || !utf8.ValidString(value) {
		return false
	}
	for _, char := range value {
		if unicode.IsControl(char) {
			return false
		}
	}
	return true
}

func validRateSnapshot(snapshot *RateLimitSnapshot) bool {
	if snapshot == nil {
		return true
	}
	if !validMetadata(snapshot.LimitID, 128) || !validMetadata(snapshot.Plan, 80) {
		return false
	}
	for _, window := range []*RateLimitWindow{snapshot.Primary, snapshot.Secondary} {
		if window == nil {
			continue
		}
		if window.UsedPercent == nil || *window.UsedPercent < 0 || window.DurationMins != nil && (*window.DurationMins <= 0 || *window.DurationMins > 525600) || window.ResetsAt != nil && (*window.ResetsAt < 0 || *window.ResetsAt > 253402300799) {
			return false
		}
	}
	return true
}

// RateLimits 通过官方组件读取当前隔离授权的用量快照。
func (c *rpcClient) RateLimits(ctx context.Context) (RateLimits, error) {
	var result RateLimits
	if err := c.call(ctx, "account/rateLimits/read", map[string]any{}, &result); err != nil {
		return RateLimits{}, err
	}
	if !validMetadata(result.AccountID, 128) || !validRateSnapshot(result.Snapshot) || len(result.ByLimitID) > 32 || result.Snapshot == nil && len(result.ByLimitID) == 0 {
		return RateLimits{}, ErrProtocol
	}
	for id, snapshot := range result.ByLimitID {
		if !validMetadata(id, 128) || !validRateSnapshot(snapshot) {
			return RateLimits{}, ErrProtocol
		}
	}
	if !validResetCredits(result.ResetCredits) {
		return RateLimits{}, ErrProtocol
	}
	return result, nil
}
