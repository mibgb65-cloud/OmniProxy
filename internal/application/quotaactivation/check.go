package quotaactivation

import (
	"context"
	"errors"
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
		changed := s.status.Sent > 0 || s.changed
		s.mu.Unlock()
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

func (s *Service) checkAccount(ctx context.Context, item account.Item) error {
	quota, err := s.quotas.ReadQuota(ctx, item.ID)
	if err != nil {
		return err
	}
	if err := s.observe(ctx, item, quota); err != nil {
		return err
	}
	if err := s.reportQuota(item, quota); err != nil {
		return err
	}
	if !NeedsActivation(quota) {
		if windowsActive(quota, s.now()) {
			s.mu.Lock()
			s.status.Activated++
			s.mu.Unlock()
		}
		return nil
	}
	claimed, err := s.claim(ctx, item, quota)
	if err != nil {
		return err
	}
	if len(claimed) == 0 {
		s.mu.Lock()
		s.status.Pending++
		s.mu.Unlock()
		return nil
	}
	enabled, err := s.enabled(ctx)
	if err != nil || !enabled || ctx.Err() != nil {
		return errors.Join(err, s.release(ctx, item, claimed))
	}
	current, err := s.accounts.Metadata(ctx, item.ID)
	if err != nil || !current.Enabled || current.AuthState != "authenticated" || identityVersion(current) != identityVersion(item) {
		return errors.Join(failure("identity_changed"), s.release(ctx, item, claimed))
	}
	// 网络结果不确定时保留登记，不能因复查失败或重启再消耗同一窗口的额度。
	sendErr := s.sender.Send(ctx, item)
	if errors.Is(sendErr, ErrNotSent) {
		return errors.Join(failure("prepare_failed"), s.release(ctx, item, claimed))
	}
	if sendErr == nil {
		s.mu.Lock()
		s.status.Sent++
		s.mu.Unlock()
	}
	fresh, readErr := s.quotas.ReadQuota(ctx, item.ID)
	if readErr == nil {
		current, err := s.accounts.Metadata(ctx, item.ID)
		if err != nil || identityVersion(current) != identityVersion(item) {
			return failure("identity_changed")
		}
		if err := s.observe(ctx, item, fresh); err != nil {
			return err
		}
		if err := s.reportQuota(item, fresh); err != nil {
			return err
		}
		if activationConfirmed(quota, fresh, s.now()) {
			s.mu.Lock()
			s.status.Activated++
			s.mu.Unlock()
			return nil
		}
	}
	s.mu.Lock()
	s.status.Pending++
	s.mu.Unlock()
	if sendErr != nil && !errors.Is(sendErr, ErrAlreadyActive) {
		return failure("request_failed")
	}
	return nil
}
