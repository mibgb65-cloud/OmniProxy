package proxy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// TargetSnapshot 仅含公开账号身份；不包含凭据、引用、请求内容或原始头部。
type TargetSnapshot struct{ ProviderID, ProviderName, ProviderKind, AccountID, AccountName, AuthMethod, Email, WorkspaceID, ProviderRegion, BillingMode string }
type UsageSnapshot struct {
	Input, Output                     int64
	InputKnown, OutputKnown, Complete bool
	CachedInput                       *int64
}
type RequestEvent struct {
	AttemptIndex                           int
	ID                                     string
	StartedAt, FinishedAt                  time.Time
	InputModel, Model, Protocol, Transport string
	Stream                                 bool
	Target                                 TargetSnapshot
	Usage                                  UsageSnapshot
	State, ErrorCode                       string
	StatusCode                             int
	LatencyMS                              int64
	FirstTokenMS                           *int64
}

// RequestObserver 在转发前持久化，消费上游用量快照时原子累计，结束更新结果。
type RequestObserver interface {
	Begin(context.Context, RequestEvent) error
	Update(context.Context, RequestEvent) error
	Finish(context.Context, RequestEvent) error
}
type Trace struct {
	upstreamMediaType          string
	contentTypeInferred        bool
	contentTypeInferenceReason string
	contentTypeInferenceBytes  int
	attempt                    *AttemptEvent
	mu                         sync.Mutex
	event                      RequestEvent
	observer                   RequestObserver
	startedAt                  time.Time
}

func (s *Server) beginTrace(ctx context.Context, protocol, transport string) (*Trace, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return nil, Failure("accounting_unavailable", 503)
	}
	now := time.Now()
	trace := &Trace{event: RequestEvent{ID: hex.EncodeToString(value), StartedAt: now.UTC(), Protocol: protocol, Transport: transport, State: "pending"}, startedAt: now, observer: s.opts.Observer}
	if trace.observer != nil {
		if err := trace.observer.Begin(ctx, trace.event); err != nil {
			return nil, Failure("accounting_unavailable", 503)
		}
	}
	return trace, nil
}
func (t *Trace) ID() string {
	if t == nil {
		return ""
	}
	return t.event.ID
}
func (t *Trace) Update(ctx context.Context, request Request, target TargetSnapshot) error {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.event.InputModel == "" {
		t.event.InputModel = request.Model
	}
	t.event.Model = request.Model
	t.event.Stream = request.Stream
	if target.AccountID != "" {
		t.event.Target = target
	}
	if t.observer != nil && t.observer.Update(ctx, t.event) != nil {
		return Failure("accounting_unavailable", 503)
	}
	return nil
}
func (t *Trace) Report(_ context.Context, usage UsageSnapshot) error {
	if t == nil {
		return nil
	}
	// 用量已经被上游报告；客户端断开不能取消这一笔持久化。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	t.mu.Lock()
	defer t.mu.Unlock()
	// 协议转换可能先读取上游用量再调用下游消费；此时才将唯一成功尝试绑定到父记录。
	if t.event.Target.AccountID == "" && t.attempt != nil {
		t.event.Target = t.attempt.Target
		t.event.Model = t.attempt.Model
	}
	if usage.InputKnown && (!t.event.Usage.InputKnown || usage.Input >= t.event.Usage.Input) {
		if usage.CachedInput != nil || usage.Input > t.event.Usage.Input {
			t.event.Usage.CachedInput = usage.CachedInput
		}
		t.event.Usage.Input = usage.Input
		t.event.Usage.InputKnown = true
	}
	if usage.OutputKnown && (!t.event.Usage.OutputKnown || usage.Output >= t.event.Usage.Output) {
		t.event.Usage.Output = usage.Output
		t.event.Usage.OutputKnown = true
	}
	t.event.Usage.Complete = (t.event.Usage.Complete || usage.Complete) && t.event.Usage.InputKnown && t.event.Usage.OutputKnown
	if t.observer != nil && t.observer.Update(ctx, t.event) != nil {
		return Failure("accounting_unavailable", 503)
	}
	return nil
}
func (t *Trace) finish(status int, code string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.event.FinishedAt = time.Now().UTC()
	t.event.LatencyMS = t.elapsedMS()
	t.event.StatusCode = status
	t.event.ErrorCode = code
	t.event.State = "success"
	if code != "" || status >= 400 {
		t.event.State = "failed"
	}
	if code == "request_cancelled" {
		t.event.State = "cancelled"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if t.observer != nil && t.observer.Finish(ctx, t.event) != nil {
		return Failure("accounting_unavailable", 503)
	}
	return nil
}
