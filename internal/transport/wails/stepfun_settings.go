package wails

import (
	"context"
	"omniproxy/internal/application/routing"
)

// GetStepFunSettings 读取 StepFun 默认计费顺序，不访问凭据或上游模型。
func (s *AppService) GetStepFunSettings() (routing.StepFunSettings, error) {
	return routing.New(s.db, nil, nil).GetStepFunSettings(context.Background())
}

// SetStepFunPreferredBillingMode 保存默认顺序并沿用受管客户端目录刷新。
func (s *AppService) SetStepFunPreferredBillingMode(mode string) (routing.StepFunSettings, error) {
	settings, err := routing.New(s.db, nil, nil).SetStepFunPreferredBillingMode(context.Background(), mode)
	if err == nil {
		s.refreshAgentCatalog()
	}
	return settings, err
}
