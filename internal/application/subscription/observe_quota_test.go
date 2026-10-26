package subscription

import (
	"context"
	"testing"
	"time"

	"omniproxy/internal/infrastructure/codex"
)

// awaitQuotaRead 等待一次额度读取发生，超时即失败。
func awaitQuotaRead(t *testing.T, reads <-chan struct{}) {
	t.Helper()
	select {
	case <-reads:
	case <-time.After(3 * time.Second):
		t.Fatal("quota observation was never refreshed")
	}
}

func assertNoQuotaRead(t *testing.T, reads <-chan struct{}) {
	t.Helper()
	select {
	case <-reads:
		t.Fatal("quota observation refreshed more often than allowed")
	case <-time.After(100 * time.Millisecond):
	}
}

type completedQuotaProbe struct {
	quotaProbeClient
	done chan struct{}
}

func (c completedQuotaProbe) Close() {
	c.quotaProbeClient.Close()
	// readQuota 在 normalize／publishQuota 返回后才执行 Close；
	// 不能在 RateLimits 返回前通知，否则测试会抢在网关观察写入前继续。
	close(c.done)
}

// TestObserveQuotaRefreshesOncePerAccountVersion 断言按需刷新只对同一账号＋凭据版本
// 去抖一次，凭据版本变化后立即重新观察，退出后不再发起查询。
func TestObserveQuotaRefreshesOncePerAccountVersion(t *testing.T) {
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.authorize("quota@example.test")
	item := awaitState(t, f.manager, view.ID, "completed").Account

	reads := make(chan struct{}, 8)
	f.manager.factory = func(context.Context, string) (codex.Client, error) {
		return quotaProbeClient{Client: client, account: client.Account, limits: func(ctx context.Context) (codex.RateLimits, error) {
			reads <- struct{}{}
			return client.RateLimits(ctx)
		}}, nil
	}

	f.manager.ObserveQuota(item.ID, item.UpdatedAt)
	awaitQuotaRead(t, reads)

	f.manager.ObserveQuota(item.ID, item.UpdatedAt)
	assertNoQuotaRead(t, reads)

	// 凭据版本变化代表账号已被替换或重新授权，必须立即重新观察。
	f.manager.ObserveQuota(item.ID, item.UpdatedAt+"-rotated")
	awaitQuotaRead(t, reads)

	// 缺账号或版本时不猜测目标，也不启动官方查询。
	f.manager.ObserveQuota("", item.UpdatedAt)
	f.manager.ObserveQuota(item.ID, "")
	assertNoQuotaRead(t, reads)

	f.manager.stopQuotaRefresh()
	f.manager.ObserveQuota(item.ID, item.UpdatedAt+"-closed")
	assertNoQuotaRead(t, reads)
}

// TestObserveQuotaFeedsGatewayAvailability 断言刷新写入的是网关读取的同一份额度观察。
func TestObserveQuotaFeedsGatewayAvailability(t *testing.T) {
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.authorize("quota@example.test")
	item := awaitState(t, f.manager, view.ID, "completed").Account
	client.mu.Lock()
	client.rateLimits = &codex.RateLimits{AccountID: item.AuthAccountID, Snapshot: quotaSnapshot("plus", 10, 100)}
	client.mu.Unlock()

	done := make(chan struct{})
	f.manager.factory = func(context.Context, string) (codex.Client, error) {
		return completedQuotaProbe{
			quotaProbeClient: quotaProbeClient{Client: client, account: client.Account, limits: client.RateLimits},
			done:             done,
		}, nil
	}
	f.manager.ObserveQuota(item.ID, item.UpdatedAt)
	awaitQuotaRead(t, done)

	until, blocked := f.manager.QuotaBlockedUntil(item.ID, item.UpdatedAt, time.Now())
	if !blocked || until.IsZero() {
		t.Fatal("on-demand quota refresh did not feed the gateway gate")
	}
}
