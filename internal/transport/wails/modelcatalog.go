package wails

import (
	"context"
	"omniproxy/internal/application/modelcatalog"
	"omniproxy/internal/application/provider"
)

func (s *AppService) modelCatalog() (*modelcatalog.Service, error) {
	accounts, err := s.accountService()
	if err != nil {
		return nil, err
	}
	subscription, err := s.subscriptionService()
	if err != nil {
		return nil, err
	}
	return modelcatalog.New(s.db, accounts, provider.NewService(s.db), subscription), nil
}
func (s *AppService) ModelCatalog() ([]modelcatalog.Item, error) {
	service, err := s.modelCatalog()
	if err != nil {
		return nil, err
	}
	return service.List(context.Background())
}
func (s *AppService) ModelSync(id string) ([]modelcatalog.Item, error) {
	service, err := s.modelCatalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := subscriptionContext()
	defer cancel()
	items, err := service.Sync(ctx, id)
	if err == nil {
		s.refreshAgentCatalog()
	}
	return items, err
}
