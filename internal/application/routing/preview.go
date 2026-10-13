package routing

import (
	"context"
	"fmt"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	core "omniproxy/internal/core/routing"
)

type snapshot struct {
	providers   []provider.Configuration
	accounts    []account.Item
	models      []discoveredModel
	preferences map[string][]string
}

func (s *Service) snapshot(ctx context.Context) (snapshot, error) {
	if s.providers == nil || s.accounts == nil {
		return snapshot{}, fmt.Errorf("routing: configuration unavailable")
	}
	providers, err := s.providers.List(ctx)
	if err != nil {
		return snapshot{}, fmt.Errorf("routing: configuration unavailable")
	}
	accounts, err := s.accounts.List(ctx)
	if err != nil {
		return snapshot{}, fmt.Errorf("routing: configuration unavailable")
	}
	models, err := s.discovered(ctx)
	if err != nil {
		return snapshot{}, fmt.Errorf("routing: configuration unavailable")
	}
	preferences, err := s.preferences(ctx)
	if err != nil {
		return snapshot{}, err
	}
	return snapshot{providers: providers, accounts: accounts, models: models, preferences: preferences}, nil
}

// List 读取配置和实时诊断，不读取凭证；删除目标后规则仍可编辑修复。
func (s *Service) List(ctx context.Context) ([]Item, error) {
	rules, err := s.rules(ctx)
	if err != nil {
		return nil, err
	}
	data, err := s.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(rules))
	for _, rule := range rules {
		preview := evaluate(rule.Alias, rule, true, data)
		accountName := ""
		for _, account := range data.accounts {
			if account.ID == rule.AccountID {
				accountName = account.Name
				break
			}
		}
		items = append(items, Item{Rule: rule, ProviderName: preview.ProviderName, AccountName: accountName, ErrorCode: preview.ErrorCode})
	}
	return items, nil
}

// Resolve 与实际网关使用同一规则，结果只基于元数据，不保证认证或额度。
func (s *Service) Resolve(ctx context.Context, model string) (Preview, error) {
	if !core.ValidModel(model) {
		return Preview{}, ErrInvalidInput
	}
	rules, err := s.rules(ctx)
	if err != nil {
		return Preview{}, err
	}
	data, err := s.snapshot(ctx)
	if err != nil {
		return Preview{}, err
	}
	rule, matched := core.Match(model, rules)
	return evaluate(model, rule, matched, data), nil
}

// Aliases 只公布已启用且配置可解析的别名，原始模型由网关保留。
func (s *Service) Aliases(ctx context.Context) ([]string, error) {
	items, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	aliases := make([]string, 0)
	for _, item := range items {
		if item.Rule.Enabled && item.ErrorCode == "" {
			aliases = append(aliases, item.Rule.Alias)
		}
	}
	return aliases, nil
}

func evaluate(model string, rule core.Rule, matched bool, data snapshot) Preview {
	result := Preview{InputModel: model, Model: model, Matched: matched}
	var providerOrder []string
	allowed := make(map[string]bool)
	if matched {
		result.RuleID, result.Model, result.ProviderID = rule.ID, rule.Model, rule.ProviderID
		found := false
		for _, provider := range data.providers {
			if provider.ID != rule.ProviderID {
				continue
			}
			found = true
			result.ProviderName = provider.Name
			if provider.Kind != "deepseek" && provider.Kind != "openai" {
				result.ErrorCode = "adapter_unavailable"
				return result
			}
			if !provider.Enabled {
				result.ErrorCode = "provider_disabled"
				return result
			}
			allowed[provider.ID] = true
			break
		}
		if !found {
			result.ErrorCode = "provider_missing"
			return result
		}
		if !targetSupports(data, rule.ProviderID, rule.AccountID, rule.Model) {
			result.ErrorCode = "unsupported_model"
			return result
		}
	} else {
		providerOrder = data.preferences[model]
		if len(providerOrder) > 0 {
			result.PreferredProviderID = providerOrder[0]
		}
		for _, provider := range data.providers {
			if provider.Enabled && targetSupports(data, provider.ID, "", model) {
				allowed[provider.ID] = true
			}
		}
		if len(allowed) == 0 {
			result.ErrorCode = "unsupported_model"
			if supportedModel(model) {
				result.ErrorCode = "no_enabled_deepseek_provider"
			}
			return result
		}
	}
	accounts := make([]core.Account, 0, len(data.accounts))
	for _, account := range data.accounts {
		accounts = append(accounts, core.Account{ID: account.ID, ProviderID: account.ProviderID, CreatedAt: account.CreatedAt, Enabled: account.Enabled && eligible(data, account, result.Model)})
	}
	selected, count, code := core.SelectPrioritizedAccount(accounts, allowed, rule.AccountID, providerOrder)
	result.CandidateCount, result.ErrorCode = count, code
	if code != "" {
		return result
	}
	result.AccountID, result.ProviderID, result.Routable = selected.ID, selected.ProviderID, true
	result.PreferenceApplied = result.PreferredProviderID != "" && result.PreferredProviderID == selected.ProviderID
	for index, id := range providerOrder {
		if id == selected.ProviderID {
			result.ProviderPriority = index + 1
			break
		}
	}
	for _, provider := range data.providers {
		if provider.ID == selected.ProviderID {
			result.ProviderName = provider.Name
			break
		}
	}
	for _, account := range data.accounts {
		if account.ID == selected.ID {
			result.AccountName = account.Name
			break
		}
	}
	return result
}
