package wails

import (
	"context"

	"omniproxy/internal/application/provider"
	"omniproxy/internal/application/proxysettings"
)

// proxySettingsService 返回与出站分流共享的代理设置服务；数据库缺失时各方法返回
// configuration_unavailable，而不是让绑定层空指针崩溃。
func (s *AppService) proxySettingsService() *proxysettings.Service {
	if s.proxySettings != nil {
		return s.proxySettings
	}
	return proxysettings.NewService(s.db, s.secrets)
}

// GetProxySettings 返回不含密码的代理设置。
func (s *AppService) GetProxySettings() (proxysettings.Settings, error) {
	return s.proxySettingsService().Read(context.Background())
}

// SaveProxySettings 保存代理连接参数；密码只进凭据库。
func (s *AppService) SaveProxySettings(input proxysettings.SaveInput) (proxysettings.Settings, error) {
	service := s.proxySettingsService()
	settings, err := service.Save(context.Background(), input)
	if err != nil {
		return settings, err
	}
	// 保存后立即让出站分流读取新配置。
	service.Invalidate()
	return settings, nil
}

// TestProxySettings 探测给定端点可用的协议，不改变已保存设置。
func (s *AppService) TestProxySettings(input proxysettings.SaveInput) (proxysettings.TestResult, error) {
	return s.proxySettingsService().Test(context.Background(), input)
}

// DiscoverProxyPorts 探测本机常见代理端口（v2rayN / Clash / mihomo）。
func (s *AppService) DiscoverProxyPorts() ([]proxysettings.Discovered, error) {
	return s.proxySettingsService().Discover(context.Background())
}

// SetProviderProxyMode 设置单个提供商的出站策略：inherit | direct | proxy。
func (s *AppService) SetProviderProxyMode(id string, mode string) (provider.Configuration, error) {
	item, err := provider.NewService(s.db).SetProxyMode(context.Background(), id, mode)
	if err == nil {
		s.proxySettingsService().Invalidate()
	}
	return item, err
}
