package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"omniproxy/internal/token"
)

var claudeSubscriptionUsageEndpoint = "https://api.anthropic.com/api/oauth/usage"

// The endpoint throttles unrecognised clients aggressively, so it has to be
// called the way Claude Code calls it. Bump this when the upstream client
// version moves on.
const (
	claudeSubscriptionUsageUserAgent = "claude-code/2.0.0"
	claudeSubscriptionUsageBeta      = "oauth-2025-04-20"
)

type claudeUsageWindow struct {
	Utilization *float64 `json:"utilization"`
	ResetsAt    string   `json:"resets_at"`
}

// queryClaudeSubscriptionUsage reads the plan rate-limit windows behind a Claude
// OAuth login. It is opt-in: the endpoint is undocumented and Anthropic
// restricts subscription credentials to its own clients, so the user has to
// enable it knowingly.
func (v *Validator) queryClaudeSubscriptionUsage(ctx context.Context, selected token.Token) (token.UsageInfo, *int, bool) {
	if !v.cfg.ClaudeSubscriptionUsageEnabled {
		return token.UsageInfo{}, nil, false
	}

	body, status, err := v.queryJSONStatus(ctx, selected, claudeSubscriptionUsageEndpoint, map[string]string{
		// applyAuth adds Messages API headers that do not belong on this call,
		// so the beta header is narrowed to the one this endpoint expects.
		"anthropic-beta": claudeSubscriptionUsageBeta,
		"User-Agent":     claudeSubscriptionUsageUserAgent,
		"Content-Type":   "application/json",
	})
	if err != nil {
		return claudeUsageUnavailable(status), nil, true
	}

	usage, ok := parseClaudeSubscriptionUsage(body)
	if !ok {
		return token.UsageInfo{
			Source:  "claude",
			Message: "Claude 用量接口返回了无法识别的结构",
		}, nil, true
	}
	remaining := usage.EffectiveRemainingPercent()
	return usage, &remaining, true
}

// claudeUsageUnavailable reports the failure to the UI instead of silently
// showing nothing, so an endpoint or credential problem is diagnosable.
func claudeUsageUnavailable(status int) token.UsageInfo {
	if status > 0 {
		return token.UsageInfo{
			Source:  "claude",
			Message: fmt.Sprintf("Claude 用量接口返回 %d", status),
		}
	}
	return token.UsageInfo{
		Source:  "claude",
		Message: "无法连接 Claude 用量接口",
	}
}

func parseClaudeSubscriptionUsage(body []byte) (token.UsageInfo, bool) {
	var payload struct {
		FiveHour *claudeUsageWindow `json:"five_hour"`
		SevenDay *claudeUsageWindow `json:"seven_day"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return token.UsageInfo{}, false
	}
	if !claudeUsageWindowUsable(payload.FiveHour) && !claudeUsageWindowUsable(payload.SevenDay) {
		return token.UsageInfo{}, false
	}

	usage := token.UsageInfo{
		Source:                     "claude",
		Message:                    "Claude subscription usage endpoint",
		SubscriptionQuotaAvailable: true,
	}
	assignClaudeUsageWindow(&usage, "primary", payload.FiveHour)
	assignClaudeUsageWindow(&usage, "secondary", payload.SevenDay)
	return usage, true
}

func claudeUsageWindowUsable(window *claudeUsageWindow) bool {
	return window != nil && window.Utilization != nil
}

func assignClaudeUsageWindow(usage *token.UsageInfo, kind string, window *claudeUsageWindow) {
	if !claudeUsageWindowUsable(window) {
		return
	}

	used := *window.Utilization
	if used < 0 {
		used = 0
	}
	if used > 100 {
		used = 100
	}
	usedPercent := int(math.Round(used))
	resetAt := claudeUsageResetUnix(window.ResetsAt)

	if kind == "secondary" {
		usage.SecondaryUsedPercent = usedPercent
		usage.SecondaryUsedPercentExact = used
		usage.SecondaryRemainingPercent = 100 - usedPercent
		usage.SecondaryResetAt = resetAt
		return
	}
	usage.PrimaryUsedPercent = usedPercent
	usage.PrimaryUsedPercentExact = used
	usage.PrimaryRemainingPercent = 100 - usedPercent
	usage.PrimaryResetAt = resetAt
}

func claudeUsageResetUnix(value string) int64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return 0
	}
	return parsed.Unix()
}
