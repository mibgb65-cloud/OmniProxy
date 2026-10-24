package quotaactivation

import (
	"context"
	"errors"
	"omniproxy/internal/application/account"
)

func (s *Service) checkAccount(ctx context.Context, item account.Item) error {
	rawQuota, err := s.quotas.ReadQuota(ctx, item.ID)
	if err != nil {
		// 额度读取失败属于可重试原因，但仍要给状态与日志一个稳定分类。
		return errors.Join(failure("quota_unavailable"), err)
	}
	assessed, err := s.observe(ctx, item, rawQuota)
	if err != nil {
		return err
	}
	quota := assessed.quota
	if err := s.reportQuota(item, rawQuota); err != nil {
		return err
	}
	if !NeedsActivation(quota) {
		if assessed.waiting {
			s.mu.Lock()
			s.status.Pending++
			s.mu.Unlock()
		} else if windowsActive(quota, s.now()) {
			s.mu.Lock()
			s.status.Activated++
			s.mu.Unlock()
		}
		return nil
	}
	claimed, err := s.claim(ctx, item, quota, assessed.floating)
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
	sendErr := s.sender.Send(ctx, item, buildSendPlan(rawQuota, claimed, assessed.floating))
	if errors.Is(sendErr, ErrNotSent) {
		releaseErr := s.release(ctx, item, claimed)
		if errors.Is(sendErr, ErrAwaitingQuota) {
			s.mu.Lock()
			s.status.Pending++
			s.mu.Unlock()
			return releaseErr
		}
		return errors.Join(prepareFailure(sendErr), releaseErr)
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
		freshAssessment, err := s.observe(ctx, item, fresh)
		if err != nil {
			return err
		}
		if err := s.reportQuota(item, fresh); err != nil {
			return err
		}
		if activationConfirmed(quota, freshAssessment.quota, s.now()) {
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
