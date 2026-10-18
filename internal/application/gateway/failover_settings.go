package gateway

import (
	"context"
	"database/sql"
	"omniproxy/internal/core/proxy"
)

// FailoverSettings 是持久化的同池切换策略；次数与成功边界由服务端固定。
type FailoverSettings struct {
	Enabled     bool `json:"enabled"`
	MaxAttempts int  `json:"maxAttempts"`
}

// FailoverSettingsService 保存当前应用的账号切换开关。
type FailoverSettingsService struct{ db *sql.DB }

// NewFailoverSettingsService 复用应用数据库，不缓存可能变更的开关。
func NewFailoverSettingsService(db *sql.DB) *FailoverSettingsService {
	return &FailoverSettingsService{db: db}
}

// Get 读取并验证持久化设置。
func (s *FailoverSettingsService) Get(ctx context.Context) (FailoverSettings, error) {
	settings := FailoverSettings{MaxAttempts: 3}
	var enabled int
	if s.db == nil || s.db.QueryRowContext(ctx, `SELECT enabled FROM gateway_failover_settings WHERE id=1`).Scan(&enabled) != nil || enabled < 0 || enabled > 1 {
		return settings, proxy.Failure("failover_settings_unavailable", 503)
	}
	settings.Enabled = enabled == 1
	return settings, nil
}

// Enabled 为网关请求提供当前策略，配置错误不能默认为开启。
func (s *FailoverSettingsService) Enabled(ctx context.Context) (bool, error) {
	settings, err := s.Get(ctx)
	return settings.Enabled, err
}

// Set 仅更新唯一的设置行，不覆盖其他网关选项。
func (s *FailoverSettingsService) Set(ctx context.Context, enabled bool) (FailoverSettings, error) {
	if s.db == nil {
		return FailoverSettings{}, proxy.Failure("failover_settings_unavailable", 503)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE gateway_failover_settings SET enabled=? WHERE id=1`, enabled)
	if err != nil {
		return FailoverSettings{}, proxy.Failure("failover_settings_unavailable", 503)
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return FailoverSettings{}, proxy.Failure("failover_settings_unavailable", 503)
	}
	return s.Get(ctx)
}
