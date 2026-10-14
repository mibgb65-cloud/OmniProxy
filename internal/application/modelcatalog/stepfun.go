package modelcatalog

import (
	"context"
	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/codex"
	"omniproxy/internal/provider/stepfun"
)

func (s *Service) syncStep(ctx context.Context, current account.Item) ([]codex.Model, error) {
	if current.StepFun == nil || !current.StepFun.Valid() || current.StepFun.BillingMode != "payg" {
		return nil, ErrUnavailable
	}
	models := []codex.Model{}
	err := s.accounts.UseAPIKeyAt(ctx, current.ID, current.UpdatedAt, func(key []byte) error {
		ids, err := s.stepfun.DiscoverModels(ctx, *current.StepFun, key)
		for _, id := range ids {
			models = append(models, codex.Model{ID: id, Name: id, Metadata: stepfun.ModelMetadata(id)})
		}
		return err
	})
	return models, err
}

func (s *Service) stepPresets(ctx context.Context, cached []Item) ([]Item, error) {
	accounts, err := s.accounts.List(ctx)
	if err != nil {
		return nil, err
	}
	synced := map[string]bool{}
	for _, item := range cached {
		synced[item.AccountID] = true
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
	items := []Item{}
	for _, a := range accounts {
		if !stepProviders[a.ProviderID] || a.AuthMethod != account.AuthAPIKey || a.StepFun == nil || !a.StepFun.Valid() || synced[a.ID] {
			continue
		}
		for _, id := range stepfun.AccountModels(*a.StepFun) {
			items = append(items, Item{AccountID: a.ID, ProviderID: a.ProviderID, Model: id, Name: id, Source: "preset", Metadata: stepfun.ModelMetadata(id)})
		}
		if len(items) > 10000 {
			return nil, ErrUnavailable
		}
	}
	return items, nil
}
