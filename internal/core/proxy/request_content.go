package proxy

import (
	"context"
	"regexp"
	"time"
	"unicode/utf8"
)

const (
	// MaxRequestContentBytes 是单条请求正文的保存上限；超出时截断并标记，不推测剩余内容。
	MaxRequestContentBytes = 256 << 10
	contentWriteTimeout    = 5 * time.Second
)

// inlineDataURL 匹配 JSON 字符串里的长 Base64 内联数据（图片、音频等）。
// Base64 字符集不含引号与反斜杠，因此在原始 JSON 文本上替换不会破坏结构。
// Go 的 regexp 重复次数上限是 1000，所以以 1000 个字符（约 750 字节原始数据）为折叠阈值。
var inlineDataURL = regexp.MustCompile(`data:[A-Za-z0-9][A-Za-z0-9.+-]*/[A-Za-z0-9][A-Za-z0-9.+-]*;base64,[A-Za-z0-9+/=]{1000,}`)

// RequestContentEvent 是一次请求正文的持久化事件；正文只在开启记录后产生。
// 不包含凭据、原始头部或 User-Agent。
type RequestContentEvent struct {
	RequestID     string
	Content       string
	OriginalBytes int
	Truncated     bool
}

// RequestContentObserver 保存开启记录后的请求正文；未实现时正文不落库。
type RequestContentObserver interface {
	RecordRequestContent(context.Context, RequestContentEvent) error
}

// recordRequestContent 按设置决定是否采集；设置读取失败按不记录处理，
// 但会留下稳定错误码，请求本身继续。
func (s *Server) recordRequestContent(ctx context.Context, trace *Trace, body []byte) {
	if s.opts.RecordRequestContent == nil || trace == nil || len(body) == 0 {
		return
	}
	enabled, err := s.opts.RecordRequestContent(ctx)
	if err != nil {
		if s.opts.Logger != nil {
			s.opts.Logger.Warn("request content setting unavailable",
				"module", "gateway", "request_id", trace.ID(), "code", "content_setting_unavailable")
		}
		return
	}
	if enabled {
		trace.RecordContent(ctx, body)
	}
}

// RecordContent 保存一份有界正文副本；重复调用只写入一次。
// 采集失败只进入诊断日志与结束日志字段，不影响请求结果。
func (t *Trace) RecordContent(ctx context.Context, body []byte) {
	if t == nil || len(body) == 0 {
		return
	}
	observer, ok := t.observer.(RequestContentObserver)
	if !ok {
		return
	}
	content, truncated, original := prepareRequestContent(body)
	if content == "" {
		return
	}
	t.mu.Lock()
	if t.contentRecorded {
		t.mu.Unlock()
		return
	}
	t.contentRecorded = true
	id, log := t.event.ID, t.logger
	t.mu.Unlock()
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), contentWriteTimeout)
	defer cancel()
	if err := observer.RecordRequestContent(writeCtx, RequestContentEvent{
		RequestID: id, Content: content, OriginalBytes: original, Truncated: truncated,
	}); err != nil {
		t.mu.Lock()
		t.contentFailed = true
		t.mu.Unlock()
		if log != nil {
			log.Warn("request content not stored", "module", "gateway", "request_id", id, "code", "content_unavailable")
		}
	}
}

// prepareRequestContent 折叠长 Base64 内联数据并按 UTF-8 边界截断。
// original 是客户端原始正文字节数，用于说明是否被截断。
// 压缩或非 UTF-8 的透明转发正文无法在不引入解压风险的前提下还原，返回空内容表示不保存。
func prepareRequestContent(body []byte) (string, bool, int) {
	original := len(body)
	if !utf8.Valid(body) {
		return "", false, original
	}
	content := inlineDataURL.ReplaceAll(body, []byte("data:application/octet-stream;base64,<omitted>"))
	truncated := false
	if len(content) > MaxRequestContentBytes {
		content = content[:truncateBoundary(content, MaxRequestContentBytes)]
		truncated = true
	}
	return string(content), truncated, original
}

// truncateBoundary 返回不超过 limit 且落在完整字符边界上的长度。
func truncateBoundary(data []byte, limit int) int {
	end := limit
	for end > 0 && !utf8.RuneStart(data[end]) {
		end--
	}
	return end
}
