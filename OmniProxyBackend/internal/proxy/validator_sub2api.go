package proxy

import (
	"math"
	"omniproxy/internal/token"
	"strings"
)

func parseSub2APIUsage(body []byte) (token.UsageInfo, *int, bool) {
	payload, err := decodeObject(body)
	if err != nil {
		return token.UsageInfo{}, nil, false
	}

	mode, _ := stringFromAny(payload["mode"])
	planType, _ := stringFromAny(payload["planName"])
	if planType == "" {
		planType = "sub2api API Key"
		if mode != "" {
			planType = "sub2api " + strings.ReplaceAll(mode, "_", " ")
		}
	}

	usage := token.UsageInfo{
		Source:   token.ProviderSub2API,
		PlanType: planType,
		Message:  "sub2api usage",
	}
	found := mode != "" || planType != ""

	balanceRemaining, balanceFound := sub2APIBalanceUsage(payload, &usage)
	if balanceFound {
		found = true
	}

	subscriptionRemaining, subscriptionFound := sub2APISubscriptionUsage(payload, &usage)
	if subscriptionFound {
		found = true
	}

	rateLimitRemaining, rateLimitFound := sub2APIRateLimitUsage(payload, &usage)
	if rateLimitFound {
		found = true
	}

	if valid, ok := boolFromAnyOK(payload["isValid"]); ok && !valid {
		usage.LimitReached = true
		zero := 0
		return usage, &zero, true
	}
	if status, ok := stringFromAny(payload["status"]); ok && sub2APIStatusLimited(status) {
		usage.LimitReached = true
		zero := 0
		return usage, &zero, true
	}

	remaining := minRemainingPercent(balanceRemaining, subscriptionRemaining, rateLimitRemaining)
	if remaining != nil && usage.SubscriptionQuotaAvailable {
		usage.PrimaryUsedPercent = 100 - *remaining
		usage.PrimaryRemainingPercent = *remaining
	}
	if remaining != nil && *remaining <= 0 {
		usage.LimitReached = true
	}

	return usage, remaining, found
}

func parseNewAPIUsage(body []byte) (token.UsageInfo, *int, bool) {
	payload, err := decodeObject(body)
	if err != nil {
		return token.UsageInfo{}, nil, false
	}
	if valid, ok := boolFromAnyOK(payload["code"]); ok && !valid {
		return token.UsageInfo{}, nil, false
	}

	data, ok := payload["data"].(map[string]any)
	if !ok {
		return token.UsageInfo{}, nil, false
	}

	total, totalOK := floatFromAny(data["total_granted"])
	used, usedOK := floatFromAny(data["total_used"])
	remainingValue, remainingOK := floatFromAny(data["total_available"])
	unlimited := boolFromAny(data["unlimited_quota"], false)

	usage := token.UsageInfo{
		Source:      token.ProviderNewAPI,
		PlanType:    "new-api API Key",
		Message:     "new-api token usage",
		BalanceUnit: "Quota",
	}
	if totalOK {
		usage.BalanceTotal = total
	}
	if usedOK {
		usage.BalanceUsed = used
	}
	if remainingOK {
		usage.BalanceRemaining = remainingValue
	}
	if unlimited {
		usage.BalanceUnlimited = true
	}

	var remaining *int
	switch {
	case unlimited:
		value := 100
		remaining = &value
	case remainingOK:
		value := balanceRemainingPercentFromValues(remainingValue, total, totalOK)
		remaining = &value
		if remainingValue <= 0 {
			usage.LimitReached = true
		}
	}

	return usage, remaining, totalOK || usedOK || remainingOK || unlimited
}

func sub2APIBalanceUsage(payload map[string]any, usage *token.UsageInfo) (*int, bool) {
	var remaining *int
	found := false
	quotaRemainingFound := false

	if quota, ok := payload["quota"].(map[string]any); ok {
		unit := "USD"
		if value, ok := stringFromAny(quota["unit"]); ok {
			unit = value
		}
		usage.BalanceUnit = unit

		limit, limitOK := floatFromAny(quota["limit"])
		used, usedOK := floatFromAny(quota["used"])
		remainingValue, remainingOK := floatFromAny(quota["remaining"])
		if !remainingOK && limitOK && usedOK {
			remainingValue = limit - used
			if remainingValue < 0 {
				remainingValue = 0
			}
			remainingOK = true
		}

		if limitOK {
			usage.BalanceTotal = limit
			found = true
		}
		if usedOK {
			usage.BalanceUsed = used
			found = true
		}
		if remainingOK {
			usage.BalanceRemaining = remainingValue
			quotaRemainingFound = true
			found = true
			value := balanceRemainingPercentFromValues(remainingValue, limit, limitOK)
			remaining = &value
		}
	}

	topRemaining, topRemainingOK := floatFromAny(payload["remaining"])
	if topRemainingOK {
		if unit, ok := stringFromAny(payload["unit"]); ok {
			usage.BalanceUnit = unit
		} else if usage.BalanceUnit == "" {
			usage.BalanceUnit = "USD"
		}
		if !quotaRemainingFound {
			usage.BalanceRemaining = topRemaining
		}
		found = true
		if remaining == nil {
			value := amountRemainingPercent(topRemaining)
			remaining = &value
		}
	}

	if balance, ok := floatFromAny(payload["balance"]); ok {
		if usage.BalanceUnit == "" {
			if unit, ok := stringFromAny(payload["unit"]); ok {
				usage.BalanceUnit = unit
			} else {
				usage.BalanceUnit = "USD"
			}
		}
		if !quotaRemainingFound && !topRemainingOK {
			usage.BalanceRemaining = balance
		}
		found = true
		if remaining == nil {
			value := amountRemainingPercent(balance)
			remaining = &value
		}
	}

	if usage.BalanceRemaining < 0 {
		usage.BalanceUnlimited = true
		value := 100
		remaining = &value
	}

	return remaining, found
}

