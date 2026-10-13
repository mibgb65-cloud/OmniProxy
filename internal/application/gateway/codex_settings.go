package gateway

import (
	"context"
	"database/sql"
	"errors"
)

// CodexSettings 是当前 OmniProxy 网关的应用级 Codex 接入设置。
type CodexSettings struct {
	WebSocketEnabled    bool `json:"webSocketEnabled"`
	AutoQuotaActivation bool `json:"autoQuotaActivation"`
}

// CodexSettingsService 持久化非敏感开关，不读取或修改客户端身份、认证和会话。
type CodexSettingsService struct{ db *sql.DB }

// NewCodexSettingsService 创建独立于桌面适配的设置服务。
func NewCodexSettingsService(db *sql.DB) *CodexSettingsService { return &CodexSettingsService{db: db} }

// Read 读取唯一设置记录；缺失或损坏必须报错，不能静默改变网关策略。
func (s *CodexSettingsService) Read(ctx context.Context) (CodexSettings, error) {
	var enabled, activation int
	if s.db == nil || s.db.QueryRowContext(ctx, `SELECT websocket_enabled,auto_quota_activation FROM codex_settings WHERE id=1`).Scan(&enabled, &activation) != nil || (enabled != 0 && enabled != 1) || (activation != 0 && activation != 1) {
		return CodexSettings{}, errors.New("gateway: configuration_unavailable")
	}
	return CodexSettings{WebSocketEnabled: enabled == 1, AutoQuotaActivation: activation == 1}, nil
}

// SetWebSocketEnabled 原子保存网关 WebSocket 开关，立即影响新握手和后续会话请求。
func (s *CodexSettingsService) SetWebSocketEnabled(ctx context.Context, enabled bool) (CodexSettings, error) {
	if s.db == nil {
		return CodexSettings{}, errors.New("gateway: configuration_unavailable")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE codex_settings SET websocket_enabled=? WHERE id=1`, enabled)
	if err != nil {
		return CodexSettings{}, errors.New("gateway: settings_save_failed")
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return CodexSettings{}, errors.New("gateway: settings_save_failed")
	}
	return s.Read(ctx)
}

// SetAutoQuotaActivation 保存用户授权的自动额度激活设置，不改变 WebSocket 开关。
func (s *CodexSettingsService) SetAutoQuotaActivation(ctx context.Context, enabled bool) (CodexSettings, error) {
	if s.db == nil {
		return CodexSettings{}, errors.New("gateway: configuration_unavailable")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE codex_settings SET auto_quota_activation=? WHERE id=1`, enabled)
	if err != nil {
		return CodexSettings{}, errors.New("gateway: settings_save_failed")
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return CodexSettings{}, errors.New("gateway: settings_save_failed")
	}
	return s.Read(ctx)
}

// WebSocketAllowed 供代理核心按请求读取当前持久化策略。
func (s *CodexSettingsService) WebSocketAllowed(ctx context.Context) (bool, error) {
	settings, err := s.Read(ctx)
	return settings.WebSocketEnabled, err
}
