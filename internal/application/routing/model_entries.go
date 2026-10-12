package routing

import (
	"context"

	catalog "omniproxy/internal/core/modelcatalog"
	core "omniproxy/internal/core/routing"
)

// Catalog 使用实际路由决策的账号元数据，不把其他同名模型的能力混入目标。
func (s *Service) Catalog(ctx context.Context) ([]catalog.Entry, error) {
	data, err := s.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := s.rules(ctx)
	if err != nil {
		return nil, err
	}
	ids := []string{"deepseek-flash", "deepseek-v4-pro"}
	for _, model := range data.models {
		ids = append(ids, model.model)
	}
	for _, rule := range rules {
		if rule.Enabled {
			ids = append(ids, rule.Alias)
		}
	}
	entries := []catalog.Entry{}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		rule, matched := core.Match(id, rules)
		decision := evaluate(id, rule, matched, data)
		if !decision.Routable {
			continue
		}
		entry := catalog.Entry{ID: id, Name: id, TargetModel: decision.Model, ProviderID: decision.ProviderID, Alias: matched}
		entry.Protocols = []string{"chat_completions", "responses", "messages"}
		for _, provider := range data.providers {
			if provider.ID == decision.ProviderID {
				entry.ProviderKind = provider.Kind
			}
		}
		for _, model := range data.models {
			if model.accountID == decision.AccountID && model.model == decision.Model {
				entry.Metadata = model.metadata
				if !matched && model.name != "" {
					entry.Name = model.name
				}
				break
			}
		}
		if matched {
			entry.Metadata.IsDefault = false
		}
		entries = append(entries, entry)
	}
	return catalog.Normalize(entries)
}