func sub2APISubscriptionUsage(payload map[string]any, usage *token.UsageInfo) (*int, bool) {
	subscription, ok := payload["subscription"].(map[string]any)
	if !ok {
		return nil, false
	}

	var candidates []int
	primarySet := false
	secondarySet := false
	if used, remaining, ok := sub2APILimitWindow(subscription, "daily_usage_usd", "daily_limit_usd"); ok {
		usage.PrimaryUsedPercent = used
		usage.PrimaryRemainingPercent = remaining
		usage.SubscriptionQuotaAvailable = true
		primarySet = true
		candidates = append(candidates, remaining)
	}
	if used, remaining, ok := sub2APILimitWindow(subscription, "weekly_usage_usd", "weekly_limit_usd"); ok {
		usage.SecondaryUsedPercent = used
		usage.SecondaryRemainingPercent = remaining
		usage.SubscriptionQuotaAvailable = true
		secondarySet = true
		candidates = append(candidates, remaining)
	}
	if used, remaining, ok := sub2APILimitWindow(subscription, "monthly_usage_usd", "monthly_limit_usd"); ok {
		switch {
		case !primarySet:
			usage.PrimaryUsedPercent = used
			usage.PrimaryRemainingPercent = remaining
			primarySet = true
		case !secondarySet:
			usage.SecondaryUsedPercent = used
			usage.SecondaryRemainingPercent = remaining
			secondarySet = true
		}
		usage.SubscriptionQuotaAvailable = true
		candidates = append(candidates, remaining)
	}
	if len(candidates) == 0 {
		return nil, true
	}
	remaining := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate < remaining {
			remaining = candidate
		}
	}
	return &remaining, true
}

func sub2APIRateLimitUsage(payload map[string]any, usage *token.UsageInfo) (*int, bool) {
	items, ok := payload["rate_limits"].([]any)
	if !ok || len(items) == 0 {
		return nil, false
	}

	var candidates []int
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		limit, limitOK := floatFromAny(item["limit"])
		remainingValue, remainingOK := floatFromAny(item["remaining"])
		usedValue, usedOK := floatFromAny(item["used"])
		if !limitOK || limit <= 0 {
			continue
		}
		if !remainingOK && usedOK {
			remainingValue = limit - usedValue
			if remainingValue < 0 {
				remainingValue = 0
			}
			remainingOK = true
		}
		if !remainingOK {
			continue
		}

		used := percent(((limit - remainingValue) / limit) * 100)
		remaining := 100 - used
		if remaining < 0 {
			remaining = 0
		}
		resetAt := unixSecondsFromAny(item["reset_at"])
		window, _ := stringFromAny(item["window"])
		if strings.EqualFold(window, "7d") || strings.EqualFold(window, "1w") {
			usage.SecondaryUsedPercent = used
			usage.SecondaryRemainingPercent = remaining
			usage.SecondaryResetAt = resetAt
		} else {
			usage.PrimaryUsedPercent = used
			usage.PrimaryRemainingPercent = remaining
			usage.PrimaryResetAt = resetAt
		}
		usage.SubscriptionQuotaAvailable = true
		candidates = append(candidates, remaining)
	}
	if len(candidates) == 0 {
		return nil, false
	}
	remaining := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate < remaining {
			remaining = candidate
		}
	}
	return &remaining, true
}

func sub2APILimitWindow(value map[string]any, usedKey string, limitKey string) (int, int, bool) {
	limit, limitOK := floatFromAny(value[limitKey])
	usedValue, usedOK := floatFromAny(value[usedKey])
	if !limitOK || !usedOK || limit <= 0 {
		return 0, 0, false
	}
	used := percent((usedValue / limit) * 100)
	remaining := 100 - used
	if remaining < 0 {
		remaining = 0
	}
	return used, remaining, true
}

func sub2APIStatusLimited(status string) bool {
	normalized := strings.ToLower(strings.TrimSpace(status))
	return strings.Contains(normalized, "exhaust") || strings.Contains(normalized, "expired")
}

func balanceRemainingPercentFromValues(remaining float64, limit float64, hasLimit bool) int {
	if hasLimit && limit > 0 {
		return percent((remaining / limit) * 100)
	}
	return amountRemainingPercent(remaining)
}

func amountRemainingPercent(remaining float64) int {
	if remaining < 0 {
		return 100
	}
	if remaining <= 0 {
		return 0
	}
	return 100
}

func basePathWithoutVersionSuffix(path string) string {
	path = strings.Trim(strings.TrimSpace(path), "/")
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	if !basePathHasVersionSuffix(path) {
		return "/" + strings.Join(parts, "/")
	}
	parts = parts[:len(parts)-1]
	if len(parts) == 0 {
		return ""
	}
	return "/" + strings.Join(parts, "/")
}

func minRemainingPercent(values ...*int) *int {
	var out *int
	for _, value := range values {
		if value == nil {
			continue
		}
		if out == nil || *value < *out {
			copyValue := *value
			out = &copyValue
		}
	}
	return out
}

func boolFromAnyOK(value any) (bool, bool) {
	typed, ok := value.(bool)
	return typed, ok
}

func percent(value float64) int {
	return int(math.Round(normalizedPercent(value)))
}

func normalizedPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
