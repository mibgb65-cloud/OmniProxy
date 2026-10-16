package wails

import (
	"context"
	"omniproxy/internal/application/routing"
)

// GetMiMoSettings 读取 MiMo 默认计费顺序，不访问凭据或上游模型。
func (s *AppService) GetMiMoSettings() (routing.MiMoSettings, error) {
	return routing.New(s.db, nil, nil).GetMiMoSettings(context.Background())
}

// SetMiMoPreferredBillingMode 保存默认顺序并沿用受管客户端目录刷新。
func (s *AppService) SetMiMoPreferredBillingMode(mode string) (routing.MiMoSettings, error) {
	settings, err := routing.New(s.db, nil, nil).SetMiMoPreferredBillingMode(context.Background(), mode)
	if err == nil {
		s.refreshAgentCatalog()
	}
	return settings, err
}
