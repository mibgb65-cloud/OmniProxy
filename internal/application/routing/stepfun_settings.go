package routing

import (
	"context"
	"errors"
)

// StepFunSettings 是所有 StepFun 提供商共用的默认计费通道顺序。
type StepFunSettings struct {
	PreferredBillingMode string `json:"preferredBillingMode"`
}

func validStepFunBillingMode(mode string) bool { return mode == "step_plan" || mode == "payg" }

// GetStepFunSettings 读取持久设置，缺失或损坏时明确失败，避免静默改变计费通道。
func (s *Service) GetStepFunSettings(ctx context.Context) (StepFunSettings, error) {
	var settings StepFunSettings
	if s.db == nil || s.db.QueryRowContext(ctx, `SELECT preferred_billing_mode FROM stepfun_settings WHERE id=1`).Scan(&settings.PreferredBillingMode) != nil || !validStepFunBillingMode(settings.PreferredBillingMode) {
		return StepFunSettings{}, errors.New("routing: stepfun_settings_unavailable")
	}
	return settings, nil
}

// SetStepFunPreferredBillingMode 保存首选计费通道，下一次路由解析立即读取新顺序。
func (s *Service) SetStepFunPreferredBillingMode(ctx context.Context, mode string) (StepFunSettings, error) {
	if !validStepFunBillingMode(mode) {
		return StepFunSettings{}, ErrInvalidInput
	}
	if s.db == nil {
		return StepFunSettings{}, errors.New("routing: stepfun_settings_unavailable")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE stepfun_settings SET preferred_billing_mode=? WHERE id=1`, mode)
	if err != nil {
		return StepFunSettings{}, errors.New("routing: stepfun_settings_save_failed")
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return StepFunSettings{}, errors.New("routing: stepfun_settings_save_failed")
	}
	return s.GetStepFunSettings(ctx)
}
