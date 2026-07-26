package proxy

import (
	"context"
	"io"
	"net/http"
	"omniproxy/internal/token"
	"strings"
)

func (v *Validator) queryProviderQuota(ctx context.Context, selected token.Token) (token.UsageInfo, *int, bool) {
	if token.NormalizeProvider(selected.Provider) == token.ProviderZhipu && selected.CredentialType == token.CredentialTypeCodingPlan {
		return v.queryZhipuCodingUsage(ctx, selected)
	}

	if token.NormalizeProvider(selected.Provider) == token.ProviderAnthropic && selected.CredentialType == token.CredentialTypeClaudeOAuth {
		return v.queryClaudeSubscriptionUsage(ctx, selected)
	}

	if selected.CredentialType != "" && selected.CredentialType != token.CredentialTypeAPIKey {
		return token.UsageInfo{}, nil, false
	}

	switch token.NormalizeProvider(selected.Provider) {
	case token.ProviderDeepSeek:
		return v.queryDeepSeekBalance(ctx, selected)
	case token.ProviderKimi:
		return v.queryKimiCodingUsage(ctx, selected)
	case token.ProviderZhipu:
		return v.queryZhipuAPIBalance(ctx, selected)
	case token.ProviderMiniMax:
		return v.queryMiniMaxCodingUsage(ctx, selected)
	default:
		return token.UsageInfo{}, nil, false
	}
}

func (v *Validator) queryDeepSeekBalance(ctx context.Context, selected token.Token) (token.UsageInfo, *int, bool) {
	target, err := joinURLPath(v.cfg.DeepSeekBaseURL, "/user/balance")
	if err != nil {
		return token.UsageInfo{}, nil, false
	}
	body, ok := v.queryJSON(ctx, selected, target)
	if !ok {
		return token.UsageInfo{}, nil, false
	}

	payload, err := decodeObject(body)
	if err != nil {
		return token.UsageInfo{}, nil, false
	}

	infos, _ := payload["balance_infos"].([]any)
	entries := deepSeekBalanceEntriesFromInfos(infos)
	balance, unit, found := preferredDeepSeekBalance(entries)
	if !found {
		return token.UsageInfo{}, nil, false
	}

	available := boolFromAny(payload["is_available"], true)
	remaining := 100
	if !available || balance <= 0 {
		remaining = 0
	}

	return token.UsageInfo{
		Source:           token.ProviderDeepSeek,
		BalanceRemaining: balance,
		BalanceUnit:      unit,
		BalancePackages:  deepSeekBalancePackages(entries),
		LimitReached:     !available || balance <= 0,
		Message:          "DeepSeek balance",
	}, &remaining, true
}

type deepSeekBalanceEntry struct {
	unit    string
	balance float64
}

func deepSeekBalanceFromInfos(infos []any) (float64, string, bool) {
	return preferredDeepSeekBalance(deepSeekBalanceEntriesFromInfos(infos))
}

func deepSeekBalanceEntriesFromInfos(infos []any) []deepSeekBalanceEntry {
	entries := make([]deepSeekBalanceEntry, 0, len(infos))
	for _, raw := range infos {
		info, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		balance, ok := deepSeekBalanceValue(info)
		if !ok {
			continue
		}
		unit := "CNY"
		if currency, ok := stringFromAny(info["currency"]); ok && strings.TrimSpace(currency) != "" {
			unit = strings.ToUpper(strings.TrimSpace(currency))
		}
		entries = append(entries, deepSeekBalanceEntry{
			unit:    unit,
			balance: balance,
		})
	}
	return entries
}

func preferredDeepSeekBalance(entries []deepSeekBalanceEntry) (float64, string, bool) {
	if len(entries) == 0 {
		return 0, "", false
	}
	for _, preferredUnit := range []string{"CNY", "USD"} {
		for _, entry := range entries {
			if entry.unit == preferredUnit && entry.balance > 0 {
				return entry.balance, entry.unit, true
			}
		}
	}
	for _, entry := range entries {
		if entry.balance > 0 {
			return entry.balance, entry.unit, true
		}
	}
	return entries[0].balance, entries[0].unit, true
}

func deepSeekBalancePackages(entries []deepSeekBalanceEntry) []token.BalancePackage {
	if len(entries) == 0 {
		return nil
	}
	packages := make([]token.BalancePackage, 0, len(entries))
	for _, entry := range entries {
		packages = append(packages, token.BalancePackage{
			Name:             entry.unit,
			ConsumeType:      "BALANCE",
			BalanceRemaining: entry.balance,
			Unit:             entry.unit,
		})
	}
	return packages
}

