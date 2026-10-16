package routing

import (
	"context"

	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/provider/atria"
	"omniproxy/internal/provider/deepseek"
	"omniproxy/internal/provider/kimi"
	"omniproxy/internal/provider/mimo"
	"omniproxy/internal/provider/minimax"
	"omniproxy/internal/provider/river"
	"omniproxy/internal/provider/stepfun"
)

// Catalog 使用实际路由决策的账号元数据，不把其他同名模型的能力混入目标。
func (s *Service) Catalog(ctx context.Context) ([]catalog.Entry, error) {
	return s.catalog(ctx, false)
}

// CompatibilityCatalog 仅供旧 Claude 上下文／桌面编码 ID 解析，不重新公布通道重复项。
func (s *Service) CompatibilityCatalog(ctx context.Context) ([]catalog.Entry, error) {
	return s.catalog(ctx, true)
}

func (s *Service) catalog(ctx context.Context, legacyChannels bool) ([]catalog.Entry, error) {
	data, err := s.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := s.rules(ctx)
	if err != nil {
		return nil, err
	}
	ids := originalModels(data)
	if legacyChannels {
		for _, model := range data.models {
			if publicModel(data, model.providerID, model.model) != model.model {
				ids = append(ids, model.model)
			}
		}
	}
	// MiMo／StepFun 同名模型按设置解析；旧通道只兼容解析，不自动发布副本。
	rules = append(rules, minimaxAliases(data)...)
	if legacyChannels {
		ids = append(ids, "k3", "kimi-for-coding-highspeed")
		rules = append(rules, kimiAliases(data)...)
		rules = append(rules, stepAliases(data)...)
		rules = append(rules, mimoAliases(data)...)
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
		if entry.ProviderKind == river.Kind {
			entry.Metadata = river.ModelMetadata(entry.TargetModel)
			if !matched {
				entry.Name = river.ModelName(entry.TargetModel)
			}
		}
		if entry.ProviderKind == atria.Kind {
			entry.Metadata = atria.ModelMetadata(entry.TargetModel)
		}
		if entry.ProviderKind == mimo.Kind {
			entry.Metadata = mimo.ModelMetadata(entry.TargetModel)
		}
		if entry.ProviderKind == "google" {
			entry.Protocols = []string{"chat_completions"}
			for _, a := range data.accounts {
				if a.ID == decision.AccountID && a.AuthMethod == "subscription_login" {
					entry.Protocols = append(entry.Protocols, "responses", "messages")
					break
				}
			}
		}
		if entry.ProviderKind == minimax.Kind {
			entry.Metadata = minimax.ModelMetadata(entry.TargetModel)
		}
		if entry.ProviderKind == kimi.Kind {
			if entry.Metadata.ContextWindow == nil {
				entry.Metadata = kimi.ModelMetadata(entry.TargetModel)
			}
			entry.Protocols = []string{"chat_completions"}
			for _, a := range data.accounts {
				if a.ID == decision.AccountID && a.Kimi != nil {
					if kimi.SupportsAccount(*a.Kimi, entry.TargetModel, "messages") {
						entry.Protocols = append(entry.Protocols, "messages")
					}
					if kimi.SupportsAccount(*a.Kimi, entry.TargetModel, "responses") {
						entry.Protocols = append(entry.Protocols, "responses")
					}
					if a.Kimi.Mode() == "coding" {
						entry.Metadata = kimi.AccountModelMetadata(*a.Kimi, entry.TargetModel)
					}
					break
				}
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
