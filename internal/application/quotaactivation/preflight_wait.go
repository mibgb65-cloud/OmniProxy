package quotaactivation

import (
	"context"
	"errors"
	"time"
)

// 同秒读取无法区分“目标不动”和秒级取整；有界等待后复查，不能永远释放登记却始终不发。
func waitForPreflight(ctx context.Context, plan SendPlan) error {
	if len(plan.Floating) == 0 {
		return nil
	}
	at, err := time.Parse(time.RFC3339, plan.Before.FetchedAt)
	if err != nil {
		return errors.Join(ErrNotSent, ErrAwaitingQuota)
	}
	wait := time.Until(at.Add(2 * time.Second))
	if wait > 3*time.Second {
		return errors.Join(ErrNotSent, ErrAwaitingQuota)
	}
	if wait <= 0 {
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return errors.Join(ErrNotSent, ctx.Err())
	case <-timer.C:
		return nil
	}
}
