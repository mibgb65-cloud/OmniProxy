package routing

// ProviderOption 是支持同名模型且具有可用账号的服务商配置。
type ProviderOption struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	CandidateCount int    `json:"candidateCount"`
}

// DefaultRoute 是从目录派生的初步映射，不会在读取时写入手工规则。
type DefaultRoute struct {
	Model                 string           `json:"model"`
	Name                  string           `json:"name"`
	Providers             []ProviderOption `json:"providers"`
	PreferredProviderID   string           `json:"preferredProviderId"`
	PreferredProviderName string           `json:"preferredProviderName"`
	Preview               Preview          `json:"preview"`
}
