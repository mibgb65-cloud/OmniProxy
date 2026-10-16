package routing

import (
	"context"
	core "omniproxy/internal/core/routing"
	"omniproxy/internal/provider/kimi"
)

func (s *Service) withKimiPresets(ctx context.Context, models []discoveredModel) ([]discoveredModel, error) {
	accounts, err := s.accounts.List(ctx)
	if err != nil {
		return nil, err
	}
	providers, err := s.providers.List(ctx)
	if err != nil {
		return nil, err
	}
	allowed := map[string]bool{}
	for _, p := range providers {
		if p.Kind == kimi.Kind {
			allowed[p.ID] = true
		}
	}
	synced := map[string]bool{}
	for _, m := range models {
		synced[m.accountID] = true
	}
	for _, a := range accounts {
		if !allowed[a.ProviderID] || a.AuthMethod != "api_key" || a.Kimi == nil || !a.Kimi.Valid() || synced[a.ID] {
			continue
		}
		for _, id := range kimi.AccountModels(*a.Kimi) {
			models = append(models, discoveredModel{accountID: a.ID, providerID: a.ProviderID, model: id, name: id, metadata: kimi.AccountModelMetadata(*a.Kimi, id)})
		}
	}
	if len(models) > 10000 {
		return nil, core.ErrInvalidInput
	}
	return models, nil
}

func kimiTargetSupports(data snapshot, providerID, accountID, model string) bool {
	for _, a := range data.accounts {
		if a.ProviderID != providerID || accountID != "" && a.ID != accountID || a.Kimi == nil || !kimi.SupportsAccount(*a.Kimi, model, "chat") {
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

func prioritizeKimiRegion(data snapshot, candidates []core.Account, allowed map[string]bool, fixed string) {
	if fixed != "" {
		return
	}
	regions := map[string]string{}
	for i, a := range data.accounts {
		if !candidates[i].Enabled || !allowed[a.ProviderID] || a.Kimi == nil || !a.Kimi.Valid() {
			continue
		}
		if regions[a.ProviderID] == "" || a.Kimi.Region == data.kimiSettings.PreferredRegion {
			regions[a.ProviderID] = a.Kimi.Region
		}
	}
	for i, a := range data.accounts {
		if a.Kimi != nil && regions[a.ProviderID] != "" && regions[a.ProviderID] != a.Kimi.Region {
			candidates[i].Enabled = false
		}
	}
}

func kimiAliases(data snapshot) []core.Rule {
	models := map[string][]discoveredModel{}
	for _, m := range data.models {
		models[m.accountID] = append(models[m.accountID], m)
	}
	providers := map[string]bool{}
	for _, p := range data.providers {
		if p.Enabled && p.Kind == kimi.Kind {
			providers[p.ID] = true
		}
	}
	seen := map[string]bool{}
	rules := []core.Rule{}
	for _, a := range data.accounts {
		if !a.Enabled || !providers[a.ProviderID] || a.Kimi == nil || !a.Kimi.Valid() {
			continue
		}
		for _, m := range models[a.ID] {
			if !kimi.SupportsAccount(*a.Kimi, m.model, "chat") {
				continue
			}
			alias := "kimi-" + a.Kimi.Region + "/" + m.model
			if a.Kimi.Mode() == "coding" {
				alias = "kimi-code-" + a.Kimi.Region + "/" + m.model
			}
			if seen[alias] {
				continue
			}
			seen[alias] = true
			rules = append(rules, core.Rule{Alias: alias, Model: m.model, ProviderID: a.ProviderID, AccountID: a.ID, Enabled: true})
		}
	}
	return rules
}
