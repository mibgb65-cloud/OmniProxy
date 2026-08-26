package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"omniproxy/internal/history"
	"omniproxy/internal/logs"
	"omniproxy/internal/proxy"
	"omniproxy/internal/token"
	"strings"
	"time"
)

func (a *appServer) validateAndRecordToken(ctx context.Context, selected token.Token) (proxy.ValidationResult, error) {
	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()
	if isCodexToken(selected) && cfg.CodexAutoActivateUsage {
		a.codexActivationMu.Lock()
		defer a.codexActivationMu.Unlock()
	}

	refreshedSelected, _, refreshErr := a.refreshAuthTokenIfNeeded(ctx, selected, false)
	if refreshErr != nil {
		_ = a.tokens.MarkInvalid(selected.ID, fmt.Sprintf("OAuth token refresh failed: %v", refreshErr))
		return proxy.ValidationResult{}, refreshErr
	}
	selected = refreshedSelected

	validator, err := proxy.NewValidator(cfg)
	if err != nil {
		return proxy.ValidationResult{}, err
	}

	result, err := validator.Validate(ctx, selected)
	if err == nil && isRefreshableAuthToken(selected) && (result.Status == http.StatusUnauthorized || result.Status == http.StatusForbidden) {
		refreshedSelected, refreshed, refreshErr := a.refreshAuthTokenIfNeeded(ctx, selected, true)
		if refreshErr != nil {
			_ = a.tokens.MarkInvalid(selected.ID, fmt.Sprintf("OAuth token refresh failed: %v", refreshErr))
			return result, refreshErr
		}
		if refreshed {
			selected = refreshedSelected
			result, err = validator.Validate(ctx, selected)
		}
	}
	if err == nil && result.OK && cfg.CodexAutoActivateUsage && isCodexToken(selected) && result.Usage != nil && proxy.CodexUsageNeedsActivation(*result.Usage) {
		if a.logs != nil {
			a.logs.Add(logs.Entry{Level: logs.LevelInfo, TokenName: a.tokenDisplayName(selected), Message: "codex usage windows are inactive; automatic activation started"})
		}
		if activationErr := validator.ActivateCodexUsage(ctx, selected); activationErr != nil {
			err = fmt.Errorf("Codex 自动激活失败: %w", activationErr)
		} else {
			result, err = validator.Validate(ctx, selected)
			if err == nil && (!result.OK || result.Usage == nil || proxy.CodexUsageNeedsActivation(*result.Usage)) {
				err = errors.New("Codex 自动激活后仍未检测到完整的额度窗口")
			}
			if err == nil && a.logs != nil {
				a.logs.Add(logs.Entry{Level: logs.LevelInfo, TokenName: a.tokenDisplayName(selected), Message: "codex usage windows activated automatically"})
			}
		}
	}
	if result.Remaining != nil {
		_ = a.tokens.RecordUsage(selected.ID, *result.Remaining)
	}
	if result.Usage != nil {
		_ = a.tokens.RecordUsageInfo(selected.ID, *result.Usage)
	}
	if result.Status == http.StatusUnauthorized || result.Status == http.StatusForbidden {
		_ = a.tokens.MarkInvalid(selected.ID, fmt.Sprintf("validation returned %d", result.Status))
	}
	if result.Status == http.StatusTooManyRequests {
		_ = a.tokens.MarkExhaustedUntil(selected.ID, "validation returned 429", validationCooldownUntil(result))
	}
	if result.OK && result.Remaining == nil && result.Usage == nil {
		_ = a.tokens.MarkActive(selected.ID)
	}
	return result, err
}

