package modelcatalog

import (
	"context"
	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/codex"
	"omniproxy/internal/provider/minimax"
)

func (s *Service) syncMiniMax(ctx context.Context, current account.Item) ([]codex.Model, error) {
	if current.MiniMax == nil || !current.MiniMax.Valid() {
		return nil, ErrUnavailable
	}
	models := []codex.Model{}
	err := s.accounts.UseAPIKeyAt(ctx, current.ID, current.UpdatedAt, func(key []byte) error {
		found, err := s.minimax.DiscoverModels(ctx, *current.MiniMax, key)
		for _, m := range found {
			models = append(models, codex.Model{ID: m.ID, Name: m.ID, Metadata: m.Metadata})
		}
		return err
	})
	return models, err
}

func (s *Service) minimaxPresets(ctx context.Context, cached []Item) ([]Item, error) {
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
	for _, m := range cached {
		synced[m.AccountID] = true
	}
	items := []Item{}
	for _, a := range accounts {
		if !minimaxProviders[a.ProviderID] || a.AuthMethod != account.AuthAPIKey || a.MiniMax == nil || !a.MiniMax.Valid() || synced[a.ID] {
			continue
		}
		for _, id := range minimax.AccountModels(*a.MiniMax) {
			items = append(items, Item{AccountID: a.ID, ProviderID: a.ProviderID, Model: id, Name: id, Source: "preset", Metadata: minimax.ModelMetadata(id)})
		}
		if len(items) > 10000 {
			return nil, ErrUnavailable
		}
	}
	return items, nil
}
