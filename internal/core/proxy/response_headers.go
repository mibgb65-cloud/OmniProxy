package proxy

import (
	"mime"
	"strings"
)

// 头部属于不可信外部输入；日志只接收固定分类，不保存原值或任何响应正文。
func responseMediaType(value string) string {
	if strings.TrimSpace(value) == "" {
		return "missing"
	}
	kind, _, err := mime.ParseMediaType(value)
	if err != nil {
		return "invalid"
	}
	switch kind {
	case "text/event-stream":
		return "sse"
	case "application/json":
		return "json"
	case "text/html":
		return "html"
	}
	return "other"
}

func (t *Trace) observeResponseHeaders(response Response) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.upstreamMediaType = responseMediaType(response.ContentType)
	t.contentTypeInferred = response.ContentTypeInferred
	t.contentTypeInferenceReason = safeInferenceReason(response.ContentTypeInferenceReason)
	t.contentTypeInferenceBytes = 0
	if response.ContentTypeInferenceBytes >= 0 && response.ContentTypeInferenceBytes <= MaxResponsesRequestBytes {
		t.contentTypeInferenceBytes = response.ContentTypeInferenceBytes
	}
	if response.ContentTypeInferred {
		t.upstreamMediaType = "missing"
	}
}

func (t *Trace) responseHeaderLogFields() []any {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.upstreamMediaType == "" {
		return nil
	}
	fields := []any{"upstream_media_type", t.upstreamMediaType, "upstream_content_type_inferred", t.contentTypeInferred}
	if t.contentTypeInferenceReason != "" {
		fields = append(fields, "upstream_inference_reason", t.contentTypeInferenceReason, "upstream_prefix_bytes", t.contentTypeInferenceBytes)
	}
	return fields
}

// 提供商诊断也经过白名单；以后新增适配器不能把动态上游错误误写入日志。
func safeInferenceReason(reason string) string {
	switch reason {
	case "verified", "missing_body", "empty", "incomplete_event", "prefix_limit", "read_error",
		"unexpected_field", "multiple_data", "invalid_json", "event_mismatch", "unexpected_event",
		"invalid_response_id", "invalid_error", "non_sse_json", "non_sse_html", "quota_rejected":
		return reason
	case "":
		return ""
	default:
		return "unknown"
	}
}
