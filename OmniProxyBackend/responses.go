package main

import (
	"fmt"
	"time"

	"omniproxy/internal/history"
	"omniproxy/internal/logs"
	"omniproxy/internal/proxy"
	"omniproxy/internal/token"
)

type tokenResponse struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	Provider         string             `json:"provider"`
	CredentialType   string             `json:"credentialType"`
	AccountID        string             `json:"accountId,omitempty"`
	Region           string             `json:"region,omitempty"`
	BaseURL          string             `json:"baseUrl,omitempty"`
	HasTokenValue    bool               `json:"hasTokenValue"`
	MaskedTokenValue string             `json:"maskedTokenValue,omitempty"`
	Remaining        int                `json:"remaining"`
	Usage            usageResponse      `json:"usage"`
	Stats            tokenStatsResponse `json:"stats"`
	Health           healthResponse     `json:"health"`
	HealthScore      int                `json:"healthScore"`
	HealthLevel      string             `json:"healthLevel,omitempty"`
	HealthMessage    string             `json:"healthMessage,omitempty"`
	Status           token.Status       `json:"status"`
	Disabled         bool               `json:"disabled"`
	Selected         bool               `json:"selected"`
	LastUsedAt       string             `json:"lastUsedAt,omitempty"`
	LastError        string             `json:"lastError,omitempty"`
	CooldownUntil    string             `json:"cooldownUntil,omitempty"`
	CreatedAt        string             `json:"createdAt"`
	UpdatedAt        string             `json:"updatedAt"`
}

type apiKeyBatchImportRequest struct {
	Provider       string `json:"provider"`
	CredentialType string `json:"credentialType"`
	Region         string `json:"region,omitempty"`
	BaseURL        string `json:"baseUrl,omitempty"`
	TokenText      string `json:"tokenText"`
}

type apiKeyBatchImportSkipped struct {
	Line   int    `json:"line"`
	Reason string `json:"reason"`
}

type apiKeyBatchImportResult struct {
	CreatedCount int                        `json:"createdCount"`
	Skipped      []apiKeyBatchImportSkipped `json:"skipped"`
}

type usageResponse struct {
	Source                         string                     `json:"source,omitempty"`
	PlanType                       string                     `json:"planType,omitempty"`
	LimitReached                   bool                       `json:"limitReached,omitempty"`
	PrimaryUsedPercent             int                        `json:"primaryUsedPercent"`
	PrimaryUsedPercentExact        float64                    `json:"primaryUsedPercentExact,omitempty"`
	PrimaryRemainingPercent        int                        `json:"primaryRemainingPercent"`
	PrimaryResetAt                 int64                      `json:"primaryResetAt,omitempty"`
	PrimaryActivationPending       bool                       `json:"primaryWindowPendingActivation,omitempty"`
	SecondaryUsedPercent           int                        `json:"secondaryUsedPercent"`
	SecondaryUsedPercentExact      float64                    `json:"secondaryUsedPercentExact,omitempty"`
	SecondaryRemainingPercent      int                        `json:"secondaryRemainingPercent"`
	SecondaryResetAt               int64                      `json:"secondaryResetAt,omitempty"`
	SecondaryActivationPending     bool                       `json:"secondaryWindowPendingActivation,omitempty"`
	APIRemaining                   int                        `json:"apiRemaining,omitempty"`
	BalanceRemaining               float64                    `json:"balanceRemaining,omitempty"`
	BalanceTotal                   float64                    `json:"balanceTotal,omitempty"`
	BalanceUsed                    float64                    `json:"balanceUsed,omitempty"`
	BalanceUnit                    string                     `json:"balanceUnit,omitempty"`
	BalanceUnlimited               bool                       `json:"balanceUnlimited,omitempty"`
	BalancePackages                []balancePackageResponse   `json:"balancePackages,omitempty"`
	CodexResetCreditsAvailable     *int                       `json:"codexResetCreditsAvailable,omitempty"`
	CodexResetCredits              []codexResetCreditResponse `json:"codexResetCredits,omitempty"`
	CodexResetCreditsNextExpiresAt int64                      `json:"codexResetCreditsNextExpiresAt,omitempty"`
	CodexResetCreditsError         string                     `json:"codexResetCreditsError,omitempty"`
	SubscriptionQuotaAvailable     bool                       `json:"subscriptionQuotaAvailable,omitempty"`
	Message                        string                     `json:"message,omitempty"`
	UpdatedAt                      string                     `json:"updatedAt,omitempty"`
}

