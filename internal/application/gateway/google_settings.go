package gateway

import (
	"context"
	"database/sql"

	"omniproxy/internal/core/proxy"
)

// GoogleSettings 是 Google AI 订阅账号的接入设置，不包含凭据或消息内容。
type GoogleSettings struct {
	AccountRotationMode string `json:"accountRotationMode"`
}

// GoogleSettingsService 持久化非敏感开关，不读取或修改账号授权。
type GoogleSettingsService struct{ db *sql.DB }

// NewGoogleSettingsService 创建独立于桌面适配的设置服务。
func NewGoogleSettingsService(db *sql.DB) *GoogleSettingsService {
	return &GoogleSettingsService{db: db}
}

// Read 读取轮换方式；缺行或取值不可用时返回稳定错误码，不猜测默认值。
func (s *GoogleSettingsService) Read(ctx context.Context) (GoogleSettings, error) {
	var value GoogleSettings
	if s.db == nil || s.db.QueryRowContext(ctx, `SELECT account_rotation_mode FROM google_settings WHERE id=1`).Scan(&value.AccountRotationMode) != nil ||
		!validRotationMode(value.AccountRotationMode) {
		return GoogleSettings{}, proxy.Failure("configuration_unavailable", 503)
	}
	return value, nil
}

// SetAccountRotationMode 只更新轮换方式，拒绝未知值。
func (s *GoogleSettingsService) SetAccountRotationMode(ctx context.Context, mode string) (GoogleSettings, error) {
	if !validRotationMode(mode) {
		return GoogleSettings{}, proxy.Failure("invalid_rotation_mode", 400)
	}
	if s.db == nil {
		return GoogleSettings{}, proxy.Failure("configuration_unavailable", 503)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE google_settings SET account_rotation_mode=? WHERE id=1`, mode)
	if err != nil {
		return GoogleSettings{}, proxy.Failure("settings_save_failed", 503)
	}
	if count, err := result.RowsAffected(); err != nil || count != 1 {
		return GoogleSettings{}, proxy.Failure("settings_save_failed", 503)
	}
	return s.Read(ctx)
}

// AccountRotationMode 只返回方式字符串，供网关每次请求读取以支持热切换。
func (s *GoogleSettingsService) AccountRotationMode(ctx context.Context) (string, error) {
	value, err := s.Read(ctx)
	return value.AccountRotationMode, err
}
