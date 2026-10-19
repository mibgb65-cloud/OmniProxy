package quotaactivation

import (
	"context"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/gateway"
)

func (s *Service) enabled(ctx context.Context) (bool, error) {
	settings, err := gateway.NewCodexSettingsService(s.db).Read(ctx)
	if err != nil {
		return false, failure("settings_unavailable")
	}
	return settings.AutoQuotaActivation, nil
}

// Check 串行检测当前启用的订阅账号，不会向未授权或已启动窗口发送消息。
func (s *Service) Check(ctx context.Context) error {
	s.checkMu.Lock()
	defer s.checkMu.Unlock()
	enabled, err := s.enabled(ctx)
	if err != nil {
		return err
	}
	if !enabled || ctx.Err() != nil {
		return nil
	}
	batchCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.activeCancel = cancel
	s.status = Status{Running: true}
	s.changed = false
	s.mu.Unlock()
	defer func() {
		cancel()
		s.mu.Lock()
		s.activeCancel = nil
		s.status.Running = false
		s.status.UpdatedAt = s.now().UTC().Format(time.RFC3339)
		callback := s.onChange
		status, log := s.status, s.log
		changed := s.status.Sent > 0 || s.changed
		s.mu.Unlock()
		if log != nil {
			log.Info("Codex quota activation checked", "module", "quotaactivation", "checked", status.Checked, "sent", status.Sent, "activated", status.Activated, "pending", status.Pending, "failed", status.Failed, "code", status.ErrorCode)
		}
		if callback != nil && changed {
			callback()
		}
	}()
	providers, err := s.providers.List(batchCtx)
	if err != nil {
		return s.failed("accounts_unavailable")
	}
	allowed := map[string]bool{}
	for _, provider := range providers {
		if provider.Kind == "openai" && provider.Enabled {
			allowed[provider.ID] = true
		}
	}
	accounts, err := s.accounts.List(batchCtx)
	if err != nil {
		return s.failed("accounts_unavailable")
	}
	present := map[string]bool{}
	for _, item := range accounts {
		present[item.ID] = true
	}
	s.mu.Lock()
	for id := range s.observations {
		if !present[id] {
			delete(s.observations, id)
		}
	}
	s.mu.Unlock()
	for _, item := range accounts {
		enabled, err := s.enabled(batchCtx)
		if err != nil || !enabled || batchCtx.Err() != nil {
			return err
		}
		if !allowed[item.ProviderID] || !item.Enabled || item.AuthMethod != account.AuthSubscription || item.AuthState != "authenticated" || item.AuthAccountID == "" {
			continue
		}
		accountCtx, finish := context.WithTimeout(batchCtx, activationTimeout)
		s.mu.Lock()
		s.status.Checked++
		s.mu.Unlock()
		err = s.checkAccount(accountCtx, item)
		finish()
		if err != nil && batchCtx.Err() == nil {
			s.failed("activation_failed")
		}
	}
	return nil
}

func (s *Service) failed(code string) error {
	s.mu.Lock()
	s.status.Failed++
	s.status.ErrorCode = code
	s.mu.Unlock()
	return failure(code)
}
