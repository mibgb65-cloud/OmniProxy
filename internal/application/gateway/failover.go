package gateway

import (
	"context"
	"errors"
	"omniproxy/internal/core/proxy"
	"strconv"
	"time"
)

var errSwitchAccount = errors.New("gateway: switch_account")

func (s *Service) forwardWithFailover(ctx context.Context, request proxy.Request, consume func(proxy.Response) error) error {
	ctx = context.WithValue(ctx, credentialClaimKey{}, map[[32]byte]bool{})
	if consume == nil {
		return proxy.Failure("invalid_configuration", 503)
	}
	enabled := true
	var err error
	if s.failoverEnabled != nil {
		enabled, err = s.failoverEnabled(ctx)
		if err != nil {
			return err
		}
	}
	initial, err := s.initialTarget(ctx, request)
	if err != nil {
		return err
	}
	if request.ProviderID == "" {
		request.ProviderID = initial.snapshot.ProviderID
	}
	if len(request.Candidates) == 0 {
		request.Candidates = []proxy.AccountCandidate{{AccountID: initial.accountID, UpdatedAt: initial.updatedAt}}
	}
	request.Candidates, err = s.rotateAccounts(ctx, initial, request)
	if err != nil {
		return err
	}
	if !enabled {
		request.Candidates = request.Candidates[:1]
	}
	seen := map[string]bool{}
	var last *proxy.Response
	lastTarget := initial
	attempts := 0
	var wait time.Duration
	for _, candidate := range request.Candidates {
		if seen[candidate.AccountID] || attempts >= maxFailoverAttempts {
			continue
		}
		seen[candidate.AccountID] = true
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if enabled && !request.FixedAccount {
			if delay := s.failover.wait(candidate.AccountID, candidate.UpdatedAt, request.Model); delay > 0 {
				if wait == 0 || delay < wait {
					wait = delay
				}
				continue
			}
		}
		current := request
		current.AccountID, current.AccountUpdatedAt = candidate.AccountID, candidate.UpdatedAt
		t, err := s.selectRoutedTarget(ctx, current)
		if err != nil {
			return err
		}
		if !sameTargetPool(initial, t) {
			return proxy.Failure("route_configuration_changed", 503)
		}
		if enabled && !request.FixedAccount {
			if delay := s.subscriptionQuotaWait(t, s.failover.now()); delay > 0 {
				if wait == 0 || delay < wait {
					wait = delay
				}
				continue
			}
		}
		lease := healthLease{}
		if enabled && !request.FixedAccount {
			var available bool
			var delay time.Duration
			lease, available, delay = s.failover.acquire(t, request.Model)
			if !available {
				if wait == 0 || delay < wait {
					wait = delay
				}
				continue
			}
		}
		released := false
		defer func() {
			if !released {
				s.failover.finish(lease, nil, false)
			}
		}()
		if err := request.Trace.BeginAttempt(ctx, current, t.snapshot); err != nil {
			return err
		}
		attempts++
		var rejected *retryReason
		accepted, bound := false, false
		var responseStatus int
		err = s.forwardAgent(ctx, t, current, func(response proxy.Response) error {
			responseStatus = response.StatusCode
			if enabled {
				rejected = rejectedAccount(t, response, s.failover.now())
			}
			if rejected != nil {
				lastTarget = t
				last = &proxy.Response{StatusCode: response.StatusCode, RetryAfter: response.RetryAfter, PreExecutionQuotaRejected: response.PreExecutionQuotaRejected, QuotaResetAt: response.QuotaResetAt}
				s.rememberQuotaRejection(t, rejected)
				if delay, ok := retryDelay(response.RetryAfter, s.failover.now()); ok {
					last.RetryAfter = strconv.Itoa(int((delay + time.Second - 1) / time.Second))
				}
				return errSwitchAccount
			}
			accepted = response.StatusCode == 200 && !response.PreExecutionQuotaRejected
			if err := request.Trace.Update(ctx, current, t.snapshot); err != nil {
				return err
			}
			bound = true
			return consume(response)
		})
		var duplicate *proxy.Error
		if enabled && errors.As(err, &duplicate) && duplicate.Code == "duplicate_credential" {
			rejected = &retryReason{code: duplicate.Code}
			responseStatus = 503
		}
		cooldown := rejected
		if cooldown != nil && cooldown.code == "duplicate_credential" {
			cooldown = nil
		}
		// 已有版本化共享额度观察时，以其窗口重置／新快照恢复为准，避免另存模型冷却阻止重置卡生效。
		if cooldown != nil && cooldown.code == "upstream_quota_exhausted" {
			if _, ok := s.subscription.(subscriptionQuotaAvailability); ok {
				cooldown = nil
			}
		}
		s.failover.finish(lease, cooldown, accepted)
		released = true
		status, code := attemptFailure(ctx, err)
		if rejected != nil {
			status, code = responseStatus, rejected.code
		}
		if ctx.Err() != nil {
			rejected = nil
			status, code = attemptFailure(ctx, ctx.Err())
		}
		if rejected == nil && !bound {
			if bindErr := request.Trace.Update(ctx, current, t.snapshot); bindErr != nil && ctx.Err() == nil {
				err = bindErr
				status, code = attemptFailure(ctx, err)
			}
		}
		if endErr := request.Trace.EndAttempt(status, code); endErr != nil {
			return endErr
		}
		if rejected != nil && !request.FixedAccount {
			continue
		}
		if rejected != nil {
			break
		}
		if err == nil || ctx.Err() != nil {
			return err
		}
		var failure *proxy.Error
		if errors.As(err, &failure) {
			return failure
		}
		return proxy.Failure(code, status)
	}
	if last != nil {
		if err := request.Trace.Update(ctx, request, lastTarget.snapshot); err != nil {
			return err
		}
		return consume(*last)
	}
	return &proxy.Error{Code: "account_pool_cooling", StatusCode: 429, RetryAfter: strconv.Itoa(max(1, int(wait.Seconds())))}
}

func sameTargetPool(a, b target) bool {
	return a.snapshot.ProviderID == b.snapshot.ProviderID && a.baseURL == b.baseURL && a.snapshot.AuthMethod == b.snapshot.AuthMethod && a.snapshot.BillingMode == b.snapshot.BillingMode && a.snapshot.ProviderRegion == b.snapshot.ProviderRegion
}

func (s *Service) initialTarget(ctx context.Context, r proxy.Request) (target, error) {
	if r.ProviderID != "" {
		return s.selectRoutedTarget(ctx, r)
	}
	t, err := s.selectTarget(ctx)
	if err != nil {
		return t, err
	}
	r.ProviderID, r.AccountID = t.snapshot.ProviderID, t.accountID
	return s.selectRoutedTarget(ctx, r)
}