func deepSeekBalanceValue(info map[string]any) (float64, bool) {
	if balance, ok := floatFromAny(info["total_balance"]); ok {
		return balance, true
	}
	granted, grantedOK := floatFromAny(info["granted_balance"])
	toppedUp, toppedUpOK := floatFromAny(info["topped_up_balance"])
	if grantedOK || toppedUpOK {
		return granted + toppedUp, true
	}
	return 0, false
}

func (v *Validator) queryKimiCodingUsage(ctx context.Context, selected token.Token) (token.UsageInfo, *int, bool) {
	target, err := joinURLPath(v.cfg.KimiBaseURL, "/v1/usages")
	if err != nil {
		return token.UsageInfo{}, nil, false
	}
	body, ok := v.queryJSON(ctx, selected, target)
	if !ok {
		return token.UsageInfo{}, nil, false
	}

	payload, err := decodeObject(body)
	if err != nil {
		return token.UsageInfo{}, nil, false
	}

	usage := token.UsageInfo{
		Source:   token.ProviderKimi,
		PlanType: "Kimi Token Plan",
		Message:  "Kimi coding usage",
	}

	if limits, ok := payload["limits"].([]any); ok {
		for _, raw := range limits {
			limitItem, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			detail, ok := limitItem["detail"].(map[string]any)
			if !ok {
				continue
			}
			used, remaining, resetAt, ok := usageWindowFromLimit(detail)
			if !ok {
				continue
			}
			usage.PrimaryUsedPercent = used
			usage.PrimaryRemainingPercent = remaining
			usage.PrimaryResetAt = resetAt
			usage.SubscriptionQuotaAvailable = true
			break
		}
	}

	if raw, ok := payload["usage"].(map[string]any); ok {
		used, remaining, resetAt, ok := usageWindowFromLimit(raw)
		if ok {
			usage.SecondaryUsedPercent = used
			usage.SecondaryRemainingPercent = remaining
			usage.SecondaryResetAt = resetAt
			usage.SubscriptionQuotaAvailable = true
		}
	}

	if !usage.SubscriptionQuotaAvailable {
		return token.UsageInfo{}, nil, false
	}
	primaryAvailable := usage.PrimaryUsedPercent != 0 || usage.PrimaryRemainingPercent != 0 || usage.PrimaryResetAt != 0
	remaining := usage.PrimaryRemainingPercent
	if !primaryAvailable && usage.SecondaryRemainingPercent > 0 {
		remaining = usage.SecondaryRemainingPercent
	}
	usage.LimitReached = remaining <= 0
	return usage, &remaining, true
}

func (v *Validator) queryZhipuAPIBalance(ctx context.Context, selected token.Token) (token.UsageInfo, *int, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, zhipuAPIBalanceURL, nil)
	if err != nil {
		return token.UsageInfo{}, nil, false
	}
	req.Header.Set("Accept", "application/json")
	secret, err := credentialSecret(selected)
	if err != nil {
		return token.UsageInfo{}, nil, false
	}
	req.Header.Set("Authorization", "Bearer "+secret)

	resp, err := v.clientForToken(selected).Do(req)
	if err != nil {
		return token.UsageInfo{}, nil, false
	}
	defer closeBody(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return token.UsageInfo{}, nil, false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return token.UsageInfo{}, nil, false
	}
	payload, err := decodeObject(body)
	if err != nil {
		return token.UsageInfo{}, nil, false
	}
	if boolFromAny(payload["success"], true) == false {
		return token.UsageInfo{}, nil, false
	}

	rows := zhipuBalanceRows(payload)
	total := 0.0
	packages := make([]token.BalancePackage, 0, len(rows))
	for _, row := range rows {
		balancePackage, ok := zhipuBalancePackage(row)
		if !ok {
			continue
		}
		packages = append(packages, balancePackage)
		if zhipuBalancePackageCountsAsTokens(balancePackage) {
			total += balancePackage.BalanceRemaining
		}
	}

	remaining := 100
	if total <= 0 {
		remaining = 0
	}
	return token.UsageInfo{
		Source:           token.ProviderZhipu,
		PlanType:         "Zhipu GLM API Key",
		BalanceRemaining: total,
		BalanceUnit:      "Token",
		BalancePackages:  packages,
		LimitReached:     total <= 0,
		Message:          "Zhipu GLM API balance",
	}, &remaining, true
}

func zhipuBalanceRows(payload map[string]any) []any {
	if rows, ok := payload["rows"].([]any); ok {
		return rows
	}
	if data, ok := payload["data"].(map[string]any); ok {
		if rows, ok := data["rows"].([]any); ok {
			return rows
		}
		if rows, ok := data["list"].([]any); ok {
			return rows
		}
	}
	return nil
}

