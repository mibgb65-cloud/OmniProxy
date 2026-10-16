package routing

import (
	"context"
	"errors"
)

// MiMoSettings 是所有 MiMo 提供商共用的默认计费通道顺序。
type MiMoSettings struct {
	PreferredBillingMode string `json:"preferredBillingMode"`
}

func validMiMoBillingMode(mode string) bool { return mode == "token_plan" || mode == "payg" }

// GetMiMoSettings 读取持久设置，缺失或损坏时明确失败，避免静默改变计费通道。
func (s *Service) GetMiMoSettings(ctx context.Context) (MiMoSettings, error) {
	var settings MiMoSettings
	if s.db == nil || s.db.QueryRowContext(ctx, `SELECT preferred_billing_mode FROM mimo_settings WHERE id=1`).Scan(&settings.PreferredBillingMode) != nil || !validMiMoBillingMode(settings.PreferredBillingMode) {
		return MiMoSettings{}, errors.New("routing: mimo_settings_unavailable")
	}
	return settings, nil
}

// SetMiMoPreferredBillingMode 保存首选计费通道，下一次路由解析立即读取新顺序。
func (s *Service) SetMiMoPreferredBillingMode(ctx context.Context, mode string) (MiMoSettings, error) {
	if !validMiMoBillingMode(mode) {
		return MiMoSettings{}, ErrInvalidInput
	}
	if s.db == nil {
		return MiMoSettings{}, errors.New("routing: mimo_settings_unavailable")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE mimo_settings SET preferred_billing_mode=? WHERE id=1`, mode)
	if err != nil {
		return MiMoSettings{}, errors.New("routing: mimo_settings_save_failed")
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return MiMoSettings{}, errors.New("routing: mimo_settings_save_failed")
	}
	return s.GetMiMoSettings(ctx)
}
