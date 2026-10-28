package wails

import (
	"context"

	"omniproxy/internal/platform/profile"
)

func (s *AppService) startGatewayOnStartup() error {
	cfg, err := profile.LoadConfig(s.prof.Kind)
	if err != nil {
		return err
	}
	if !cfg.GatewayAutoStart {
		return nil
	}
	if _, err := s.GatewayStart(); err != nil {
		if !isPortUnavailable(err) || profile.ProxyPortEnvOverride() {
			return err
		}
		// 自动启动时端口被其它程序占用：避让到同区间内下一个可用端口，
		// 持久化新端口并同步受管客户端地址，而不是让用户先自行处理占用。
		s.settingsMu.Lock()
		_, avoidErr := s.applyProxyPort(context.Background(), s.prof.ProxyPort)
		s.settingsMu.Unlock()
		if avoidErr != nil {
			return err
		}
		if _, err := s.GatewayStart(); err != nil {
			return err
		}
		if s.log != nil {
			s.log.Warn("gateway port was unavailable; started on an alternative port",
				"module", "gateway", "code", "port_avoided", "port", s.prof.ProxyPort)
		}
	}
	return nil
}
