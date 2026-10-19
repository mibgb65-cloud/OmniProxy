package logger

import (
	"context"
	"errors"
	"log/slog"
)

// fanoutHandler fans every record out to all inner handlers.
type fanoutHandler struct {
	handlers []slog.Handler
}

func (h fanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, sub := range h.handlers {
		if sub.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h fanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	var failures []error
	// Windows GUI 进程可能没有有效 stderr；单个出口失败不能阻断文件和界面诊断。
	for _, sub := range h.handlers {
		if err := sub.Handle(ctx, r); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (h fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	subs := make([]slog.Handler, len(h.handlers))
	for i, sub := range h.handlers {
		subs[i] = sub.WithAttrs(attrs)
	}
	return fanoutHandler{handlers: subs}
}

func (h fanoutHandler) WithGroup(name string) slog.Handler {
	subs := make([]slog.Handler, len(h.handlers))
	for i, sub := range h.handlers {
		subs[i] = sub.WithGroup(name)
	}
	return fanoutHandler{handlers: subs}
}

// redactingHandler redacts the record message and string attribute values
// before delegating to the inner handler. Every sink in OmniProxy is wrapped
// by this handler, so no module can accidentally leak a credential.
type redactingHandler struct {
	inner          slog.Handler
	redactor       *Redactor
	sensitiveGroup bool
}

func newRedactingHandler(inner slog.Handler, redactor *Redactor) slog.Handler {
	return redactingHandler{inner: inner, redactor: redactor}
}

func (h redactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h redactingHandler) Handle(ctx context.Context, r slog.Record) error {
	redacted := slog.NewRecord(r.Time, r.Level, h.redactor.Redact(r.Message), r.PC)
	r.Attrs(func(a slog.Attr) bool {
		redacted.AddAttrs(h.redactAttr(a))
		return true
	})
	return h.inner.Handle(ctx, redacted)
}

func (h redactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	copied := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		copied[i] = h.redactAttr(a)
	}
	return redactingHandler{inner: h.inner.WithAttrs(copied), redactor: h.redactor, sensitiveGroup: h.sensitiveGroup}
}

func (h redactingHandler) WithGroup(name string) slog.Handler {
	return redactingHandler{inner: h.inner.WithGroup(h.redactor.Redact(name)), redactor: h.redactor, sensitiveGroup: h.sensitiveGroup || h.redactor.SensitiveKey(name)}
}

func (h redactingHandler) redactAttr(a slog.Attr) slog.Attr {
	return h.redactAttrDepth(a, 0)
}

func (h redactingHandler) redactAttrDepth(a slog.Attr, depth int) slog.Attr {
	if depth > 16 || h.sensitiveGroup || h.redactor.SensitiveKey(a.Key) {
		return slog.String(a.Key, masked)
	}
	a.Value = h.redactValue(a.Value, depth)
	return a
}
