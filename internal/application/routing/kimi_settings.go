package routing

import (
	"context"
	"errors"
)

// KimiSettings 独立保存计费方式及同计费方式内的地区优先级。
type KimiSettings struct {
	PreferredBillingMode string `json:"preferredBillingMode"`
	PreferredRegion      string `json:"preferredRegion"`
}

// GetKimiSettings 缺失或损坏时明确失败，避免静默改变计费及地区。
func (s *Service) GetKimiSettings(ctx context.Context) (KimiSettings, error) {
	var v KimiSettings
	if s.db == nil || s.db.QueryRowContext(ctx, `SELECT preferred_billing_mode,preferred_region FROM kimi_settings WHERE id=1`).Scan(&v.PreferredBillingMode, &v.PreferredRegion) != nil || (v.PreferredBillingMode != "coding" && v.PreferredBillingMode != "payg") || (v.PreferredRegion != "cn" && v.PreferredRegion != "global") {
		return KimiSettings{}, errors.New("routing: kimi_settings_unavailable")
	}
	return v, nil
}

func (s *Service) saveKimiSetting(ctx context.Context, query, value string) (KimiSettings, error) {
	if s.db == nil {
		return KimiSettings{}, errors.New("routing: kimi_settings_unavailable")
	}
	r, err := s.db.ExecContext(ctx, query, value)
	if err != nil {
		return KimiSettings{}, errors.New("routing: kimi_settings_save_failed")
	}
	n, err := r.RowsAffected()
	if err != nil || n != 1 {
		return KimiSettings{}, errors.New("routing: kimi_settings_save_failed")
	}
	return s.GetKimiSettings(ctx)
}

// SetKimiPreferredBillingMode 只更新计费偏好，不覆盖并行保存的地区偏好。
func (s *Service) SetKimiPreferredBillingMode(ctx context.Context, mode string) (KimiSettings, error) {
	if mode != "coding" && mode != "payg" {
		return KimiSettings{}, ErrInvalidInput
	}
	return s.saveKimiSetting(ctx, `UPDATE kimi_settings SET preferred_billing_mode=? WHERE id=1`, mode)
}

// SetKimiPreferredRegion 只更新地区偏好，计费方式优先于地区。
func (s *Service) SetKimiPreferredRegion(ctx context.Context, region string) (KimiSettings, error) {
	if region != "cn" && region != "global" {
		return KimiSettings{}, ErrInvalidInput
	}
	return s.saveKimiSetting(ctx, `UPDATE kimi_settings SET preferred_region=? WHERE id=1`, region)
}
