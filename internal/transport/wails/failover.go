package wails

import (
	"context"
	"omniproxy/internal/application/gateway"
	"omniproxy/internal/core/proxy"
)

// GatewayFailoverSettings 读取同池切换开关，不触发上游请求。
func (s *AppService) GatewayFailoverSettings() (gateway.FailoverSettings, error) {
	return gateway.NewFailoverSettingsService(s.db).Get(context.Background())
}

// GatewaySetFailoverEnabled 保存后对后续请求生效，不改变已在途请求的策略。
func (s *AppService) GatewaySetFailoverEnabled(enabled bool) (gateway.FailoverSettings, error) {
	return gateway.NewFailoverSettingsService(s.db).Set(context.Background(), enabled)
}

// GatewayFailoverHealth 只读当前运行状态，不修改账号启用或授权状态。
func (s *AppService) GatewayFailoverHealth() ([]gateway.AccountHealth, error) {
	if s.gateway == nil {
		return nil, proxy.Failure("invalid_configuration", 503)
	}
	return s.gateway.FailoverHealth(), nil
}
