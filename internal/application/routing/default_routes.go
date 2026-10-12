package routing

import (
	"context"
	"sort"

	core "omniproxy/internal/core/routing"
	"omniproxy/internal/provider/deepseek"
)

func originalModels(data snapshot) []string {
	seen := make(map[string]bool)
	for _, provider := range data.providers {
		if provider.Kind == "deepseek" {
			for _, id := range deepseek.Models() {
				seen[id] = true
			}
		}
	}
	for _, model := range data.models {
		seen[model.model] = true
	}
	for model := range data.preferences {
		seen[model] = true
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// DefaultRoutes 读取目录的同名初步映射和实际选择，保留无效偏好用于诊断。
func (s *Service) DefaultRoutes(ctx context.Context) ([]DefaultRoute, error) {
	data, err := s.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := s.rules(ctx)
	if err != nil {
		return nil, err
	}
	items := []DefaultRoute{}
	for _, id := range originalModels(data) {
		rule, matched := core.Match(id, rules)
		item := DefaultRoute{Model: id, Name: id, PreferredProviderID: data.preferences[id], Providers: []ProviderOption{}, Preview: evaluate(id, rule, matched, data)}
		for _, model := range data.models {
			if model.model == id && model.name != "" {
				item.Name = model.name
				break
			}
		}
		for _, provider := range data.providers {
			if provider.ID == item.PreferredProviderID {
				item.PreferredProviderName = provider.Name
			}
			preview := evaluate(id, core.Rule{ProviderID: provider.ID, Model: id}, true, data)
			if preview.Routable {
				item.Providers = append(item.Providers, ProviderOption{ID: provider.ID, Name: provider.Name, CandidateCount: preview.CandidateCount})
			}
		}
		sort.Slice(item.Providers, func(i, j int) bool { return item.Providers[i].ID < item.Providers[j].ID })
		items = append(items, item)
	}
	return items, nil
}
