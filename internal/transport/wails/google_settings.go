package wails

import (
	"context"

	"omniproxy/internal/application/gateway"
)

// GetGoogleSettings 读取 Google 订阅账号的同池轮换设置。
func (s *AppService) GetGoogleSettings() (gateway.GoogleSettings, error) {
	return gateway.NewGoogleSettingsService(s.db).Read(context.Background())
}

// SetGoogleAccountRotationMode 保存 Google 订阅账号的轮换方式，立即影响新请求。
func (s *AppService) SetGoogleAccountRotationMode(mode string) (gateway.GoogleSettings, error) {
	return gateway.NewGoogleSettingsService(s.db).SetAccountRotationMode(context.Background(), mode)
}
