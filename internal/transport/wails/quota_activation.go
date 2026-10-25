package wails

import (
	"context"
	"errors"

	"github.com/wailsapp/wails/v3/pkg/application"
	"omniproxy/internal/application/gateway"
	"omniproxy/internal/application/quotaactivation"
)

func init() {
	application.RegisterEvent[SubscriptionNotice]("subscription:quota-updated")
	application.RegisterEvent[quotaactivation.Status]("subscription:activation-updated")
}

// SetCodexQuotaAutoActivation 保存明确授权的自动激活开关，开启后唤醒后台检测。
func (s *AppService) SetCodexQuotaAutoActivation(enabled bool) (gateway.CodexSettings, error) {
	if enabled && s.quotaActivation == nil {
		return gateway.CodexSettings{}, errors.New("activation: unavailable")
	}
	settings, err := gateway.NewCodexSettingsService(s.db).SetAutoQuotaActivation(context.Background(), enabled)
	if err == nil && s.quotaActivation != nil {
		s.quotaActivation.SettingsChanged(enabled)
	}
	if err == nil && enabled {
		// 后台检测依赖进程存活：关闭窗口改为留在托盘，不再退出应用。
		s.ensureTrayCloseBehavior()
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

// bindActivationEvents 把每轮检测结果推给界面；事件只含计数与稳定错误码。
func (s *AppService) bindActivationEvents(app *application.App) {
	s.setActivationEmitter(func(status quotaactivation.Status) {
		app.Event.Emit("subscription:activation-updated", status)
	})
}

func (s *AppService) setActivationEmitter(emit func(quotaactivation.Status)) {
	s.activationMu.Lock()
	defer s.activationMu.Unlock()
	s.emitActivation = emit
}

func (s *AppService) notifyActivation(status quotaactivation.Status) {
	s.activationMu.Lock()
	emit := s.emitActivation
	s.activationMu.Unlock()
	if emit != nil {
		emit(status)
	}
}

// ensureBackgroundActivation 让已开启自动激活的用户在下次启动后继续保持后台检测。
func (s *AppService) ensureBackgroundActivation() {
	if s.db == nil {
		return
	}
	settings, err := gateway.NewCodexSettingsService(s.db).Read(context.Background())
	if err != nil || !settings.AutoQuotaActivation {
		return
	}
	s.ensureTrayCloseBehavior()
}