func (a *appServer) refreshStoredAuthToken(ctx context.Context, id string) (token.Token, error) {
	selected, err := a.tokens.Get(id)
	if err != nil {
		return token.Token{}, err
	}
	if !isRefreshableAuthToken(selected) {
		return token.Token{}, errors.New("token credential does not support refresh")
	}

	updated, _, err := a.refreshAuthTokenIfNeeded(ctx, selected, true)
	if err != nil {
		_ = a.tokens.MarkInvalid(selected.ID, fmt.Sprintf("OAuth token refresh failed: %v", err))
		return token.Token{}, err
	}
	if a.logs != nil {
		a.logs.Add(logs.Entry{Level: logs.LevelInfo, TokenName: a.tokenDisplayName(updated), Message: "manual OAuth token refresh completed"})
	}
	return updated, nil
}

type parsedAPIKeyLine struct {
	line  int
	value string
}

func (a *appServer) importAPIKeys(req apiKeyBatchImportRequest) (apiKeyBatchImportResult, error) {
	provider, credentialType, err := token.NormalizeProviderAndCredential(req.Provider, token.CredentialTypeAPIKey)
	if err != nil {
		return apiKeyBatchImportResult{}, err
	}
	if credentialType != token.CredentialTypeAPIKey {
		return apiKeyBatchImportResult{}, errors.New("批量导入仅支持 API Key")
	}
	region, err := token.NormalizeRegion(provider, credentialType, req.Region)
	if err != nil {
		return apiKeyBatchImportResult{}, err
	}
	baseURL, err := token.NormalizeBaseURL(provider, req.BaseURL, true)
	if err != nil {
		return apiKeyBatchImportResult{}, err
	}

	lines := parseAPIKeyBatchLines(req.TokenText)
	if len(lines) == 0 {
		return apiKeyBatchImportResult{}, errors.New("未找到可导入的 API Key")
	}

	usedNames := map[string]bool{}
	existingKeys := map[string]bool{}
	for _, item := range a.tokens.List() {
		if token.NormalizeProvider(item.Provider) != provider || item.CredentialType != credentialType {
			continue
		}
		usedNames[strings.ToLower(strings.TrimSpace(item.Name))] = true
		if strings.TrimSpace(item.BaseURL) == baseURL {
			existingKeys[strings.TrimSpace(item.TokenValue)] = true
		}
	}

	result := apiKeyBatchImportResult{
		Skipped: []apiKeyBatchImportSkipped{},
	}
	seenKeys := map[string]bool{}
	for _, line := range lines {
		if seenKeys[line.value] {
			result.Skipped = append(result.Skipped, apiKeyBatchImportSkipped{Line: line.line, Reason: "本次导入中重复"})
			continue
		}
		seenKeys[line.value] = true
		if existingKeys[line.value] {
			result.Skipped = append(result.Skipped, apiKeyBatchImportSkipped{Line: line.line, Reason: "账号池中已存在"})
			continue
		}

		name := uniqueAPIKeyImportName(line.value, usedNames)
		if _, err := a.tokens.Add(token.UpsertRequest{
			Name:           name,
			Provider:       provider,
			CredentialType: credentialType,
			Region:         region,
			BaseURL:        baseURL,
			TokenValue:     line.value,
		}); err != nil {
			delete(usedNames, strings.ToLower(name))
			result.Skipped = append(result.Skipped, apiKeyBatchImportSkipped{Line: line.line, Reason: err.Error()})
			continue
		}
		existingKeys[line.value] = true
		result.CreatedCount++
	}

	if a.logs != nil && result.CreatedCount > 0 {
		a.logs.Add(logs.Entry{
			Level:   logs.LevelInfo,
			Message: fmt.Sprintf("%s batch imported API keys: %d created, %d skipped", provider, result.CreatedCount, len(result.Skipped)),
		})
	}
	return result, nil
}

func parseAPIKeyBatchLines(text string) []parsedAPIKeyLine {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")
	out := []parsedAPIKeyLine{}
	for index, raw := range lines {
		line := strings.TrimSpace(raw)
		if before, _, ok := strings.Cut(line, "#"); ok {
			line = strings.TrimSpace(before)
		}
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		out = append(out, parsedAPIKeyLine{line: index + 1, value: strings.TrimSpace(fields[0])})
	}
	return out
}

