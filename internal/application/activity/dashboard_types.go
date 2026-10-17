package activity

// DashboardQuery 使用客户端本地时区生成的 UTC 边界，不按服务端时区猜测日期。
type DashboardQuery struct {
	Boundaries []int64 `json:"boundaries"`
}

// DashboardSummary 的结果与 Token 属于同一批按开始时间归属的请求。
type DashboardSummary struct {
	AverageFirstTokenMS *float64 `json:"averageFirstTokenMs,omitempty"`
	FirstTokenSamples   int64    `json:"firstTokenSamples,omitempty"`
	Requests            int64    `json:"requests"`
	Success             int64    `json:"success"`
	Failed              int64    `json:"failed"`
	Cancelled           int64    `json:"cancelled"`
	Pending             int64    `json:"pending"`
	InputTokens         string   `json:"inputTokens"`
	OutputTokens        string   `json:"outputTokens"`
	TotalTokens         string   `json:"totalTokens"`
	CompleteUsage       int64    `json:"completeUsage"`
	SuccessRate         *float64 `json:"successRate"`
	AverageLatencyMS    *float64 `json:"averageLatencyMs"`
}

// DashboardPoint 对应一个本地小时或日期区间。
type DashboardPoint struct {
	Start   int64            `json:"start"`
	End     int64            `json:"end"`
	Summary DashboardSummary `json:"summary"`
}

// DashboardInfo 不包含凭据或请求正文，缺失历史不会推算为完整数据。
type DashboardInfo struct {
	ProviderDistribution []DashboardProvider `json:"providerDistribution,omitempty"`
	PendingNow           *int64              `json:"pendingNow,omitempty"`
	Summary              DashboardSummary    `json:"summary"`
	Points               []DashboardPoint    `json:"points"`
	Recent               []RequestItem       `json:"recent"`
	HistoryIncomplete    bool                `json:"historyIncomplete"`
	CapturedFrom         int64               `json:"capturedFrom"`
	UpdatedAt            string              `json:"updatedAt"`
}

// DashboardProvider 使用与明细保留期独立的累计路由计数，不冒充所选时段的分布。
type DashboardProvider struct {
	ProviderID   string `json:"providerId"`
	ProviderName string `json:"providerName"`
	Requests     int64  `json:"requests"`
}
