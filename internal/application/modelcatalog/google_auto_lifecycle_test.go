package modelcatalog

import (
	"context"
	"errors"
	"testing"
	"time"

	"omniproxy/internal/provider/antigravity"
)

func TestGoogleAutoSyncCoalescesAndCancelsInFlightRequests(t *testing.T) {
	s := googleAutoFixture(t)
	if _, err := s.db.Exec(`UPDATE accounts SET enabled=0 WHERE id='z'`); err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan context.Context, 4), make(chan struct{})
	s.SetGoogleSubscription(googleAutoSource(func(ctx context.Context, id string) ([]antigravity.Model, string, error) {
		entered <- ctx
		select {
		case <-release:
			return googleSnapshot(s, ctx, id, "gemini-test")
		case <-ctx.Done():
			return nil, "", ctx.Err()
		}
	}))
	results := make(chan error, 4)
	auto := StartGoogleAutoSync(context.Background(), s, nil, func(err error) { results <- err })
	t.Cleanup(auto.Close)
	ctx := awaitGoogleSync(t, entered)
	deadline, limited := ctx.Deadline()
	if !limited || time.Until(deadline) > googleSyncTimeout {
		t.Fatal("每账号请求没有独立超时")
	}
	for range 100 {
		auto.Request()
	}
	release <- struct{}{}
	if err := awaitGoogleSync(t, results); err != nil {
		t.Fatal(err)
	}
	awaitGoogleSync(t, entered)
	closed := make(chan struct{})
	go func() { auto.Close(); close(closed) }()
	awaitGoogleSync(t, closed)
	auto.Request()
	if len(entered) != 0 || len(results) != 0 {
		t.Fatal("关闭后继续同步或发布取消结果")
	}
}

func TestGoogleAutoSyncReportsClientRefreshFailureWithoutLosingModels(t *testing.T) {
	s := googleAutoFixture(t)
	s.SetGoogleSubscription(googleAutoSource(func(ctx context.Context, id string) ([]antigravity.Model, string, error) {
		return googleSnapshot(s, ctx, id, "gemini-"+id)
	}))
	conflict := errors.New("agent: config_changed")
	results := make(chan error, 2)
	auto := StartGoogleAutoSync(context.Background(), s, func(context.Context) error {
		return conflict
	}, func(err error) { results <- err })
	t.Cleanup(auto.Close)
	if err := awaitGoogleSync(t, results); !errors.Is(err, conflict) {
		t.Fatal("客户端外部修改错误被吞掉", err)
	}
	items, err := s.List(context.Background())
	if err != nil || len(items) != 2 {
		t.Fatal("客户端更新失败影响网关目录", err)
	}
}