func zhipuBalancePackage(row any) (token.BalancePackage, bool) {
	item, ok := row.(map[string]any)
	if !ok {
		return token.BalancePackage{}, false
	}
	balance, ok := zhipuBalanceAmount(item, "availableBalance", "tokenBalance", "balance", "remaining")
	if !ok {
		return token.BalancePackage{}, false
	}
	total, totalOK := zhipuBalanceAmount(item, "tokensMagnitude", "balanceTotal", "totalBalance", "tokenBalance")
	if !totalOK {
		total = balance
	}
	consumeType, _ := stringFromAny(item["consumeType"])
	unit := "Token"
	if strings.EqualFold(consumeType, "TIMES") {
		unit = "次"
	}
	name, _ := stringFromAny(item["resourcePackageName"])
	if name == "" {
		name, _ = stringFromAny(item["tokenNo"])
	}
	status, _ := stringFromAny(item["status"])
	expiration, _ := stringFromAny(item["packageExpirationTime"])
	if expiration == "" {
		expiration, _ = stringFromAny(item["expirationTime"])
	}
	suitableModel, _ := stringFromAny(item["suitableModel"])
	suitableScene, _ := stringFromAny(item["suitableScene"])
	return token.BalancePackage{
		Name:             name,
		ConsumeType:      consumeType,
		BalanceRemaining: balance,
		BalanceTotal:     total,
		Unit:             unit,
		Status:           status,
		ExpirationTime:   expiration,
		SuitableModel:    suitableModel,
		SuitableScene:    suitableScene,
	}, true
}

func zhipuBalancePackageCountsAsTokens(balancePackage token.BalancePackage) bool {
	if balancePackage.Status != "" && !strings.EqualFold(balancePackage.Status, "EFFECTIVE") {
		return false
	}
	return balancePackage.ConsumeType == "" || strings.EqualFold(balancePackage.ConsumeType, "TOKENS")
}

func zhipuBalanceAmount(item map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		if balance, ok := floatFromAny(item[key]); ok {
			return balance, true
		}
	}
	return 0, false
}

func (v *Validator) queryZhipuCodingUsage(ctx context.Context, selected token.Token) (token.UsageInfo, *int, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, zhipuCodingPlanUsageURL, nil)
	if err != nil {
		return token.UsageInfo{}, nil, false
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en-US,en")
	secret, err := credentialSecret(selected)
	if err != nil {
		return token.UsageInfo{}, nil, false
	}
	req.Header.Set("Authorization", secret)

	resp, err := v.clientForToken(selected).Do(req)
	if err != nil {
		return token.UsageInfo{}, nil, false
	}
	defer closeBody(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return token.UsageInfo{}, nil, false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return token.UsageInfo{}, nil, false
	}
	payload, err := decodeObject(body)
	if err != nil {
		return token.UsageInfo{}, nil, false
	}
	if boolFromAny(payload["success"], true) == false {
		return token.UsageInfo{}, nil, false
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		return token.UsageInfo{}, nil, false
	}

	usage := token.UsageInfo{
		Source:   token.ProviderZhipu,
		PlanType: "Zhipu GLM Token Plan",
		Message:  "Zhipu GLM coding usage",
	}
	if level, ok := stringFromAny(data["level"]); ok {
		usage.PlanType = "Zhipu GLM " + level
	}

	primaryFound := false
	secondaryFound := false
	if limits, ok := data["limits"].([]any); ok {
		for _, raw := range limits {
			limitItem, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			limitType, _ := stringFromAny(limitItem["type"])
			if limitType != "TOKENS_LIMIT" {
				continue
			}
			used, remaining, resetAt, ok := zhipuCodingUsageWindow(limitItem)
			if !ok {
				continue
			}
			if zhipuCodingLimitIsWeekly(limitItem) || primaryFound {
				usage.SecondaryUsedPercent = used
				usage.SecondaryRemainingPercent = remaining
				usage.SecondaryResetAt = resetAt
				secondaryFound = true
			} else {
				usage.PrimaryUsedPercent = used
				usage.PrimaryRemainingPercent = remaining
				usage.PrimaryResetAt = resetAt
				primaryFound = true
			}
			usage.SubscriptionQuotaAvailable = true
		}
	}
	if !usage.SubscriptionQuotaAvailable {
		return token.UsageInfo{}, nil, false
	}

	remaining := usage.PrimaryRemainingPercent
	if !primaryFound && secondaryFound {
		remaining = usage.SecondaryRemainingPercent
	} else if primaryFound && secondaryFound && usage.SecondaryRemainingPercent < remaining {
		remaining = usage.SecondaryRemainingPercent
	}
	usage.LimitReached = remaining <= 0
	return usage, &remaining, true
}

