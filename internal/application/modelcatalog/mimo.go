package modelcatalog

import (
	"context"
	"omniproxy/internal/application/account"

	"omniproxy/internal/infrastructure/codex"
	"omniproxy/internal/provider/mimo"
)

func (s *Service) syncMiMo(ctx context.Context, current account.Item, base string) ([]codex.Model, error) {
	if current.MiMo == nil {
		return nil, ErrUnavailable
	}
	base, err := current.MiMo.BaseURL(base)
	if err != nil {
		return nil, ErrUnavailable
	}
	models := []codex.Model{}
	err = s.accounts.UseAPIKeyAt(ctx, current.ID, current.UpdatedAt, func(key []byte) error {
		ids, err := s.mimo.DiscoverModels(ctx, base, key)
		for _, model := range ids {
			models = append(models, codex.Model{ID: model, Name: model, Metadata: mimo.ModelMetadata(model)})
		}
		return err
	})
	return models, err
}
