package routing

import (
	"context"
	"omniproxy/internal/application/account"
	core "omniproxy/internal/core/routing"
	"omniproxy/internal/provider/mimo"
)

func (s *Service) withMiMoPresets(ctx context.Context, models []discoveredModel) ([]discoveredModel, error) {
	accounts, err := s.accounts.List(ctx)
	if err != nil {
		return nil, err
	}
	providers, err := s.providers.List(ctx)
	if err != nil {
		return nil, err
	}
	mimoProviders := map[string]bool{}
	for _, p := range providers {
		if p.Kind == mimo.Kind {
			mimoProviders[p.ID] = true
		}
	}
	synced := map[string]bool{}
	for _, m := range models {
		synced[m.accountID] = true
	}
	for _, a := range accounts {
		if !mimoProviders[a.ProviderID] || a.AuthMethod != account.AuthAPIKey || a.MiMo == nil || !a.MiMo.Valid() || synced[a.ID] {
			continue
		}
		for _, id := range mimo.AccountModels(*a.MiMo) {
			models = append(models, discoveredModel{accountID: a.ID, providerID: a.ProviderID, model: id, name: id, metadata: mimo.ModelMetadata(id)})
		}
	}
	if len(models) > 10000 {
		return nil, core.ErrInvalidInput
	}
	return models, nil
}

func mimoTargetSupports(data snapshot, providerID, accountID, model string) bool {
	for _, a := range data.accounts {
		if a.ProviderID != providerID || accountID != "" && a.ID != accountID || a.MiMo == nil || !mimo.SupportsAccountModel(*a.MiMo, model) {
			continue
		}
		for _, m := range data.models {
			if m.accountID == a.ID && m.model == model {
				return true
			}
		}
	}
	return false
}

func mimoAmbiguous(data snapshot, allowed map[string]bool, model, fixed string) bool {
	if fixed != "" {
		return false
	}
	var first *mimo.Options
	for _, a := range data.accounts {
		if !a.Enabled || a.AuthMethod != account.AuthAPIKey || !allowed[a.ProviderID] || a.MiMo == nil || !mimoTargetSupports(data, a.ProviderID, a.ID, model) {
			continue
		}
		if first != nil && !first.SamePool(*a.MiMo) {
			return true
		}
		first = a.MiMo
	}
	return false
}

// 默认通道别名由结构化配置生成，再精确匹配，不根据用户输入拆解并猜测计费方式。
func mimoAliases(data snapshot) []core.Rule {
	ids := map[string]bool{}
	rules := []core.Rule{}
	modelsByAccount := map[string][]discoveredModel{}
	for _, model := range data.models {
		modelsByAccount[model.accountID] = append(modelsByAccount[model.accountID], model)
	}
	for _, a := range data.accounts {
		if !a.Enabled || a.AuthMethod != account.AuthAPIKey || a.MiMo == nil || !a.MiMo.Valid() {
			continue
		}
		for _, p := range data.providers {
			if !p.Enabled || p.ID != a.ProviderID || p.Kind != mimo.Kind {
				continue
			}
			for _, m := range modelsByAccount[a.ID] {
				alias := a.MiMo.Channel() + "/" + m.model
				if ids[alias] {
					continue
				}
				ids[alias] = true
				rules = append(rules, core.Rule{Alias: alias, Model: m.model, ProviderID: p.ID, AccountID: a.ID, Enabled: true})
			}
		}
	}
	return rules
}
