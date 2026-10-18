package proxy

import (
	"context"
	"time"
)

// AttemptEvent 保存一次上游尝试，不包含正文、凭据或原始错误。
type AttemptEvent struct {
	RequestID               string
	Index                   int
	StartedAt, FinishedAt   time.Time
	Target                  TargetSnapshot
	Model, State, ErrorCode string
	StatusCode              int
	LatencyMS               int64
	Usage                   UsageSnapshot
}

// AttemptObserver 在发送前登记尝试，完成事件必须幂等。
type AttemptObserver interface {
	RecordAttempt(context.Context, AttemptEvent) error
}

// BeginAttempt 不绑定父记录的账号；明确拒绝后仍可选择其他账号。
func (t *Trace) BeginAttempt(ctx context.Context, request Request, target TargetSnapshot) error {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.event.AttemptIndex >= 3 || t.event.Target.AccountID != "" {
		return Failure("accounting_unavailable", 503)
	}
	t.event.AttemptIndex++
	t.attempt = &AttemptEvent{RequestID: t.event.ID, Index: t.event.AttemptIndex, StartedAt: time.Now().UTC(), Target: target, Model: request.Model, State: "pending"}
	if observer, ok := t.observer.(AttemptObserver); ok && observer.RecordAttempt(ctx, *t.attempt) != nil {
		return Failure("accounting_unavailable", 503)
	}
	return nil
}

// EndAttempt 只记录真实观察到的结果；未知用量保持未知。
func (t *Trace) EndAttempt(status int, code string) error {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.attempt == nil {
		return Failure("accounting_unavailable", 503)
	}
	event := *t.attempt
	event.FinishedAt = time.Now().UTC()
	event.LatencyMS = max(0, event.FinishedAt.Sub(event.StartedAt).Milliseconds())
	event.StatusCode, event.ErrorCode = status, code
	if t.attempt.StatusCode != 0 {
		event.StatusCode = t.attempt.StatusCode
	}
	event.State = "success"
	if code != "" || status >= 400 {
		event.State = "failed"
	}
	if code == "request_cancelled" {
		event.State = "cancelled"
	}
	if t.event.Target.AccountID == event.Target.AccountID {
		event.Usage = t.event.Usage
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if observer, ok := t.observer.(AttemptObserver); ok && observer.RecordAttempt(ctx, event) != nil {
		return Failure("accounting_unavailable", 503)
	}
	t.attempt = nil
	return nil
}

func (t *Trace) observeStatus(status int) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.attempt != nil {
		t.attempt.StatusCode = status
	}
}
