package routing

// ProviderOption 是支持同名模型且具有可用账号的服务商配置。
type ProviderOption struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	CandidateCount int    `json:"candidateCount"`
}

// ProviderPriority 保留模型的已保存服务商顺序及当前可用性，资源删除后仍可诊断。
type ProviderPriority struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
}

// DefaultRoute 是从目录派生的初步映射，不会在读取时写入手工规则。
type DefaultRoute struct {
	Model                 string             `json:"model"`
	Name                  string             `json:"name"`
	Providers             []ProviderOption   `json:"providers"`
	PreferredProviderID   string             `json:"preferredProviderId"`
	PreferredProviderName string             `json:"preferredProviderName"`
	ProviderOrder         []ProviderPriority `json:"providerOrder"`
	Preview               Preview            `json:"preview"`
}
