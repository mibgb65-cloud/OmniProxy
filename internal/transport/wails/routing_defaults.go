package wails

import (
	"context"

	routes "omniproxy/internal/application/routing"
)

// ListDefaultRoutes 返回目录派生的同名映射、服务商候选和当前实际选择。
func (s *AppService) ListDefaultRoutes() ([]routes.DefaultRoute, error) {
	service, err := s.routingService()
	if err != nil {
		return nil, err
	}
	return service.DefaultRoutes(context.Background())
}

// SetModelPreferredProvider 将服务商移到首位并保留后续优先级，空值恢复自动选择。
func (s *AppService) SetModelPreferredProvider(model, providerID string) error {
	service, err := s.routingService()
	if err != nil {
		return err
	}
	err = service.SetPreferredProvider(context.Background(), model, providerID)
	if err == nil {
		s.refreshAgentCatalog()
	}
	return err
}

// SetModelProviderOrder 保存同名模型的服务商优先级；空列表恢复自动选择。
func (s *AppService) SetModelProviderOrder(model string, providerIDs []string) error {
	service, err := s.routingService()
	if err != nil {
		return err
	}
	if err := service.SetProviderOrder(context.Background(), model, providerIDs); err != nil {
		return err
	}
	s.refreshAgentCatalog()
	return nil
}