func uniqueAPIKeyImportName(value string, used map[string]bool) string {
	base := apiKeyImportName(value)
	name := base
	for suffix := 2; used[strings.ToLower(name)]; suffix++ {
		name = fmt.Sprintf("%s-%d", base, suffix)
	}
	used[strings.ToLower(name)] = true
	return name
}

func apiKeyImportName(value string) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) == 0 {
		return "api-key"
	}
	if len(runes) > 8 {
		runes = runes[:8]
	}
	return string(runes)
}

func (a *appServer) recordTokenMaintenanceHistory(event string, selected token.Token, result proxy.ValidationResult, err error) {
	a.mu.Lock()
	recorder := a.history
	a.mu.Unlock()
	if recorder == nil {
		return
	}
	if a.tokens != nil && strings.TrimSpace(selected.ID) != "" {
		if latest, latestErr := a.tokens.Get(selected.ID); latestErr == nil {
			selected = latest
		}
	}

	path, protocol, label := tokenMaintenanceEventMeta(event)
	level := logs.LevelInfo
	if err != nil || !result.OK {
		level = logs.LevelWarn
	}
	recorder.Add(history.Entry{
		Level:     string(level),
		Method:    "CHECK",
		Path:      path,
		Provider:  token.NormalizeProvider(selected.Provider),
		Protocol:  protocol,
		Model:     selected.CredentialType,
		Status:    result.Status,
		Duration:  result.Duration,
		TokenID:   selected.ID,
		TokenName: token.DisplayName(selected),
		Message:   tokenMaintenanceHistoryMessage(label, result, err),
	})
}

func tokenMaintenanceEventMeta(event string) (string, string, string) {
	switch event {
	case historyEventCodexRefreshAdd:
		return "/maintenance/codex-usage-refresh", "quota-refresh", "codex usage refresh after add completed"
	case historyEventStartupCodexUsage:
		return "/maintenance/startup-codex-usage-refresh", "quota-refresh", "startup codex quota refresh completed"
	case historyEventCurrentQuota:
		return "/maintenance/current-token-quota-refresh", "quota-refresh", "current token quota refresh completed"
	case historyEventHealthCheck:
		return "/maintenance/token-health-check", "health-check", "token health check completed"
	default:
		return "/maintenance/token-validation", "token-validation", "manual token validation completed"
	}
}

func tokenMaintenanceHistoryMessage(label string, result proxy.ValidationResult, err error) string {
	parts := []string{label}
	if err != nil {
		parts = append(parts, err.Error())
	} else if strings.TrimSpace(result.Message) != "" {
		parts = append(parts, result.Message)
	}
	if result.Remaining != nil {
		parts = append(parts, fmt.Sprintf("remaining=%d%%", *result.Remaining))
	}
	if result.Usage != nil && result.Usage.SubscriptionQuotaAvailable {
		parts = append(parts, fmt.Sprintf("primary=%d%%", result.Usage.PrimaryRemainingPercent))
		parts = append(parts, fmt.Sprintf("secondary=%d%%", result.Usage.SecondaryRemainingPercent))
	}
	return strings.Join(parts, " · ")
}

func authTokenNeedsRefresh(selected token.Token, now time.Time) bool {
	switch {
	case isCodexToken(selected):
		return proxy.CodexAuthNeedsRefresh(selected.TokenValue, now)
	case isClaudeOAuthToken(selected):
		return proxy.ClaudeOAuthNeedsRefresh(selected.TokenValue, now)
	default:
		return false
	}
}

