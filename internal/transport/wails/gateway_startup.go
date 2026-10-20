package wails

import "omniproxy/internal/platform/profile"

func (s *AppService) startGatewayOnStartup() error {
	cfg, err := profile.LoadConfig(s.prof.Kind)
	if err != nil {
		return err
	}
	if !cfg.GatewayAutoStart {
		return nil
	}
	_, err = s.GatewayStart()
	return err
}
