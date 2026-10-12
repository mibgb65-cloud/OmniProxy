package logger

import (
	"encoding/json"
	"log/slog"
)

func (h redactingHandler) redactValue(value slog.Value, depth int) slog.Value {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		return slog.StringValue(h.redactor.Redact(value.String()))
	case slog.KindGroup:
		attrs := value.Group()
		out := make([]slog.Attr, len(attrs))
		for i, attr := range attrs {
			out[i] = h.redactAttrDepth(attr, depth+1)
		}
		return slog.GroupValue(out...)
	case slog.KindAny:
		if err, ok := value.Any().(error); ok {
			return slog.StringValue(h.redactor.Redact(err.Error()))
		}
		// 复杂值先转为 JSON 树再逐字段脱敏；不能安全序列化的对象统一遮掩。
		data, err := json.Marshal(value.Any())
		if err != nil || len(data) > 64<<10 {
			return slog.StringValue(masked)
		}
		var tree any
		if json.Unmarshal(data, &tree) != nil {
			return slog.StringValue(masked)
		}
		return slog.AnyValue(h.redactor.redactTree(tree, 0))
	default:
		return value
	}
}

func (r *Redactor) redactTree(value any, depth int) any {
	if depth > 16 {
		return masked
	}
	switch item := value.(type) {
	case string:
		return r.Redact(item)
	case map[string]any:
		for key, field := range item {
			if r.SensitiveKey(key) {
				item[key] = masked
			} else {
				item[key] = r.redactTree(field, depth+1)
			}
		}
	case []any:
		for i, field := range item {
			item[i] = r.redactTree(field, depth+1)
		}
	}
	return value
}