func zhipuCodingUsageWindow(limitItem map[string]any) (int, int, int64, bool) {
	if usedValue, ok := floatFromAny(limitItem["percentage"]); ok {
		used := percent(usedValue)
		return used, 100 - used, unixSecondsFromAny(limitItem["nextResetTime"]), true
	}
	total, totalOK := floatFromAny(limitItem["usage"])
	usedCount, usedOK := floatFromAny(limitItem["currentValue"])
	if !totalOK || !usedOK || total <= 0 {
		return 0, 0, 0, false
	}
	used := percent((usedCount / total) * 100)
	remaining := 100 - used
	if remaining < 0 {
		remaining = 0
	}
	return used, remaining, unixSecondsFromAny(limitItem["nextResetTime"]), true
}

func zhipuCodingLimitIsWeekly(limitItem map[string]any) bool {
	unit, unitOK := floatFromAny(limitItem["unit"])
	number, numberOK := floatFromAny(limitItem["number"])
	if unitOK && numberOK {
		if int(unit) == 6 && int(number) >= 7 {
			return true
		}
		if int(number) >= 7 {
			return true
		}
	}
	window, _ := stringFromAny(limitItem["name"])
	if strings.Contains(strings.ToLower(window), "week") || strings.Contains(window, "周") {
		return true
	}
	return false
}

func (v *Validator) queryMiniMaxCodingUsage(ctx context.Context, selected token.Token) (token.UsageInfo, *int, bool) {
	target, err := joinURLPath(v.cfg.MiniMaxBaseURL, "/api/openplatform/coding_plan/remains")
	if err != nil {
		return token.UsageInfo{}, nil, false
	}
	body, ok := v.queryJSON(ctx, selected, target)
	if !ok {
		return token.UsageInfo{}, nil, false
	}

	payload, err := decodeObject(body)
	if err != nil {
		return token.UsageInfo{}, nil, false
	}
	if baseResp, ok := payload["base_resp"].(map[string]any); ok {
		statusCode, _ := floatFromAny(baseResp["status_code"])
		if statusCode != 0 {
			return token.UsageInfo{}, nil, false
		}
	}

	modelRemains, ok := payload["model_remains"].([]any)
	if !ok || len(modelRemains) == 0 {
		return token.UsageInfo{}, nil, false
	}
	first, ok := modelRemains[0].(map[string]any)
	if !ok {
		return token.UsageInfo{}, nil, false
	}

	usage := token.UsageInfo{
		Source:   token.ProviderMiniMax,
		PlanType: "MiniMax Token Plan",
		Message:  "MiniMax coding usage",
	}
	if modelName, ok := stringFromAny(first["model"]); ok {
		usage.PlanType = "MiniMax " + modelName
	}

	primaryAvailable := false
	secondaryAvailable := false
	if used, remaining, resetAt, ok := minimaxUsageWindow(first, "current_interval_total_count", "current_interval_usage_count", "end_time"); ok {
		usage.PrimaryUsedPercent = used
		usage.PrimaryRemainingPercent = remaining
		usage.PrimaryResetAt = resetAt
		usage.SubscriptionQuotaAvailable = true
		primaryAvailable = true
	}
	if used, remaining, resetAt, ok := minimaxUsageWindow(first, "current_weekly_total_count", "current_weekly_usage_count", "weekly_end_time"); ok {
		usage.SecondaryUsedPercent = used
		usage.SecondaryRemainingPercent = remaining
		usage.SecondaryResetAt = resetAt
		usage.SubscriptionQuotaAvailable = true
		secondaryAvailable = true
	}
	if !usage.SubscriptionQuotaAvailable {
		return token.UsageInfo{}, nil, false
	}

	remaining := usage.PrimaryRemainingPercent
	if !primaryAvailable && secondaryAvailable {
		remaining = usage.SecondaryRemainingPercent
	}
	usage.LimitReached = remaining <= 0
	return usage, &remaining, true
}

func minimaxUsageWindow(value map[string]any, totalKey string, remainingKey string, resetKey string) (int, int, int64, bool) {
	total, ok := floatFromAny(value[totalKey])
	if !ok || total <= 0 {
		return 0, 0, 0, false
	}
	remainingValue, ok := floatFromAny(value[remainingKey])
	if !ok {
		return 0, 0, 0, false
	}
	used := percent(((total - remainingValue) / total) * 100)
	remaining := 100 - used
	if remaining < 0 {
		remaining = 0
	}
	return used, remaining, unixSecondsFromAny(value[resetKey]), true
}
