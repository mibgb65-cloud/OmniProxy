package gateway

import (
	"context"
	"errors"
	"time"

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
	if s.fixed || !s.enabled {
		return false
	}
	return passthroughRetryableStatus(status)
}

// Done 结束本次尝试：成功清除冷却，明确拒绝写入冷却窗口。
func (s *passthroughSession) Done(_ context.Context, attempt proxy.PassthroughAttempt, status int) {
	if attempt.AccountID == "" || s.current.accountID != attempt.AccountID {
		return
	}
	s.release(passthroughRejection(status), status >= 200 && status < 300)
	s.current = target{}
}

// release 归还健康租约；没有租约（固定账号）时只清空本地状态。
func (s *passthroughSession) release(rejected *retryReason, healthy bool) {
	if s.lease.id != 0 {
		s.service.failover.finish(s.lease, rejected, healthy)
	}
	s.lease = healthLease{}
}

// passthroughRetryableStatus 与 v1 的可重试集合一致，另外把 401 视为账号级失效。
func passthroughRetryableStatus(status int) bool {
	switch status {
	case 401, 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

// passthroughRejection 只依据明确拒绝分类写入冷却，不猜泛化 5xx 之外的语义。
func passthroughRejection(status int) *retryReason {
	switch status {
	case 401:
		return &retryReason{code: "upstream_authentication_failed", delay: 5 * time.Minute}
	case 429:
		return &retryReason{code: "upstream_rate_limited", delay: 30 * time.Second}
	case 500, 502, 503, 504:
		return &retryReason{code: "upstream_unavailable", delay: 30 * time.Second}
	default:
		return nil
	}
}