func (a *appServer) refreshAuthTokenIfNeeded(ctx context.Context, selected token.Token, force bool) (token.Token, bool, error) {
	if !isRefreshableAuthToken(selected) {
		return selected, false, nil
	}

	// Expiry is decided by parsing alone, so it is checked before the refresh
	// lock and before a client is built. Otherwise every Codex and Claude
	// request funnels through one global mutex that a single in-flight refresh
	// can hold for the whole request timeout.
	if !force && !authTokenNeedsRefresh(selected, time.Now()) {
		return selected, false, nil
	}

	a.codexRefreshMu.Lock()
	defer a.codexRefreshMu.Unlock()

	if latest, err := a.tokens.Get(selected.ID); err == nil {
		selected = latest
	}
	// Another request may have refreshed this token while we waited for the lock.
	if !force && !authTokenNeedsRefresh(selected, time.Now()) {
		return selected, false, nil
	}

	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()
	client, err := proxy.NewTokenHTTPClient(cfg, selected, healthRequestTimeout)
	if err != nil {
		return selected, false, err
	}
	var updatedValue string
	var refreshed bool
	switch {
	case isCodexToken(selected):
		updatedValue, refreshed, err = proxy.RefreshCodexAuthJSON(ctx, client, selected.TokenValue, force, time.Now())
	case isClaudeOAuthToken(selected):
		updatedValue, refreshed, err = proxy.RefreshClaudeOAuthJSON(ctx, client, selected.TokenValue, force, time.Now())
	default:
		return selected, false, nil
	}
	if err != nil || !refreshed {
		return selected, refreshed, err
	}

	updated, err := a.tokens.UpdateTokenValue(selected.ID, updatedValue)
	if err != nil {
		return selected, true, err
	}
	if a.logs != nil {
		a.logs.Add(logs.Entry{Level: logs.LevelInfo, TokenName: a.tokenDisplayName(updated), Message: "OAuth access token refreshed"})
	}
	return updated, true, nil
}

func (a *appServer) refreshCodexUsageOnStartup(ctx context.Context) {
	items := a.tokens.List()
	total := 0
	failed := 0
	for _, item := range items {
		if item.Disabled {
			continue
		}
		if !isCodexToken(item) {
			continue
		}
		total++
		result, err := a.validateAndRecordToken(ctx, item)
		a.recordTokenMaintenanceHistory(historyEventStartupCodexUsage, item, result, err)
		if err != nil {
			failed++
			a.logs.Add(logs.Entry{Level: logs.LevelWarn, TokenName: a.tokenDisplayName(item), Message: fmt.Sprintf("startup codex usage refresh failed: %v", err)})
		}
	}
	if total == 0 {
		return
	}
	message := fmt.Sprintf("startup codex usage refresh completed: %d accounts", total)
	level := logs.LevelInfo
	if failed > 0 {
		level = logs.LevelWarn
		message = fmt.Sprintf("startup codex usage refresh completed: %d accounts, %d failed", total, failed)
	}
	a.logs.Add(logs.Entry{Level: level, Message: message})
}

func (a *appServer) startHealthMonitor() {
	a.mu.Lock()
	if a.healthStop != nil || a.tokens == nil {
		a.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.healthStop = cancel
	a.mu.Unlock()

	go a.healthMonitor(ctx)
}

func (a *appServer) stopHealthMonitor() {
	a.mu.Lock()
	cancel := a.healthStop
	a.healthStop = nil
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (a *appServer) healthMonitor(ctx context.Context) {
	healthTimer := time.NewTimer(30 * time.Second)
	defer healthTimer.Stop()
	currentQuotaTimer := time.NewTimer(currentQuotaInterval)
	defer currentQuotaTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-healthTimer.C:
			a.runDueHealthChecks(ctx)
			healthTimer.Reset(healthCheckTick)
		case <-currentQuotaTimer.C:
			a.refreshCurrentTokenUsage(ctx)
			currentQuotaTimer.Reset(currentQuotaInterval)
		}
	}
}

