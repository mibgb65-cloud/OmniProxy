package gateway

import (
	"context"
	"errors"
	"net/http"

	"omniproxy/internal/core/proxy"
)

// passthroughSession 按候选顺序取号：明确拒绝就换号，成功即绑定当前账号。
// 与协议路径的差别只在负载形态，账号冷却与租约规则完全复用同一份状态。
type passthroughSession struct {
	service *Service
	request proxy.Request
	initial target
	current target
	lease   healthLease
	seen    map[string]bool
	model   string
	base    string
	fixed   bool
	enabled bool
	// credentialFailed 记录本会话是否有账号借不出令牌：这种情况不是「账号冷却」，
	// 报告成 429 会让客户端以为稍后重试即可，实际上凭据根本不可用。
	credentialFailed bool
	prepareTimedOut  bool
}

func (s *Service) newPassthroughSession(ctx context.Context, initial target, plan proxy.Request, model, base string, fixed bool) (proxy.PassthroughSession, error) {
	enabled := true
	if s.failoverEnabled != nil {
		value, err := s.failoverEnabled(ctx)
		if err != nil {
			return nil, err
		}
		enabled = value
	}
	return &passthroughSession{
		service: s, request: plan, initial: initial, model: model,
		base: base, fixed: fixed, enabled: enabled, seen: map[string]bool{},
	}, nil
}

// Next 返回下一个可用账号的尝试；固定账号不遍历候选，与显式选择一致。
func (s *passthroughSession) Next(ctx context.Context) (proxy.PassthroughAttempt, error) {
	if err := ctx.Err(); err != nil {
		return proxy.PassthroughAttempt{}, err
	}
	if s.fixed {
		attempt, available, err := s.attemptFor(ctx, proxy.AccountCandidate{AccountID: s.initial.accountID, UpdatedAt: s.initial.updatedAt})
		if err != nil {
			return proxy.PassthroughAttempt{}, err
		}
		if !available {
			if s.prepareTimedOut {
				return proxy.PassthroughAttempt{}, proxy.Failure("credential_prepare_timeout", 504)
			}
			if s.credentialFailed {
				return proxy.PassthroughAttempt{}, proxy.Failure("credential_unavailable", 503)
			}
			return proxy.PassthroughAttempt{}, proxy.Failure("route_account_unavailable", 503)
		}
		return attempt, nil
	}
	for _, candidate := range s.request.Candidates {
		if err := ctx.Err(); err != nil {
			return proxy.PassthroughAttempt{}, err
		}
		if s.seen[candidate.AccountID] {
			continue
		}
		s.seen[candidate.AccountID] = true
		attempt, available, err := s.tryCandidate(ctx, candidate)
		if err != nil {
			return proxy.PassthroughAttempt{}, err
		}
		if available {
			return attempt, nil
		}
	}
	return proxy.PassthroughAttempt{}, s.unavailable()
}

// unavailable 区分「全部账号冷却」与「借不出凭据」两种原因，不把后者伪装成限流。
func (s *passthroughSession) unavailable() error {
	if s.prepareTimedOut {
		return proxy.Failure("credential_prepare_timeout", 504)
	}
	if s.credentialFailed {
		return proxy.Failure("credential_unavailable", 503)
	}
	return &proxy.Error{Code: "account_pool_cooling", StatusCode: 429, RetryAfter: "1"}
}

// tryCandidate 逐项复核冷却、额度窗口与同池约束后才借出令牌。
func (s *passthroughSession) tryCandidate(ctx context.Context, candidate proxy.AccountCandidate) (proxy.PassthroughAttempt, bool, error) {
	if s.enabled {
		if delay := s.service.failover.wait(candidate.AccountID, candidate.UpdatedAt, s.model); delay > 0 {
			return proxy.PassthroughAttempt{}, false, nil
		}
	}
	return s.attemptFor(ctx, candidate)
}

