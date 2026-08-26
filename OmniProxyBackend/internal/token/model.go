package token

import "time"

type Status string

const (
	StatusActive    Status = "active"
	StatusLow       Status = "low"
	StatusExhausted Status = "exhausted"
	StatusInvalid   Status = "invalid"
)

const (
	ProviderOpenAI      = "openai"
	ProviderAnthropic   = "anthropic"
	ProviderDeepSeek    = "deepseek"
	ProviderKimi        = "kimi"
	ProviderXiaomi      = "xiaomi"
	ProviderZhipu       = "zhipu"
	ProviderMiniMax     = "minimax"
	ProviderGemini      = "gemini"
	ProviderOpenRouter  = "openrouter"
	ProviderTokenRouter = "tokenrouter"
	ProviderSub2API     = "sub2api"
	ProviderNewAPI      = "newapi"
	ProviderAnyRouter   = "anyrouter"
	ProviderZo          = "zo"
	ProviderPrem        = "prem"
	ProviderCustom      = "custom"
)

const (
	CredentialTypeAPIKey        = "api_key"
	CredentialTypeCodexAuthJSON = "codex_auth_json"
	CredentialTypeMimoTokenPlan = "mimo_token_plan"
	CredentialTypeCodingPlan    = "coding_plan"
	CredentialTypeClaudeOAuth   = "claude_oauth_json"
)

const (
	MimoRegionCN  = "cn"
	MimoRegionSGP = "sgp"
	MimoRegionAMS = "ams"
)