type codexResetCreditResponse struct {
	ID         string `json:"id,omitempty"`
	Status     string `json:"status,omitempty"`
	ResetType  string `json:"resetType,omitempty"`
	GrantedAt  int64  `json:"grantedAt,omitempty"`
	ExpiresAt  int64  `json:"expiresAt,omitempty"`
	RedeemedAt int64  `json:"redeemedAt,omitempty"`
	RawStatus  string `json:"rawStatus,omitempty"`
}

type balancePackageResponse struct {
	Name             string  `json:"name,omitempty"`
	ConsumeType      string  `json:"consumeType,omitempty"`
	BalanceRemaining float64 `json:"balanceRemaining,omitempty"`
	BalanceTotal     float64 `json:"balanceTotal,omitempty"`
	Unit             string  `json:"unit,omitempty"`
	Status           string  `json:"status,omitempty"`
	ExpirationTime   string  `json:"expirationTime,omitempty"`
	SuitableModel    string  `json:"suitableModel,omitempty"`
	SuitableScene    string  `json:"suitableScene,omitempty"`
}

type tokenStatsResponse struct {
	RequestCount                int64                              `json:"requestCount"`
	InputTokens                 int64                              `json:"inputTokens"`
	OutputTokens                int64                              `json:"outputTokens"`
	TotalTokens                 int64                              `json:"totalTokens"`
	CacheCreationTokens         int64                              `json:"cacheCreationTokens,omitempty"`
	CacheReadTokens             int64                              `json:"cacheReadTokens,omitempty"`
	CodexWeeklyEstimateResetAt  int64                              `json:"codexWeeklyEstimateResetAt,omitempty"`
	CodexWeeklyEstimateBaseline *token.CodexWeeklyEstimateBaseline `json:"codexWeeklyEstimateBaseline,omitempty"`
	LastInputTokens             int                                `json:"lastInputTokens,omitempty"`
	LastOutputTokens            int                                `json:"lastOutputTokens,omitempty"`
	LastTotalTokens             int                                `json:"lastTotalTokens,omitempty"`
	LastCacheCreationTokens     int                                `json:"lastCacheCreationTokens,omitempty"`
	LastCacheReadTokens         int                                `json:"lastCacheReadTokens,omitempty"`
	Daily                       []token.DailyTokenUsage            `json:"daily,omitempty"`
	UpdatedAt                   string                             `json:"updatedAt,omitempty"`
}

type healthResponse struct {
	LastCheckedAt     string `json:"lastCheckedAt,omitempty"`
	NextCheckAt       string `json:"nextCheckAt,omitempty"`
	ConsecutiveErrors int    `json:"consecutiveErrors,omitempty"`
	LastStatus        int    `json:"lastStatus,omitempty"`
	LastMessage       string `json:"lastMessage,omitempty"`
}

type validationResponse struct {
	OK          bool           `json:"ok"`
	Status      int            `json:"status"`
	Duration    int64          `json:"durationMs"`
	Remaining   *int           `json:"remaining,omitempty"`
	Usage       *usageResponse `json:"usage,omitempty"`
	Message     string         `json:"message"`
	CheckedPath string         `json:"checkedPath"`
}

type codexUsageActivationResponse struct {
	Token                  tokenResponse `json:"token"`
	Activated              bool          `json:"activated"`
	AlreadyActive          bool          `json:"alreadyActive"`
	BeforePrimaryResetAt   int64         `json:"beforePrimaryResetAt,omitempty"`
	BeforeSecondaryResetAt int64         `json:"beforeSecondaryResetAt,omitempty"`
	AfterPrimaryResetAt    int64         `json:"afterPrimaryResetAt,omitempty"`
	AfterSecondaryResetAt  int64         `json:"afterSecondaryResetAt,omitempty"`
	Message                string        `json:"message"`
}

type logResponse struct {
	ID         int64      `json:"id"`
	Time       string     `json:"time"`
	Level      logs.Level `json:"level"`
	Method     string     `json:"method,omitempty"`
	Path       string     `json:"path,omitempty"`
	Model      string     `json:"model,omitempty"`
	ClientKey  string     `json:"clientKey,omitempty"`
	ClientName string     `json:"clientName,omitempty"`
	Status     int        `json:"status,omitempty"`
	Duration   int64      `json:"durationMs,omitempty"`
	TokenName  string     `json:"tokenName,omitempty"`
	Message    string     `json:"message"`
}

