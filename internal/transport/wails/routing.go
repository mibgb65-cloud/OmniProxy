package wails

import (
	"context"

	"omniproxy/internal/application/provider"
	routes "omniproxy/internal/application/routing"
	core "omniproxy/internal/core/routing"
)

func (s *AppService) routingService() (*routes.Service, error) {
	accounts, err := s.accountService()
	if err != nil {
		return nil, err
	}
	return routes.New(s.db, provider.NewService(s.db), accounts), nil
}

// ListRoutes 返回规则和配置诊断。
func (s *AppService) ListRoutes() ([]routes.Item, error) {
	service, err := s.routingService()
	if err != nil {
		return nil, err
	}
	return service.List(context.Background())
}

// CreateRoute 保存模型精确映射。
func (s *AppService) CreateRoute(input routes.Input) (core.Rule, error) {
	service, err := s.routingService()
	if err != nil {
		return core.Rule{}, err
	}
	item, err := service.Create(context.Background(), input)
	if err == nil {
		s.refreshAgentCatalog()
	}
	return item, err
}

// UpdateRoute 修改模型映射。
func (s *AppService) UpdateRoute(id string, input routes.Input) (core.Rule, error) {
	service, err := s.routingService()
	if err != nil {
		return core.Rule{}, err
	}
	item, err := service.Update(context.Background(), id, input)
	if err == nil {
		s.refreshAgentCatalog()
	}
	return item, err
}

// SetRouteEnabled 修改规则开关。
func (s *AppService) SetRouteEnabled(id string, enabled bool) (core.Rule, error) {
	service, err := s.routingService()
	if err != nil {
		return core.Rule{}, err
	}
	item, err := service.SetEnabled(context.Background(), id, enabled)
	if err == nil {
		s.refreshAgentCatalog()
	}
	return item, err
}

// DeleteRoute 只删除映射，不修改账号或提供商。
func (s *AppService) DeleteRoute(id string) error {
	service, err := s.routingService()
	if err != nil {
		return err
	}
	err = service.Delete(context.Background(), id)
	if err == nil {
		s.refreshAgentCatalog()
	}
	return err
}

// PreviewRoute 解析配置，不调用模型或读取凭证。
func (s *AppService) PreviewRoute(model string) (routes.Preview, error) {
	service, err := s.routingService()
	if err != nil {
		return routes.Preview{}, err
	}
	return service.Resolve(context.Background(), model)
}
