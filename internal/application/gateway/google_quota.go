package gateway

import (
	"context"
	"errors"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/core/proxy"
)

// googleRateLimitCooldown 是没有已知重置时间时的休息时长：比通用 30 秒长，
// 避免 5h 窗口耗尽时每隔半分钟再撞同一个账号。
const googleRateLimitCooldown = 60 * time.Second

// GoogleQuotaGate 只读已发布的额度快照，判断某 Google 订阅账号的某模型是否已经耗尽。
// 它不发起官方查询，也不改变账号状态。
type GoogleQuotaGate interface {
	BlockedUntil(ctx context.Context, accountID, updatedAt, model string, now time.Time) (time.Time, bool)
}

// googleQuotaRefresher 让刚被 429 的账号刷新一次快照：
// 同组的其它模型随后也能被跳过，不必各撞一次。
type googleQuotaRefresher interface {
	ObserveQuota(accountID string)
}

// SetGoogleQuotaGate 安装 Google 订阅的额度预筛；未安装时按原有行为选号。
func (s *Service) SetGoogleQuotaGate(gate GoogleQuotaGate) { s.googleQuota = gate }

// googleQuotaWait 返回该账号该模型还要等多久；非 Google 订阅账号或无法判断时为 0。
func (s *Service) googleQuotaWait(ctx context.Context, t target, model string, now time.Time) time.Duration {
	if s.googleQuota == nil || t.kind != "google" || t.authMethod != account.AuthSubscription ||
		t.accountID == "" || model == "" {
		return 0
	}
	until, blocked := s.googleQuota.BlockedUntil(ctx, t.accountID, t.updatedAt, model, now)
	if !blocked {
		return 0
	}
	delay := until.Sub(now)
	if delay <= 0 || delay > maxQuotaBlockWindow {
		return 0
	}
	return max(time.Second, delay)
}

// googleRateLimited 把 Google 订阅的 429 视为账号级拒绝：订阅额度按账号计费，
// 换同池其它账号有机会成功。普通 API Key 仍共享项目额度，交由服务商级逻辑处理。
// 休息时长优先取上游重置秒数，其次取快照里该模型的已知重置时间，未知时退到 60 秒。
func (s *Service) googleRateLimited(ctx context.Context, t target, model string, err error) (*retryReason, *proxy.Error) {
	if t.kind != "google" || t.authMethod != account.AuthSubscription {
		return nil, nil
	}
	var failure *proxy.Error
	if !errors.As(err, &failure) || failure.StatusCode != 429 {
		return nil, nil
	}
	if failure.Code != "rate_limited" && failure.Code != "upstream_rate_limited" {
		return nil, nil
	}
	now := s.failover.now()
	delay := googleRateLimitCooldown
	if parsed, ok := retryDelay(failure.RetryAfter, now); ok && parsed <= maxQuotaBlockWindow {
		delay = parsed
	} else if blocked := s.googleQuotaWait(ctx, t, model, now); blocked > 0 {
		delay = blocked
	}
	if delay > maxQuotaBlockWindow {
		delay = maxQuotaBlockWindow
	}
	return &retryReason{code: failure.Code, delay: delay}, failure
}

// refreshGoogleQuota 让刚被 429 的账号刷新一次额度快照；没有实现时什么也不做。
func (s *Service) refreshGoogleQuota(t target) {
	if s == nil || t.kind != "google" || t.authMethod != account.AuthSubscription || t.accountID == "" {
		return
	}
	if source, ok := s.googleAuth.(googleQuotaRefresher); ok {
		source.ObserveQuota(t.accountID)
	}
}
