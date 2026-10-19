package wails

import (
	"context"
	"omniproxy/internal/application/gateway"
)

// GetCodexSettings 读取当前 OmniProxy 网关的 Codex WebSocket 开关。
func (s *AppService) GetCodexSettings() (gateway.CodexSettings, error) {
	return gateway.NewCodexSettingsService(s.db).Read(context.Background())
}

// SetCodexWebSocketEnabled 保存全局开关，不改变 Codex 提供商身份或客户端文件。
func (s *AppService) SetCodexWebSocketEnabled(enabled bool) (gateway.CodexSettings, error) {
	return gateway.NewCodexSettingsService(s.db).SetWebSocketEnabled(context.Background(), enabled)
}

// SetCodexAccountRotationMode 保存 OpenAI 同池账号轮换方式。
func (s *AppService) SetCodexAccountRotationMode(mode string) (gateway.CodexSettings, error) {
	return gateway.NewCodexSettingsService(s.db).SetAccountRotationMode(context.Background(), mode)
}
