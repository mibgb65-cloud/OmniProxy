package wails

import (
	"context"
	"errors"

	"github.com/wailsapp/wails/v3/pkg/application"
	"omniproxy/internal/application/gateway"
	"omniproxy/internal/application/quotaactivation"
)

func init() { application.RegisterEvent[SubscriptionNotice]("subscription:quota-updated") }

// SetCodexQuotaAutoActivation 保存明确授权的自动激活开关，开启后唤醒后台检测。
func (s *AppService) SetCodexQuotaAutoActivation(enabled bool) (gateway.CodexSettings, error) {
	if enabled && s.quotaActivation == nil {
		return gateway.CodexSettings{}, errors.New("activation: unavailable")
	}
	settings, err := gateway.NewCodexSettingsService(s.db).SetAutoQuotaActivation(context.Background(), enabled)
	if err == nil && s.quotaActivation != nil {
		s.quotaActivation.SettingsChanged(enabled)
	}
	return settings, err
}

// GetCodexQuotaActivationStatus 只返回计数与安全状态，不暴露账号身份或消息内容。
func (s *AppService) GetCodexQuotaActivationStatus() (quotaactivation.Status, error) {
	if s.quotaActivation == nil {
		return quotaactivation.Status{}, errors.New("activation: unavailable")
	}
	return s.quotaActivation.Status(), nil
}
