package logger

import (
	"context"
	"log/slog"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var modulePattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,47}$`)
var uiFieldKeys = map[string]bool{
	"provider": true, "provider_id": true, "account_id": true, "request_id": true,
	"client": true, "model": true, "stream": true, "code": true, "status_code": true,
	"latency_ms": true, "duration_ms": true, "proxy_port": true, "control_port": true,
	"profile": true, "app_id": true, "freed_bytes": true, "error": true,
	"operation": true, "enabled": true, "stage": true, "attempt": true, "state": true,
	"input_tokens": true, "output_tokens": true, "usage_state": true,
	"first_token_ms":      true,
	"upstream_media_type": true, "upstream_content_type_inferred": true,
	"upstream_inference_reason": true, "upstream_prefix_bytes": true,
}

type scopedAttr struct {
	groups string
	attr   slog.Attr
}
type uiHandler struct {
	buffer *UIBuffer
	level  slog.Level
	groups string
	attrs  []scopedAttr
}

func (h uiHandler) Enabled(_ context.Context, level slog.Level) bool { return level >= h.level }
func (h uiHandler) Handle(_ context.Context, record slog.Record) error {
	level := "INFO"
	if record.Level >= slog.LevelError {
		level = "ERROR"
	} else if record.Level >= slog.LevelWarn {
		level = "WARN"
	} else if record.Level < slog.LevelInfo {
		level = "DEBUG"
	}
	entry := Entry{Time: record.Time.UTC().Format(time.RFC3339Nano), Level: level, Module: moduleFor(record.PC), Message: boundedText(record.Message), Fields: map[string]string{}}
	for _, item := range h.attrs {
		captureAttr(&entry, item.groups, item.attr)
	}
	record.Attrs(func(attr slog.Attr) bool { captureAttr(&entry, h.groups, attr); return true })
	h.buffer.addEntry(entry)
	return nil
}
func (h uiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.attrs = append([]scopedAttr(nil), h.attrs...)
	for _, attr := range attrs {
		h.attrs = append(h.attrs, scopedAttr{h.groups, attr})
	}
	return h
}
func (h uiHandler) WithGroup(name string) slog.Handler {
	if name != "" {
		h.groups += name + "."
	}
	return h
}

func captureAttr(entry *Entry, prefix string, attr slog.Attr) {
	value := attr.Value.Resolve()
	if value.Kind() == slog.KindGroup {
		if attr.Key != "" {
			prefix += attr.Key + "."
		}
		if len(prefix) > 64 {
			return
		}
		for _, child := range value.Group() {
			captureAttr(entry, prefix, child)
		}
		return
	}
	if attr.Key == "module" && value.Kind() == slog.KindString && modulePattern.MatchString(value.String()) {
		entry.Module = value.String()
		return
	}
	// 界面只接收诊断白名单，不把未知对象或请求／回复字段送到前端。
	if !safeUIPrefix(prefix) || !uiFieldKeys[attr.Key] || len(entry.Fields) >= 24 || len(prefix) > 64 {
		return
	}
	var bytes int
	for _, field := range entry.Fields {
		bytes += len(field)
	}
	if bytes >= 8192 {
		return
	}
	entry.Fields[prefix+attr.Key] = boundedText(value.String())
}

func safeUIPrefix(prefix string) bool {
	for _, part := range strings.Split(strings.TrimSuffix(prefix, "."), ".") {
		switch part {
		case "", "http", "request", "gateway", "routing", "runtime", "storage":
		default:
			return false
		}
	}
	return true
}

func moduleFor(pc uintptr) string {
	if function := runtime.FuncForPC(pc); function != nil {
		name := strings.ToLower(function.Name())
		for key, module := range map[string]string{"/core/proxy": "gateway", "/routing": "routing", "/application/account": "accounts", "/application/provider": "providers", "storage": "storage", "/httpapi": "control"} {
			if strings.Contains(name, key) {
				return module
			}
		}
	}
	return "app"
}
