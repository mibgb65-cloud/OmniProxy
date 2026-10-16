package wails

import (
	"context"
	"errors"
	"omniproxy/internal/application/routing"
)

// GetKimiSettings 读取独立计费及地区偏好，不读取凭据。
func (s *AppService) GetKimiSettings() (routing.KimiSettings, error) {
	return routing.New(s.db, nil, nil).GetKimiSettings(context.Background())
}

// SetKimiPreferredBillingMode 保存计费顺序并刷新受管目录。
func (s *AppService) SetKimiPreferredBillingMode(mode string) (routing.KimiSettings, error) {
	v, err := routing.New(s.db, nil, nil).SetKimiPreferredBillingMode(context.Background(), mode)
	if err == nil {
		s.refreshAgentCatalog()
	}
	return v, err
}

// SetKimiPreferredRegion 保存同计费方式内的地区顺序。
func (s *AppService) SetKimiPreferredRegion(region string) (routing.KimiSettings, error) {
	v, err := routing.New(s.db, nil, nil).SetKimiPreferredRegion(context.Background(), region)
	if err == nil {
		s.refreshAgentCatalog()
	}
	return v, err
}

// OpenKimiCodeConsole 只打开固定官方订阅控制台。
func (s *AppService) OpenKimiCodeConsole(id string) error {
	a, err := s.accountService()
	if err != nil {
		return err
	}
	item, err := a.Metadata(context.Background(), id)
	if err != nil || item.Kimi == nil || !item.Kimi.Valid() || item.Kimi.Mode() != "coding" {
		return errors.New("kimi: invalid_configuration")
	}
	if s.openAgentURL == nil {
		return errors.New("kimi: browser_unavailable")
	}
	return s.openAgentURL("https://www.kimi.com/code/console")
}
