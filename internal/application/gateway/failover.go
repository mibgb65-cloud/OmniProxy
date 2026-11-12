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
	for index, candidate := range request.Candidates {
		if seen[candidate.AccountID] || attempts >= maxFailoverAttempts {
			continue
		}
		seen[candidate.AccountID] = true
		// 后面还有可尝试的候选：订阅通道据此决定「已知拒绝的 prompt」是否先跳过该账号。
		next := hasLaterCandidate(request.Candidates[index+1:], seen) && attempts+1 < maxFailoverAttempts
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
		current.LastCandidate = !next
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
		if skipped, ok := refusedBefore(err); ok && !request.FixedAccount {
			// 该账号 6 小时内拒绝过同一 system prompt：不再发送、不写冷却，直接换下一个候选。
			s.failover.finish(lease, nil, false)
			released = true
			if endErr := request.Trace.EndAttempt(skipped.StatusCode, skipped.Code); endErr != nil {
				return endErr
			}
			continue
		}
		if rejected == nil {
			if quota, ok := quotaExhausted(err); ok && !request.FixedAccount {
				if !next {
					// 没有别的候选：把真实错误交还客户端，不伪装成账号池冷却。
					s.failover.finish(lease, nil, false)
					released = true
					if endErr := request.Trace.EndAttempt(quota.StatusCode, quota.Code); endErr != nil {
						return endErr
					}
					return err
				}
				// 还有候选：按上游给出的重置时间休息本账号，由下一个候选接替。
				rejected = &retryReason{code: quota.Code, delay: quotaRetryDelay(quota, s.failover.now())}
			}
		}
		cooldown := rejected
		if cooldown != nil && cooldown.code == "duplicate_credential" {
			cooldown = nil
		}
		// 已有版本化共享额度观察时，以其窗口重置／新快照恢复为准，避免另存模型冷却阻止重置卡生效。
		// 这条只针对 Codex 订阅；Google 订阅的额度按账号独立，需要休息本账号。
		if cooldown != nil && cooldown.code == "upstream_quota_exhausted" && t.kind == "openai" && codexLoginMethod(t.authMethod) {
			if _, ok := s.subscription.(subscriptionQuotaAvailability); ok {
				cooldown = nil
			}
		}
		if t.kind == "river" {
			s.failover.finishResult(lease, cooldown, accepted && err == nil, rejected == nil && riverTransientFailure(ctx, responseStatus, err))
		} else {
			s.failover.finish(lease, cooldown, accepted)
		}
		released = true
		if accepted && err == nil {
			// 成功请求之后按需刷新该账号额度观察，让下一条消息不必先撞一次已知耗尽账号。
			s.refreshQuota(t)
		}
		status, code := attemptFailure(ctx, err)
		if t.kind == "river" && rejected == nil && responseStatus >= 400 {
			status, code = responseStatus, "upstream_rejected_request"
			if responseStatus >= 500 {
				code = "upstream_unavailable"
			}
		}
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
