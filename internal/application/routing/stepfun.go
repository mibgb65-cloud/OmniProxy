package routing

import (
	"context"
	"omniproxy/internal/application/account"
	core "omniproxy/internal/core/routing"
	"omniproxy/internal/provider/stepfun"
)

func (s *Service) withStepPresets(ctx context.Context, models []discoveredModel) ([]discoveredModel, error) {
	accounts, err := s.accounts.List(ctx)
	if err != nil {
		return nil, err
	}
	providers, err := s.providers.List(ctx)
	if err != nil {
		return nil, err
	}
	stepProviders := map[string]bool{}
	for _, p := range providers {
		if p.Kind == stepfun.Kind {
			stepProviders[p.ID] = true
		}
	}
	synced := map[string]bool{}
	for _, m := range models {
		synced[m.accountID] = true
	}
	for _, a := range accounts {
		if !stepProviders[a.ProviderID] || a.AuthMethod != account.AuthAPIKey || a.StepFun == nil || !a.StepFun.Valid() || synced[a.ID] {
			continue
		}
		for _, id := range stepfun.AccountModels(*a.StepFun) {
			models = append(models, discoveredModel{accountID: a.ID, providerID: a.ProviderID, model: id, name: id, metadata: stepfun.ModelMetadata(id)})
		}
	}
	if len(models) > 10000 {
		return nil, core.ErrInvalidInput
	}
	return models, nil
}

func stepTargetSupports(data snapshot, providerID, accountID, model string) bool {
	for _, a := range data.accounts {
		if a.ProviderID != providerID || accountID != "" && a.ID != accountID || a.StepFun == nil || !stepfun.Supports(*a.StepFun, model, "chat") {
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

func stepAmbiguous(data snapshot, allowed map[string]bool, model, fixed string) bool {
	if fixed != "" {
		return false
	}
	var first *stepfun.Options
	for _, a := range data.accounts {
		if !a.Enabled || !allowed[a.ProviderID] || a.StepFun == nil || !stepTargetSupports(data, a.ProviderID, a.ID, model) {
			continue
		}
		if first != nil && !first.SamePool(*a.StepFun) {
			return true
		}
		first = a.StepFun
	}
	return false
}

// 默认通道别名由结构化配置生成，再精确匹配，不根据用户输入拆解并猜测计费方式。
func stepAliases(data snapshot) []core.Rule {
	ids := map[string]bool{}
	rules := []core.Rule{}
	modelsByAccount := map[string][]discoveredModel{}
	for _, model := range data.models {
		modelsByAccount[model.accountID] = append(modelsByAccount[model.accountID], model)
	}
	for _, a := range data.accounts {
		if !a.Enabled || a.StepFun == nil || !a.StepFun.Valid() {
			continue
		}
		for _, p := range data.providers {
			if !p.Enabled || p.ID != a.ProviderID || p.Kind != stepfun.Kind {
				continue
			}
			for _, m := range modelsByAccount[a.ID] {
				mode := "api"
				if a.StepFun.BillingMode == "step_plan" {
					mode = "plan"
				}
				alias := "stepfun-" + a.StepFun.Region + "-" + mode + "/" + m.model
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

func matchRoute(model string, rules []core.Rule, data snapshot) (core.Rule, bool) {
	if rule, matched := core.Match(model, rules); matched {
		return rule, true
	}
	if rule, matched := core.Match(model, stepAliases(data)); matched {
		return rule, true
	}
	return core.Match(model, kimiAliases(data))
}
