package routing

import (
	"context"

	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/provider/atria"
	"omniproxy/internal/provider/deepseek"
	"omniproxy/internal/provider/kimi"
	"omniproxy/internal/provider/mimo"
	"omniproxy/internal/provider/minimax"
	"omniproxy/internal/provider/stepfun"
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
	ids := originalModels(data)
	rules = append(append(append(rules, stepAliases(data)...), kimiAliases(data)...), minimaxAliases(data)...)
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
		rule, matched := matchRoute(id, rules, data)
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
		if entry.ProviderKind == "deepseek" && entry.Metadata.ContextWindow == nil {
			// 兼容已同步但尚未保存上下文元数据的旧快照，按实际路由目标补齐。
			entry.Metadata.ContextWindow = deepseek.ModelMetadata(entry.TargetModel).ContextWindow
		}
		if entry.ProviderKind == atria.Kind {
			entry.Metadata = atria.ModelMetadata(entry.TargetModel)
		}
		if entry.ProviderKind == mimo.Kind {
			entry.Metadata = mimo.ModelMetadata(entry.TargetModel)
		}
		if entry.ProviderKind == "google" {
			// 首版只有 Chat 无损透传，不能把尚未验证的跨协议 Agent 模型写入客户端目录。
			entry.Protocols = []string{"chat_completions"}
		}
		if entry.ProviderKind == minimax.Kind {
			entry.Metadata = minimax.ModelMetadata(entry.TargetModel)
		}
		if entry.ProviderKind == kimi.Kind {
			if entry.Metadata.ContextWindow == nil {
				entry.Metadata = kimi.ModelMetadata(entry.TargetModel)
			}
			entry.Protocols = []string{"chat_completions"}
			if kimi.Supports(entry.TargetModel, "messages") {
				entry.Protocols = append(entry.Protocols, "messages")
			}
			if kimi.Supports(entry.TargetModel, "responses") {
				entry.Protocols = append(entry.Protocols, "responses")
			}
		}
		if entry.ProviderKind == stepfun.Kind {
			entry.Metadata = stepfun.ModelMetadata(entry.TargetModel)
			entry.Protocols = []string{"chat_completions", "messages"}
			for _, a := range data.accounts {
				if a.ID == decision.AccountID && a.StepFun != nil && stepfun.Supports(*a.StepFun, entry.TargetModel, "responses") {
					entry.Protocols = append(entry.Protocols, "responses")
				}
			}
		}
		entries = append(entries, entry)
	}
	return catalog.Normalize(entries)
}
