package routing

import (
	"context"
	"omniproxy/internal/application/account"
	core "omniproxy/internal/core/routing"
	"omniproxy/internal/provider/minimax"
)

func (s *Service) withMiniMaxPresets(ctx context.Context, models []discoveredModel) ([]discoveredModel, error) {
	accounts, err := s.accounts.List(ctx)
	if err != nil {
		return nil, err
	}
	providers, err := s.providers.List(ctx)
	if err != nil {
		return nil, err
	}
	minimaxProviders := map[string]bool{}
	for _, p := range providers {
		if p.Kind == minimax.Kind {
			minimaxProviders[p.ID] = true
		}
	}
	synced := map[string]bool{}
	for _, m := range models {
		synced[m.accountID] = true
	}
	for _, a := range accounts {
		if !minimaxProviders[a.ProviderID] || a.AuthMethod != account.AuthAPIKey || a.MiniMax == nil || !a.MiniMax.Valid() || synced[a.ID] {
			continue
		}
		for _, id := range minimax.AccountModels(*a.MiniMax) {
			models = append(models, discoveredModel{accountID: a.ID, providerID: a.ProviderID, model: id, name: id, metadata: minimax.ModelMetadata(id)})
		}
	}
	if len(models) > 10000 {
		return nil, core.ErrInvalidInput
	}
	return models, nil
}

func minimaxTargetSupports(data snapshot, providerID, accountID, model string) bool {
	for _, a := range data.accounts {
		if a.ProviderID != providerID || accountID != "" && a.ID != accountID || a.MiniMax == nil || !minimax.Supports(*a.MiniMax, model, "chat") {
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

func minimaxAmbiguous(data snapshot, allowed map[string]bool, model, fixed string) bool {
	if fixed != "" {
		return false
	}
	var first *minimax.Options
	for _, a := range data.accounts {
		if !a.Enabled || !allowed[a.ProviderID] || a.MiniMax == nil || !minimaxTargetSupports(data, a.ProviderID, a.ID, model) {
			continue
		}
		if first != nil && !first.SamePool(*a.MiniMax) {
			return true
		}
		first = a.MiniMax
	}
	return false
}

// 默认通道别名由结构化配置生成，再精确匹配，不根据用户输入拆解并猜测计费方式。
func minimaxAliases(data snapshot) []core.Rule {
	ids := map[string]bool{}
	rules := []core.Rule{}
	modelsByAccount := map[string][]discoveredModel{}
	for _, model := range data.models {
		modelsByAccount[model.accountID] = append(modelsByAccount[model.accountID], model)
	}
	for _, a := range data.accounts {
		if !a.Enabled || a.MiniMax == nil || !a.MiniMax.Valid() {
			continue
		}
		for _, p := range data.providers {
			if !p.Enabled || p.ID != a.ProviderID || p.Kind != minimax.Kind {
				continue
			}
			for _, m := range modelsByAccount[a.ID] {
				mode := "api"
				if a.MiniMax.BillingMode == "subscription" {
					mode = "plan"
				}
				alias := "minimax-" + a.MiniMax.Region + "-" + mode + "/" + m.model
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