// attemptFor 绑定账号版本后借出一次令牌；凭据不可用只跳过该账号。
func (s *passthroughSession) attemptFor(ctx context.Context, candidate proxy.AccountCandidate) (proxy.PassthroughAttempt, bool, error) {
	current := s.request
	current.AccountID, current.AccountUpdatedAt = candidate.AccountID, candidate.UpdatedAt
	t, err := s.service.selectRoutedTarget(ctx, current)
	if err != nil {
		if ctx.Err() != nil {
			return proxy.PassthroughAttempt{}, false, ctx.Err()
		}
		return proxy.PassthroughAttempt{}, false, err
	}
	if !sameTargetPool(s.initial, t) {
		return proxy.PassthroughAttempt{}, false, proxy.Failure("route_configuration_changed", 503)
	}
	if s.enabled && !s.fixed {
		if s.service.subscriptionQuotaWait(t, s.service.failover.now()) > 0 {
			return proxy.PassthroughAttempt{}, false, nil
		}
		lease, available, _ := s.service.failover.acquire(t, s.model)
		if !available {
			return proxy.PassthroughAttempt{}, false, nil
		}
		s.lease = lease
	}
	prepareCtx, cancel := proxy.SocketCredentialContext(ctx)
	stage := proxy.BeginSocketStage(prepareCtx, "credential_prepare", max(1, len(s.seen)))
	attempt, tokenErr := s.service.passthroughAttempt(prepareCtx, t, s.model, s.base)
	stage(tokenErr)
	prepareErr := prepareCtx.Err()
	cancel()
	if ctx.Err() != nil || prepareErr != nil {
		discardAttemptHeaders(attempt.Header)
		s.release(nil, false)
		if ctx.Err() != nil {
			return proxy.PassthroughAttempt{}, false, ctx.Err()
		}
		s.prepareTimedOut = true
		return proxy.PassthroughAttempt{}, false, nil
	}
	if tokenErr != nil {
		// 取消不能被折叠成「凭据损坏」，也不能消耗下一个账号。
		var failure *proxy.Error
		if errors.Is(tokenErr, context.DeadlineExceeded) ||
			errors.As(tokenErr, &failure) && failure.Code == "credential_prepare_timeout" {
			s.prepareTimedOut = true
			s.release(nil, false)
			return proxy.PassthroughAttempt{}, false, nil
		}
		if errors.Is(tokenErr, context.Canceled) {
			s.release(nil, false)
			return proxy.PassthroughAttempt{}, false, tokenErr
		}
		s.credentialFailed = true
		s.release(nil, false)
		return proxy.PassthroughAttempt{}, false, nil
	}
	s.current = t
	return attempt, true, nil
}

// Retryable 只在关闭固定账号且故障切换开启时，按明确拒绝状态码换号。
func (s *passthroughSession) Retryable(_ proxy.PassthroughAttempt, status int) bool {
	if !s.Switchable() {
		return false
	}
	return passthroughRetryableStatus(status)
}

// Switchable 报告是否允许换号：固定账号与关闭的故障切换都不遍历候选。
func (s *passthroughSession) Switchable() bool { return !s.fixed && s.enabled }

// Done 结束本次尝试：成功清除冷却并按需刷新额度观察；明确拒绝写入冷却或额度阻断。
func (s *passthroughSession) Done(_ context.Context, attempt proxy.PassthroughAttempt, outcome proxy.PassthroughOutcome) {
	if attempt.AccountID == "" || s.current.accountID != attempt.AccountID {
		return
	}
	current := s.current
	s.current = target{}
	if outcome.UpstreamQuotaRejected {
		// 执行前额度拒绝按真实重置时间阻断账号；共享窗口对该账号的所有模型生效。
		rejected := quotaRetryReason(outcome.QuotaResetAt, s.service.failover.now())
		s.service.rememberQuotaRejection(current, rejected)
		s.release(rejected, false)
		s.service.refreshQuota(current)
		return
	}
	// WebSocket 握手成功同样是「这次账号可用」的证据，不能因为没有 2xx 就留下旧冷却。
	healthy := outcome.StatusCode == http.StatusSwitchingProtocols || outcome.StatusCode >= 200 && outcome.StatusCode < 300
	s.release(passthroughRejection(outcome.StatusCode, outcome.RetryAfter, s.service.failover.now()), healthy)
	if healthy {
		s.service.refreshQuota(current)
	}
}

// release 归还健康租约；没有租约（固定账号）时只清空本地状态。
func (s *passthroughSession) release(rejected *retryReason, healthy bool) {
	if s.lease.id != 0 {
		s.service.failover.finish(s.lease, rejected, healthy)
	}
	s.lease = healthLease{}
}
