package proxy

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"

	"omniproxy/internal/token"
)

var claudeSubscriptionUsageEndpoint = "https://api.anthropic.com/api/oauth/usage"

// The endpoint throttles unrecognised clients aggressively, so it has to be
// called the way Claude Code calls it. Bump this when the upstream client
// version moves on.
const claudeSubscriptionUsageUserAgent = "claude-code/2.0.0"

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

	body, ok := v.queryJSONWithHeaders(ctx, selected, claudeSubscriptionUsageEndpoint, map[string]string{
		"User-Agent": claudeSubscriptionUsageUserAgent,
	})
	if !ok {
		return token.UsageInfo{}, nil, false
	}

	usage, ok := parseClaudeSubscriptionUsage(body)
	if !ok {
		return token.UsageInfo{}, nil, false
	}
	remaining := usage.EffectiveRemainingPercent()
	return usage, &remaining, true
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
