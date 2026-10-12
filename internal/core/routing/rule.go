package routing

// Rule 是精确匹配配置，不包含凭证或请求内容。
type Rule struct {
	ID         string `json:"id"`
	Alias      string `json:"alias"`
	ProviderID string `json:"providerId"`
	Model      string `json:"model"`
	AccountID  string `json:"accountId"`
	Enabled    bool   `json:"enabled"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

// Match 区分大小写，只接受已启用的同名规则；数据层保证别名全局唯一。
func Match(model string, rules []Rule) (Rule, bool) {
	for _, rule := range rules {
		if rule.Enabled && rule.Alias == model {
			return rule, true
		}
	}
	return Rule{}, false
}
