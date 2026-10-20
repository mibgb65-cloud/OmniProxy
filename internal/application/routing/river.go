package routing

import (
	"context"
	"omniproxy/internal/application/account"
	core "omniproxy/internal/core/routing"
	"omniproxy/internal/provider/river"
)

func (s *Service) withRiverPresets(ctx context.Context, models []discoveredModel) ([]discoveredModel, error) {
	accounts, err := s.accounts.List(ctx)
	if err != nil {
		return nil, err
	}
	providers, err := s.providers.List(ctx)
	if err != nil {
		return nil, err
	}
	synced := map[string]bool{}
	for _, m := range models {
		synced[m.accountID] = true
	}
	for _, p := range providers {
		if p.Kind != river.Kind {
			continue
		}
		// 旧快照保留实际可用型号，但显示名随官方映射更新，不必改写持久化目录。
		for i := range models {
			if models[i].providerID == p.ID && river.SupportsModel(models[i].model) {
				models[i].name = river.ModelName(models[i].model)
			}
		}
		for _, a := range accounts {
			if a.ProviderID != p.ID || a.AuthMethod != account.AuthAPIKey || synced[a.ID] {
				continue
			}
			for _, id := range river.Models() {
				models = append(models, discoveredModel{accountID: a.ID, providerID: p.ID, model: id, name: river.ModelName(id), metadata: river.ModelMetadata(id)})
			}
		}
	}
	if len(models) > 10000 {
		return nil, core.ErrInvalidInput
	}
	return models, nil
}

func riverTargetSupports(data snapshot, providerID, accountID, model string) bool {
	model = river.UpstreamModel(model)
	if !river.SupportsModel(model) {
		return false
	}
	for _, m := range data.models {
		if m.providerID == providerID && m.model == model && (accountID == "" || m.accountID == accountID) {
			return true
		}
	}
	return false
}

func publicModel(data snapshot, providerID, model string) string {
	for _, p := range data.providers {
		if p.ID == providerID && p.Kind == river.Kind {
			return river.PublicModel(model)
		}
	}
	return model
}

func upstreamModel(data snapshot, providerID, model string) string {
	for _, p := range data.providers {
		if p.ID == providerID && p.Kind == river.Kind {
			return river.UpstreamModel(model)
		}
	}
	return model
}
