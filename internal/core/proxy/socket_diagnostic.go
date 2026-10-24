package proxy

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"
)

type socketHandshakeKey struct{}

var socketSequence atomic.Uint64

type socketHandshake struct {
	log *slog.Logger
	id  string
}

func (s *Server) socketHandshakeContext(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := s.opts.SocketHandshakeTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	id := strconv.FormatInt(time.Now().UnixNano(), 16) + "-" + strconv.FormatUint(socketSequence.Add(1), 16)
	ctx = context.WithValue(ctx, socketHandshakeKey{}, &socketHandshake{log: s.opts.Logger, id: id})
	return context.WithTimeout(ctx, timeout)
}

// BeginSocketStage 只在透明握手中记录受控阶段与稳定错误码，绝不输出底层错误或凭据。
// 应用层借此把凭据排队耗时关联到同一次握手，而普通 HTTP 不新增握手日志。
func BeginSocketStage(ctx context.Context, stage string, attempt ...int) func(error) {
	switch stage {
	case "selection", "credential_queue", "credential_prepare", "credentials", "upstream_dial", "downstream_accept", "handshake", "relay":
	default:
		return func(error) {}
	}
	trace, _ := ctx.Value(socketHandshakeKey{}).(*socketHandshake)
	if trace == nil || trace.log == nil {
		return func(error) {}
	}
	start := time.Now()
	return func(err error) {
		code, status := "", 0
		attrs := []any{"module", "gateway", "connection_id", trace.id, "stage", stage,
			"elapsed_ms", time.Since(start).Milliseconds()}
		if len(attempt) > 0 && attempt[0] > 0 {
			attrs = append(attrs, "attempt_index", attempt[0])
		}
		var closeEvent *socketRelayError
		if errors.As(err, &closeEvent) {
			attrs = append(attrs, "direction", closeEvent.direction)
			if closeEvent.closeCode > 0 {
				attrs = append(attrs, "close_code", int(closeEvent.closeCode))
			}
			err = closeEvent.failure
		}
		if err != nil {
			failure := safeFailure(ctx, err).(*Error)
			code, status = failure.Code, failure.StatusCode
		} else if stage == "handshake" || stage == "downstream_accept" || stage == "upstream_dial" {
			status = 101
		}
		attrs = append(attrs, "code", code, "status_code", status)
		trace.log.Info("gateway websocket stage", attrs...)
	}
}

// SocketCredentialContext 将单候选准备限制在握手预算内；普通 HTTP 保持既有期限。
func SocketCredentialContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx.Value(socketHandshakeKey{}) != nil {
		// 为官方 GiB 级 KDF 冷读取留出余量，仍受全部凭据合计 5 秒的约束。
		return context.WithTimeout(ctx, 3*time.Second)
	}
	return context.WithCancel(ctx)
}
