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
	ID             string        `json:"id"`
	StartedAt      string        `json:"startedAt"`
	FinishedAt     string        `json:"finishedAt"`
	Protocol       string        `json:"protocol"`
	Transport      string        `json:"transport"`
	InputModel     string        `json:"inputModel"`
	Model          string        `json:"model"`
	Stream         bool          `json:"stream"`
	ProviderID     string        `json:"providerId"`
	ProviderName   string        `json:"providerName"`
	AccountID      string        `json:"accountId"`
	AccountName    string        `json:"accountName"`
	Email          string        `json:"email"`
	WorkspaceID    string        `json:"workspaceId"`
	AccountDeleted bool          `json:"accountDeleted"`
	State          string        `json:"state"`
	StatusCode     int           `json:"statusCode"`
	ErrorCode      string        `json:"errorCode"`
	LatencyMS      int64         `json:"latencyMs"`
	FirstTokenMS   *int64        `json:"firstTokenMs,omitempty"`
	InputTokens    string        `json:"inputTokens"`
	OutputTokens   string        `json:"outputTokens"`
	InputKnown     bool          `json:"inputKnown"`
	OutputKnown    bool          `json:"outputKnown"`
	UsageComplete  bool          `json:"usageComplete"`
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
type UsageInfo struct {
	ProviderModels []ProviderModel `json:"providerModels,omitempty"`
	AllTime        Summary         `json:"allTime"`
	Days           []Breakdown     `json:"days"`
	Accounts       []Breakdown     `json:"accounts"`
	Models         []Breakdown     `json:"models"`
}
type Retention struct {
	MaxRecords    int `json:"maxRecords"`
	StoredRecords int `json:"storedRecords"`
	LogCapacity   int `json:"logCapacity"`
}
