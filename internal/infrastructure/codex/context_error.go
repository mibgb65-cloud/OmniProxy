package codex

import (
	"context"
	"errors"
)

// rpcContextError 保留可检查的取消原因，稳定错误码中不携带官方请求内容。
type rpcContextError struct{ cause error }

func (e *rpcContextError) Error() string {
	if errors.Is(e.cause, context.DeadlineExceeded) {
		return "subscription: operation_timeout"
	}
	return "subscription: operation_cancelled"
}

func (e *rpcContextError) Unwrap() error { return e.cause }