type historyResponse struct {
	ID                int64                  `json:"id"`
	Time              string                 `json:"time"`
	Level             string                 `json:"level"`
	Method            string                 `json:"method,omitempty"`
	Path              string                 `json:"path,omitempty"`
	Provider          string                 `json:"provider,omitempty"`
	Protocol          string                 `json:"protocol,omitempty"`
	ClientKey         string                 `json:"clientKey,omitempty"`
	ClientName        string                 `json:"clientName,omitempty"`
	Model             string                 `json:"model,omitempty"`
	Status            int                    `json:"status,omitempty"`
	Duration          int64                  `json:"durationMs,omitempty"`
	TokenID           string                 `json:"tokenId,omitempty"`
	TokenName         string                 `json:"tokenName,omitempty"`
	InputTokens       int                    `json:"inputTokens,omitempty"`
	OutputTokens      int                    `json:"outputTokens,omitempty"`
	TotalTokens       int                    `json:"totalTokens,omitempty"`
	CooldownTriggered bool                   `json:"cooldownTriggered,omitempty"`
	RetryChain        []retryAttemptResponse `json:"retryChain,omitempty"`
	Message           string                 `json:"message"`
}

type activeRequestResponse struct {
	ID         int64  `json:"id"`
	StartedAt  string `json:"startedAt"`
	ClientKey  string `json:"clientKey,omitempty"`
	ClientName string `json:"clientName,omitempty"`
	Method     string `json:"method,omitempty"`
	Path       string `json:"path,omitempty"`
	Provider   string `json:"provider,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	Model      string `json:"model,omitempty"`
	TokenID    string `json:"tokenId,omitempty"`
	TokenName  string `json:"tokenName,omitempty"`
}

type retryAttemptResponse struct {
	Attempt           int    `json:"attempt"`
	Provider          string `json:"provider,omitempty"`
	Protocol          string `json:"protocol,omitempty"`
	Model             string `json:"model,omitempty"`
	Status            int    `json:"status,omitempty"`
	Duration          int64  `json:"durationMs,omitempty"`
	TokenID           string `json:"tokenId,omitempty"`
	TokenName         string `json:"tokenName,omitempty"`
	CooldownTriggered bool   `json:"cooldownTriggered,omitempty"`
	Message           string `json:"message,omitempty"`
}

func tokenResponses(items []token.Token) []tokenResponse {
	out := make([]tokenResponse, len(items))
	for i, item := range items {
		out[i] = tokenResponseFor(item)
	}
	return out
}

func tokenResponseFor(item token.Token) tokenResponse {
	return tokenResponse{
		ID:               item.ID,
		Name:             item.Name,
		Provider:         item.Provider,
		CredentialType:   item.CredentialType,
		AccountID:        tokenAccountID(item),
		Region:           item.Region,
		BaseURL:          item.BaseURL,
		HasTokenValue:    item.TokenValue != "",
		MaskedTokenValue: maskedTokenValue(item),
		Remaining:        item.Remaining,
		Usage:            usageResponseFor(item.Usage),
		Stats:            tokenStatsResponseFor(item.Stats),
		Health:           healthResponseFor(item.Health),
		HealthScore:      tokenHealthScore(item),
		HealthLevel:      tokenHealthLevel(item),
		HealthMessage:    tokenHealthMessage(item),
		Status:           item.Status,
		Disabled:         item.Disabled,
		Selected:         item.Selected,
		LastUsedAt:       timePtrString(item.LastUsedAt),
		LastError:        item.LastError,
		CooldownUntil:    timePtrString(item.CooldownUntil),
		CreatedAt:        timeString(item.CreatedAt),
		UpdatedAt:        timeString(item.UpdatedAt),
	}
}

func tokenAccountID(item token.Token) string {
	return token.CodexAccountID(item)
}

func tokenHealthScore(item token.Token) int {
	if item.Disabled {
		return 0
	}
	switch item.Status {
	case token.StatusInvalid:
		return 10
	case token.StatusExhausted:
		return 20
	}
	score := 100
	if item.Status == token.StatusLow {
		score -= 30
	}
	if item.CooldownUntil != nil && time.Now().Before(*item.CooldownUntil) {
		score -= 45
	}
	if item.Health.ConsecutiveErrors > 0 {
		score -= item.Health.ConsecutiveErrors * 15
	}
	if item.LastError != "" {
		score -= 15
	}
	if item.Remaining > 0 && item.Remaining <= 15 {
		score -= 15
	}
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func tokenHealthLevel(item token.Token) string {
	score := tokenHealthScore(item)
	switch {
	case item.Disabled:
		return "disabled"
	case score >= 80:
		return "healthy"
	case score >= 50:
		return "watch"
	default:
		return "risk"
	}
}

func tokenHealthMessage(item token.Token) string {
	if item.Disabled {
		return "已停用，不参与调度"
	}
	if item.CooldownUntil != nil && time.Now().Before(*item.CooldownUntil) {
		return "冷却中，等待自动复检"
	}
	switch item.Status {
	case token.StatusInvalid:
		return "凭据无效或刷新失败"
	case token.StatusExhausted:
		return "额度已耗尽"
	case token.StatusLow:
		return "额度偏低，建议准备备用账号"
	}
	if item.Health.ConsecutiveErrors > 0 {
		return fmt.Sprintf("连续健康检查失败 %d 次", item.Health.ConsecutiveErrors)
	}
	if item.Health.LastCheckedAt != nil {
		return "近期健康检查正常"
	}
	return "等待健康检查"
}

func usageResponseFor(usage token.UsageInfo) usageResponse {
	return usageResponse{
		Source:                         usage.Source,
		PlanType:                       usage.PlanType,
		LimitReached:                   usage.LimitReached,
		PrimaryUsedPercent:             usage.PrimaryUsedPercent,
		PrimaryUsedPercentExact:        usage.PrimaryUsedPercentExact,
		PrimaryRemainingPercent:        usage.PrimaryRemainingPercent,
		PrimaryResetAt:                 usage.PrimaryResetAt,
		PrimaryActivationPending:       usage.PrimaryActivationPending,
		SecondaryUsedPercent:           usage.SecondaryUsedPercent,
		SecondaryUsedPercentExact:      usage.SecondaryUsedPercentExact,
		SecondaryRemainingPercent:      usage.SecondaryRemainingPercent,
		SecondaryResetAt:               usage.SecondaryResetAt,
		SecondaryActivationPending:     usage.SecondaryActivationPending,
		APIRemaining:                   usage.APIRemaining,
		BalanceRemaining:               usage.BalanceRemaining,
		BalanceTotal:                   usage.BalanceTotal,
		BalanceUsed:                    usage.BalanceUsed,
		BalanceUnit:                    usage.BalanceUnit,
		BalanceUnlimited:               usage.BalanceUnlimited,
		BalancePackages:                balancePackageResponses(usage.BalancePackages),
		CodexResetCreditsAvailable:     usage.CodexResetCreditsAvailable,
		CodexResetCredits:              codexResetCreditResponses(usage.CodexResetCredits),
		CodexResetCreditsNextExpiresAt: usage.CodexResetCreditsNextExpiresAt,
		CodexResetCreditsError:         usage.CodexResetCreditsError,
		SubscriptionQuotaAvailable:     usage.SubscriptionQuotaAvailable,
		Message:                        usage.Message,
		UpdatedAt:                      timePtrString(usage.UpdatedAt),
	}
}

func codexResetCreditResponses(items []token.CodexResetCredit) []codexResetCreditResponse {
	if len(items) == 0 {
		return nil
	}
	out := make([]codexResetCreditResponse, len(items))
	for i, item := range items {
		out[i] = codexResetCreditResponse{
			ID:         item.ID,
			Status:     item.Status,
			ResetType:  item.ResetType,
			GrantedAt:  item.GrantedAt,
			ExpiresAt:  item.ExpiresAt,
			RedeemedAt: item.RedeemedAt,
			RawStatus:  item.RawStatus,
		}
	}
	return out
}

func balancePackageResponses(items []token.BalancePackage) []balancePackageResponse {
	if len(items) == 0 {
		return nil
	}
	out := make([]balancePackageResponse, len(items))
	for i, item := range items {
		out[i] = balancePackageResponse{
			Name:             item.Name,
			ConsumeType:      item.ConsumeType,
			BalanceRemaining: item.BalanceRemaining,
			BalanceTotal:     item.BalanceTotal,
			Unit:             item.Unit,
			Status:           item.Status,
			ExpirationTime:   item.ExpirationTime,
			SuitableModel:    item.SuitableModel,
			SuitableScene:    item.SuitableScene,
		}
	}
	return out
}

func tokenStatsResponseFor(stats token.TokenStats) tokenStatsResponse {
	return tokenStatsResponse{
		RequestCount:                stats.RequestCount,
		InputTokens:                 stats.InputTokens,
		OutputTokens:                stats.OutputTokens,
		TotalTokens:                 stats.TotalTokens,
		CacheCreationTokens:         stats.CacheCreationTokens,
		CacheReadTokens:             stats.CacheReadTokens,
		CodexWeeklyEstimateResetAt:  stats.CodexWeeklyEstimateResetAt,
		CodexWeeklyEstimateBaseline: stats.CodexWeeklyEstimateBaseline,
		LastInputTokens:             stats.LastInputTokens,
		LastOutputTokens:            stats.LastOutputTokens,
		LastTotalTokens:             stats.LastTotalTokens,
		LastCacheCreationTokens:     stats.LastCacheCreationTokens,
		LastCacheReadTokens:         stats.LastCacheReadTokens,
		Daily:                       append([]token.DailyTokenUsage(nil), stats.Daily...),
		UpdatedAt:                   timePtrString(stats.UpdatedAt),
	}
}

func healthResponseFor(health token.HealthInfo) healthResponse {
	return healthResponse{
		LastCheckedAt:     timePtrString(health.LastCheckedAt),
		NextCheckAt:       timePtrString(health.NextCheckAt),
		ConsecutiveErrors: health.ConsecutiveErrors,
		LastStatus:        health.LastStatus,
		LastMessage:       health.LastMessage,
	}
}

func validationResponseFor(result proxy.ValidationResult) validationResponse {
	out := validationResponse{
		OK:          result.OK,
		Status:      result.Status,
		Duration:    result.Duration,
		Remaining:   result.Remaining,
		Message:     result.Message,
		CheckedPath: result.CheckedPath,
	}
	if result.Usage != nil {
		usage := usageResponseFor(*result.Usage)
		out.Usage = &usage
	}
	return out
}

func logResponses(entries []logs.Entry) []logResponse {
	out := make([]logResponse, len(entries))
	for i, entry := range entries {
		out[i] = logResponseFor(entry)
	}
	return out
}

func logResponseFor(entry logs.Entry) logResponse {
	return logResponse{
		ID:         entry.ID,
		Time:       timeString(entry.Time),
		Level:      entry.Level,
		Method:     entry.Method,
		Path:       entry.Path,
		Model:      entry.Model,
		ClientKey:  entry.ClientKey,
		ClientName: entry.ClientName,
		Status:     entry.Status,
		Duration:   entry.Duration,
		TokenName:  entry.TokenName,
		Message:    entry.Message,
	}
}

func historyResponses(entries []history.Entry) []historyResponse {
	out := make([]historyResponse, len(entries))
	for i, entry := range entries {
		out[i] = historyResponseFor(entry)
	}
	return out
}

func historyResponseFor(entry history.Entry) historyResponse {
	return historyResponse{
		ID:                entry.ID,
		Time:              timeString(entry.Time),
		Level:             entry.Level,
		Method:            entry.Method,
		Path:              entry.Path,
		Provider:          entry.Provider,
		Protocol:          entry.Protocol,
		ClientKey:         entry.ClientKey,
		ClientName:        entry.ClientName,
		Model:             entry.Model,
		Status:            entry.Status,
		Duration:          entry.Duration,
		TokenID:           entry.TokenID,
		TokenName:         entry.TokenName,
		InputTokens:       entry.InputTokens,
		OutputTokens:      entry.OutputTokens,
		TotalTokens:       entry.TotalTokens,
		CooldownTriggered: entry.CooldownTriggered,
		RetryChain:        retryAttemptResponses(entry.RetryChain),
		Message:           entry.Message,
	}
}

func activeRequestResponses(entries []proxy.ActiveRequest) []activeRequestResponse {
	out := make([]activeRequestResponse, len(entries))
	for i, entry := range entries {
		out[i] = activeRequestResponse{
			ID:         entry.ID,
			StartedAt:  timeString(entry.StartedAt),
			ClientKey:  entry.ClientKey,
			ClientName: entry.ClientName,
			Method:     entry.Method,
			Path:       entry.Path,
			Provider:   entry.Provider,
			Protocol:   entry.Protocol,
			Model:      entry.Model,
			TokenID:    entry.TokenID,
			TokenName:  entry.TokenName,
		}
	}
	return out
}

func retryAttemptResponses(entries []history.RetryAttempt) []retryAttemptResponse {
	if len(entries) == 0 {
		return nil
	}
	out := make([]retryAttemptResponse, len(entries))
	for i, entry := range entries {
		out[i] = retryAttemptResponse{
			Attempt:           entry.Attempt,
			Provider:          entry.Provider,
			Protocol:          entry.Protocol,
			Model:             entry.Model,
			Status:            entry.Status,
			Duration:          entry.Duration,
			TokenID:           entry.TokenID,
			TokenName:         entry.TokenName,
			CooldownTriggered: entry.CooldownTriggered,
			Message:           entry.Message,
		}
	}
	return out
}

func timeString(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func timePtrString(value *time.Time) string {
	if value == nil {
		return ""
	}
	return timeString(*value)
}
