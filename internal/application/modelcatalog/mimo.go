package modelcatalog

import (
	"context"

	"omniproxy/internal/infrastructure/codex"
	"omniproxy/internal/provider/mimo"
)

func (s *Service) syncMiMo(ctx context.Context, id, base string) ([]codex.Model, error) {
	models := []codex.Model{}
	err := s.accounts.UseAPIKey(ctx, id, func(key []byte) error {
		ids, err := mimo.New(nil).DiscoverModels(ctx, base, key)
		for _, model := range ids {
			models = append(models, codex.Model{ID: model, Name: model, Metadata: mimo.ModelMetadata(model)})
		}
		return err
	})
	return models, err
}
