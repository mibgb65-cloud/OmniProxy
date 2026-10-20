package codex

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type silentRPCWriter struct {
	calls int
	sent  chan struct{}
}

func (w *silentRPCWriter) Write(data []byte) (int, error) {
	w.calls++
	if w.sent != nil {
		close(w.sent)
	}
	return len(data), nil
}
func (*silentRPCWriter) Close() error { return nil }

func TestRPCCancelAndDeadlineRemainDistinct(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancelled", true: "timeout"}[deadline], func(t *testing.T) {
			writer := &silentRPCWriter{sent: make(chan struct{})}
			client := &rpcClient{input: writer, pending: map[uint64]chan wireResponse{}, done: make(chan struct{})}
			ctx, cancel := context.WithCancel(context.Background())
			cause, code := context.Canceled, "operation_cancelled"
			if deadline {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
				cause, code = context.DeadlineExceeded, "operation_timeout"
			} else {
				go func() { <-writer.sent; cancel() }()
			}
			defer cancel()
			var result json.RawMessage
			err := client.call(ctx, "account/read", nil, &result)
			if !errors.Is(err, cause) || err.Error() != "subscription: "+code || len(client.pending) != 0 {
				t.Fatalf("RPC context lost its classification: %v", err)
			}
		})
	}
}

func TestExpiredRPCDoesNotWriteRequest(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	writer := &silentRPCWriter{}
	client := &rpcClient{input: writer, pending: map[uint64]chan wireResponse{}, done: make(chan struct{})}
	var result json.RawMessage
	err := client.call(ctx, "account/read", nil, &result)
	if !errors.Is(err, context.DeadlineExceeded) || writer.calls != 0 || len(client.pending) != 0 {
		t.Fatal("expired RPC still reached the official component", err)
	}
}
