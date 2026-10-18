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
	return []any{"upstream_media_type", t.upstreamMediaType, "upstream_content_type_inferred", t.contentTypeInferred}
}