type Token struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Provider       string     `json:"provider"`
	CredentialType string     `json:"credentialType"`
	Region         string     `json:"region,omitempty"`
	BaseURL        string     `json:"baseUrl,omitempty"`
	TokenValue     string     `json:"tokenValue"`
	Remaining      int        `json:"remaining"`
	Usage          UsageInfo  `json:"usage"`
	Stats          TokenStats `json:"stats"`
	Health         HealthInfo `json:"health"`
	Status         Status     `json:"status"`
	Disabled       bool       `json:"disabled,omitempty"`
	Selected       bool       `json:"selected,omitempty"`
	Pinned         bool       `json:"pinned,omitempty"`
	LastUsedAt     *time.Time `json:"lastUsedAt,omitempty"`
	LastError      string     `json:"lastError,omitempty"`
	CooldownUntil  *time.Time `json:"cooldownUntil,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type UsageInfo struct {
	Source                         string             `json:"source,omitempty"`
	PlanType                       string             `json:"planType,omitempty"`
	LimitReached                   bool               `json:"limitReached,omitempty"`
	PrimaryUsedPercent             int                `json:"primaryUsedPercent,omitempty"`
	PrimaryUsedPercentExact        float64            `json:"primaryUsedPercentExact,omitempty"`
	PrimaryRemainingPercent        int                `json:"primaryRemainingPercent,omitempty"`
	PrimaryResetAt                 int64              `json:"primaryResetAt,omitempty"`
	PrimaryActivationPending       bool               `json:"primaryWindowPendingActivation,omitempty"`
	SecondaryUsedPercent           int                `json:"secondaryUsedPercent,omitempty"`
	SecondaryUsedPercentExact      float64            `json:"secondaryUsedPercentExact,omitempty"`
	SecondaryRemainingPercent      int                `json:"secondaryRemainingPercent,omitempty"`
	SecondaryResetAt               int64              `json:"secondaryResetAt,omitempty"`
	SecondaryActivationPending     bool               `json:"secondaryWindowPendingActivation,omitempty"`
	APIRemaining                   int                `json:"apiRemaining,omitempty"`
	BalanceRemaining               float64            `json:"balanceRemaining,omitempty"`
	BalanceTotal                   float64            `json:"balanceTotal,omitempty"`
	BalanceUsed                    float64            `json:"balanceUsed,omitempty"`
	BalanceUnit                    string             `json:"balanceUnit,omitempty"`
	BalanceUnlimited               bool               `json:"balanceUnlimited,omitempty"`
	BalancePackages                []BalancePackage   `json:"balancePackages,omitempty"`
	CodexResetCreditsAvailable     *int               `json:"codexResetCreditsAvailable,omitempty"`
	CodexResetCredits              []CodexResetCredit `json:"codexResetCredits,omitempty"`
	CodexResetCreditsNextExpiresAt int64              `json:"codexResetCreditsNextExpiresAt,omitempty"`
	CodexResetCreditsError         string             `json:"codexResetCreditsError,omitempty"`
	CodexResetCreditsCheckedAt     int64              `json:"codexResetCreditsCheckedAt,omitempty"`
	SubscriptionQuotaAvailable     bool               `json:"subscriptionQuotaAvailable,omitempty"`
	Message                        string             `json:"message,omitempty"`
	UpdatedAt                      *time.Time         `json:"updatedAt,omitempty"`
}

type CodexResetCredit struct {
	ID         string `json:"id,omitempty"`
	Status     string `json:"status,omitempty"`
	ResetType  string `json:"resetType,omitempty"`
	GrantedAt  int64  `json:"grantedAt,omitempty"`
	ExpiresAt  int64  `json:"expiresAt,omitempty"`
	RedeemedAt int64  `json:"redeemedAt,omitempty"`
	RawStatus  string `json:"rawStatus,omitempty"`
}

func (usage UsageInfo) EffectiveRemainingPercent() int {
	switch {
	case usage.hasPrimaryQuotaWindow():
		return clampPercent(usage.PrimaryRemainingPercent)
	case usage.hasSecondaryQuotaWindow():
		return clampPercent(usage.SecondaryRemainingPercent)
	default:
		return clampPercent(usage.PrimaryRemainingPercent)
	}
}

func (usage UsageInfo) hasPrimaryQuotaWindow() bool {
	return usage.PrimaryUsedPercent != 0 || usage.PrimaryRemainingPercent != 0 || usage.PrimaryResetAt != 0
}

func (usage UsageInfo) hasSecondaryQuotaWindow() bool {
	return usage.SecondaryUsedPercent != 0 || usage.SecondaryRemainingPercent != 0 || usage.SecondaryResetAt != 0
}

func clampPercent(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

type BalancePackage struct {
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

type TokenStats struct {
	RequestCount                int64                        `json:"requestCount"`
	InputTokens                 int64                        `json:"inputTokens"`
	OutputTokens                int64                        `json:"outputTokens"`
	TotalTokens                 int64                        `json:"totalTokens"`
	CacheCreationTokens         int64                        `json:"cacheCreationTokens,omitempty"`
	CacheReadTokens             int64                        `json:"cacheReadTokens,omitempty"`
	CodexWeeklyEstimateResetAt  int64                        `json:"codexWeeklyEstimateResetAt,omitempty"`
	CodexWeeklyEstimateBaseline *CodexWeeklyEstimateBaseline `json:"codexWeeklyEstimateBaseline,omitempty"`
	LastInputTokens             int                          `json:"lastInputTokens,omitempty"`
	LastOutputTokens            int                          `json:"lastOutputTokens,omitempty"`
	LastTotalTokens             int                          `json:"lastTotalTokens,omitempty"`
	LastCacheCreationTokens     int                          `json:"lastCacheCreationTokens,omitempty"`
	LastCacheReadTokens         int                          `json:"lastCacheReadTokens,omitempty"`
	Daily                       []DailyTokenUsage            `json:"daily,omitempty"`
	UpdatedAt                   *time.Time                   `json:"updatedAt,omitempty"`
}

type CodexWeeklyEstimateBaseline struct {
	InputTokens         int64 `json:"inputTokens"`
	OutputTokens        int64 `json:"outputTokens"`
	TotalTokens         int64 `json:"totalTokens"`
	CacheCreationTokens int64 `json:"cacheCreationTokens,omitempty"`
	CacheReadTokens     int64 `json:"cacheReadTokens,omitempty"`
}

type HealthInfo struct {
	LastCheckedAt     *time.Time `json:"lastCheckedAt,omitempty"`
	NextCheckAt       *time.Time `json:"nextCheckAt,omitempty"`
	ConsecutiveErrors int        `json:"consecutiveErrors,omitempty"`
	LastStatus        int        `json:"lastStatus,omitempty"`
	LastMessage       string     `json:"lastMessage,omitempty"`
}

type DailyTokenUsage struct {
	Date                string `json:"date"`
	RequestCount        int64  `json:"requestCount"`
	InputTokens         int64  `json:"inputTokens"`
	OutputTokens        int64  `json:"outputTokens"`
	TotalTokens         int64  `json:"totalTokens"`
	CacheCreationTokens int64  `json:"cacheCreationTokens,omitempty"`
	CacheReadTokens     int64  `json:"cacheReadTokens,omitempty"`
}

type TokenConsumption struct {
	InputTokens         int `json:"inputTokens"`
	OutputTokens        int `json:"outputTokens"`
	TotalTokens         int `json:"totalTokens"`
	CacheCreationTokens int `json:"cacheCreationTokens,omitempty"`
	CacheReadTokens     int `json:"cacheReadTokens,omitempty"`
}

type UpsertRequest struct {
	Name           string `json:"name"`
	Provider       string `json:"provider"`
	CredentialType string `json:"credentialType"`
	Region         string `json:"region,omitempty"`
	BaseURL        string `json:"baseUrl,omitempty"`
	TokenValue     string `json:"tokenValue"`
}
