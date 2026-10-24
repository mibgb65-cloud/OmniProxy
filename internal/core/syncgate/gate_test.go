package syncgate

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAccountIsolationCancellationAndGlobalExclusion(t *testing.T) {
	var gate Gate
	ctx := context.Background()
	if err := gate.LockAccount(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if err := gate.LockAccount(ctx, "b"); err != nil {
		t.Fatal("独立账号被串行阻塞", err)
	}
	for _, global := range []bool{false, true} {
		wait, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
		var err error
		if global {
			err = gate.Lock(wait)
		} else {
			err = gate.LockAccount(wait, "a")
		}
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("同账号或全局操作未等待活动账号", err)
		}
	}
	gate.UnlockAccount("a")
	gate.UnlockAccount("b")
	if err := gate.Lock(ctx); err != nil {
		t.Fatal(err)
	}
	wait, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()
	if err := gate.LockAccount(wait, "a"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("全局提交期间仍能取得账号准备门", err)
	}
	gate.Unlock()
	gate.mu.Lock()
	defer gate.mu.Unlock()
	if gate.global || gate.waitingGlobal != 0 || len(gate.accounts) != 0 {
		t.Fatal("取消或释放后残留锁状态")
	}
}
