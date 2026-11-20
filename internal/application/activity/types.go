// Package activity 保存与资源生命周期独立的代理请求和累计用量。
package activity

import (
	"database/sql"
	"errors"
	"sync"
)

const DefaultRetention = 10000
const MinRetention = 100
const MaxRetention = 1000000

var ErrUnavailable = errors.New("activity: unavailable")
var ErrInvalid = errors.New("activity: invalid_input")

type Service struct {
	db      *sql.DB
	mu      sync.Mutex
	updates chan struct{}
}

func New(db *sql.DB) *Service { return &Service{db: db, updates: make(chan struct{}, 1)} }

type RequestItem struct {
	AttemptCount   int           `json:"attemptCount,omitempty"`
	Attempts       []AttemptItem `json:"attempts,omitempty"`
	ProviderRegion string        `json:"providerRegion,omitempty"`
	BillingMode    string        `json:"billingMode,omitempty"`
	// Client 是发起请求的受管 Agent 标识；直接调用或历史记录为空。
	Client     string `json:"client,omitempty"`
	ID         string `json:"id"`
	StartedAt  string `json:"startedAt"`
	FinishedAt string `json:"finishedAt"`
	Protocol   string `json:"protocol"`
	Transport  string `json:"transport"`
	InputModel string `json:"inputModel"`
	Model      string `json:"model"`
	// Effort 是本次请求携带的推理档位；空串表示未指定（含历史记录），不推测上游默认值。
	Effort         string `json:"effort,omitempty"`
	Stream         bool   `json:"stream"`
	ProviderID     string `json:"providerId"`
	ProviderName   string `json:"providerName"`
	AccountID      string `json:"accountId"`
	AccountName    string `json:"accountName"`
	Email          string `json:"email"`
	WorkspaceID    string `json:"workspaceId"`
	AccountDeleted bool   `json:"accountDeleted"`
	State          string `json:"state"`
	StatusCode     int    `json:"statusCode"`
	ErrorCode      string `json:"errorCode"`
	LatencyMS      int64  `json:"latencyMs"`
	FirstTokenMS   *int64 `json:"firstTokenMs,omitempty"`
	InputTokens    string `json:"inputTokens"`
	OutputTokens   string `json:"outputTokens"`
	InputKnown     bool   `json:"inputKnown"`
	OutputKnown    bool   `json:"outputKnown"`
	UsageComplete  bool   `json:"usageComplete"`
	// CachedInputTokens 是上游报告的前缀缓存命中量，已包含在输入中；未知与真实的零分开表示。
	CachedInputTokens string `json:"cachedInputTokens"`
	CachedInputKnown  bool   `json:"cachedInputKnown"`
	// CostMin／CostMax 是按基础参考价估算的美元成本，区间仅在缓存明细未知时出现。
	CostMin   float64 `json:"costMin"`
	CostMax   float64 `json:"costMax"`
	CostKnown bool    `json:"costKnown"`
}
type RequestQuery struct {
	Search string `json:"search"`
	State  string `json:"state"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}
type RequestPage struct {
	Items      []RequestItem `json:"items"`
	Total      int           `json:"total"`
	Limit      int           `json:"limit"`
	MaxRecords int           `json:"maxRecords"`
}
type Summary struct {
	InputTokens   string `json:"inputTokens"`
	OutputTokens  string `json:"outputTokens"`
	TotalTokens   string `json:"totalTokens"`
	Requests      int64  `json:"requests"`
	CompleteUsage int64  `json:"completeUsage"`
}
type Breakdown struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Email       string  `json:"email"`
	WorkspaceID string  `json:"workspaceId"`
	Summary     Summary `json:"summary"`
}

// HourUsage 是近 30 天按本地时刻聚合的用量；0–23 全量返回，空时段保持零。
type HourUsage struct {
	Hour    int     `json:"hour"`
	Summary Summary `json:"summary"`
}

// ModelDay 是单个模型在近 30 天中某一天的用量，用于模型结构趋势。
type ModelDay struct {
	Day     string  `json:"day"`
	Model   string  `json:"model"`
	Summary Summary `json:"summary"`
}

// DayUsage 是一个本地日期按开始时间归属的用量，用于用量页的年度日历热图。
// 只包含有记录（含仅请求数）的日期；没有记录的日期由前端补零，不伪造为已统计。
type DayUsage struct {
	Day          string `json:"day"`
	InputTokens  string `json:"inputTokens"`
	OutputTokens string `json:"outputTokens"`
	TotalTokens  string `json:"totalTokens"`
	Requests     int64  `json:"requests"`
}

// CacheUsage 是缓存命中的可确认下界：只统计上游明确报告过缓存明细的请求。
type CacheUsage struct {
	CachedInputTokens string `json:"cachedInputTokens"`
	KnownRequests     int64  `json:"knownRequests"`
}

// Reliability 是近 30 天已结束请求的质量汇总；成功率与仪表盘口径一致。
type Reliability struct {
	Requests            int64    `json:"requests"`
	Success             int64    `json:"success"`
	Failed              int64    `json:"failed"`
	Cancelled           int64    `json:"cancelled"`
	SuccessRate         *float64 `json:"successRate"`
	AverageLatencyMS    *float64 `json:"averageLatencyMs"`
	AverageFirstTokenMS *float64 `json:"averageFirstTokenMs"`
	FirstTokenSamples   int64    `json:"firstTokenSamples"`
}

type UsageInfo struct {
	ProviderModels []ProviderModel `json:"providerModels,omitempty"`
	Previous       Summary         `json:"previous"`
	Cache          CacheUsage      `json:"cache"`
	Reliability    Reliability     `json:"reliability"`
	AllTime        Summary         `json:"allTime"`
	Days           []Breakdown     `json:"days"`
	Calendar       []DayUsage      `json:"calendar"`
	Hours          []HourUsage     `json:"hours"`
	Providers      []Breakdown     `json:"providers"`
	Accounts       []Breakdown     `json:"accounts"`
	Models         []Breakdown     `json:"models"`
	ModelDays      []ModelDay      `json:"modelDays"`
}
type Retention struct {
	MaxRecords     int   `json:"maxRecords"`
	StoredRecords  int   `json:"storedRecords"`
	LogCapacity    int   `json:"logCapacity"`
	ContentEnabled bool  `json:"contentEnabled"`
	ContentBytes   int64 `json:"contentBytes"`
	ContentBudget  int64 `json:"contentBudget"`
}