func (a *appServer) refreshCurrentTokenUsage(ctx context.Context) {
	a.mu.Lock()
	manager := a.tokens
	a.mu.Unlock()
	if manager == nil {
		return
	}

	selected, ok := currentQuotaRefreshCandidate(manager.List(), time.Now())
	if !ok {
		return
	}

	checkCtx, cancel := context.WithTimeout(ctx, healthRequestTimeout)
	result, err := a.validateAndRecordToken(checkCtx, selected)
	cancel()
	a.recordTokenMaintenanceHistory(historyEventCurrentQuota, selected, result, err)

	message := result.Message
	if err != nil {
		message = err.Error()
	}
	_ = manager.RecordHealthCheck(selected.ID, result.OK, result.Status, message, nextHealthCheckAt(result, err))

	if err != nil || !result.OK {
		a.logs.Add(logs.Entry{
			Level:     logs.LevelWarn,
			Status:    result.Status,
			Duration:  result.Duration,
			TokenName: a.tokenDisplayName(selected),
			Message:   fmt.Sprintf("current token quota refresh failed: %v", message),
		})
	}
}

func currentQuotaRefreshCandidate(items []token.Token, now time.Time) (token.Token, bool) {
	var selected token.Token
	found := false
	for _, item := range items {
		if item.Disabled {
			continue
		}
		if strings.TrimSpace(item.TokenValue) == "" || item.Stats.UpdatedAt == nil {
			continue
		}
		if item.Status == token.StatusInvalid {
			continue
		}
		if item.CooldownUntil != nil && now.Before(*item.CooldownUntil) {
			continue
		}
		if item.Health.NextCheckAt != nil && now.Before(*item.Health.NextCheckAt) {
			continue
		}
		if !found || item.Stats.UpdatedAt.After(*selected.Stats.UpdatedAt) {
			selected = item
			found = true
		}
	}
	return selected, found
}

func (a *appServer) runDueHealthChecks(ctx context.Context) {
	a.mu.Lock()
	manager := a.tokens
	a.mu.Unlock()
	if manager == nil {
		return
	}

	candidates := manager.HealthCheckCandidates(time.Now(), activeHealthInterval, retryHealthInterval)
	if len(candidates) == 0 {
		return
	}

	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: fmt.Sprintf("health check started: %d accounts", len(candidates))})
	for _, item := range candidates {
		select {
		case <-ctx.Done():
			return
		default:
		}

		checkCtx, cancel := context.WithTimeout(ctx, healthRequestTimeout)
		result, err := a.validateAndRecordToken(checkCtx, item)
		cancel()
		a.recordTokenMaintenanceHistory(historyEventHealthCheck, item, result, err)

		message := result.Message
		if err != nil {
			message = err.Error()
		}
		nextCheck := nextHealthCheckAt(result, err)
		_ = manager.RecordHealthCheck(item.ID, result.OK, result.Status, message, nextCheck)

		level := logs.LevelInfo
		if err != nil || !result.OK {
			level = logs.LevelWarn
		}
		a.logs.Add(logs.Entry{
			Level:     level,
			Status:    result.Status,
			Duration:  result.Duration,
			TokenName: a.tokenDisplayName(item),
			Message:   "health check completed",
		})
	}
}

func nextHealthCheckAt(result proxy.ValidationResult, err error) *time.Time {
	if result.Status == http.StatusTooManyRequests {
		return validationCooldownUntil(result)
	}
	now := time.Now()
	wait := activeHealthInterval
	if err != nil || !result.OK {
		wait = failedHealthRetryWait
	}
	next := now.Add(wait)
	return &next
}

func isCodexToken(item token.Token) bool {
	return token.NormalizeProvider(item.Provider) == token.ProviderOpenAI &&
		item.CredentialType == token.CredentialTypeCodexAuthJSON
}

func isClaudeOAuthToken(item token.Token) bool {
	return token.NormalizeProvider(item.Provider) == token.ProviderAnthropic &&
		item.CredentialType == token.CredentialTypeClaudeOAuth
}

func isRefreshableAuthToken(item token.Token) bool {
	return isCodexToken(item) || isClaudeOAuthToken(item)
}

func validationCooldownUntil(result proxy.ValidationResult) *time.Time {
	now := time.Now()
	if result.Usage != nil && result.Usage.PrimaryResetAt > now.Unix() {
		until := time.Unix(result.Usage.PrimaryResetAt, 0)
		return &until
	}
	until := now.Add(5 * time.Minute)
	return &until
}
