package proxy

import (
	"bytes"
	"context"
	"io"
	"mime"
	"net/http"
)

// passthroughSniffBytes 是上游未声明内容类型时用于判定流形态的前缀上限。
const passthroughSniffBytes = 4 << 10

// ObservePassthroughResponse 让透明转发也能统计用量。
//
// 上游声明了 SSE／JSON 时沿用同一套观察器；官方 Codex 的流式响应可能不带
// Content-Type，此时只读一段前缀判定形态，正文再按原字节继续转发，
// 既不重新编码也不改变客户端收到的内容。
func ObservePassthroughResponse(ctx context.Context, trace *Trace, response Response, protocol string) Response {
	if observableMediaType(response.ContentType) || response.Body == nil || response.StatusCode != http.StatusOK {
		return ObserveResponse(ctx, trace, response, protocol)
	}
	body, media, reason := sniffPassthroughStream(response.Body)
	response.Body = body
	if media != "sse" && media != "json" {
		return ObserveResponse(ctx, trace, response, protocol)
	}
	response.ContentTypeInferred, response.ContentTypeInferenceReason = true, reason
	trace.observeStatus(response.StatusCode)
	trace.observeResponseHeaders(response)
	response.Body = &usageReader{reader: body, trace: trace, ctx: ctx, protocol: protocol, stream: media == "sse"}
	return response
}

// observableMediaType 判定上游声明的类型是否已经可以直接观察。
func observableMediaType(value string) bool {
	kind, _, err := mime.ParseMediaType(value)
	return err == nil && (kind == "text/event-stream" || kind == "application/json")
}

// sniffPassthroughStream 只读第一段可用字节判定形态，再把前缀与剩余正文接回去。
// 前缀上限很小，不会为判定而整段缓冲流式响应。
func sniffPassthroughStream(body io.Reader) (io.Reader, string, string) {
	buffered := make([]byte, passthroughSniffBytes)
	n, _ := body.Read(buffered)
	if n <= 0 {
		return body, "other", ""
	}
	rest := io.MultiReader(bytes.NewReader(buffered[:n]), body)
	prefix := bytes.TrimSpace(buffered[:n])
	switch {
	case bytes.HasPrefix(prefix, []byte("data:")),
		bytes.HasPrefix(prefix, []byte("event:")),
		bytes.HasPrefix(prefix, []byte(":")):
		return rest, "sse", "verified"
	case bytes.HasPrefix(prefix, []byte("{")):
		return rest, "json", "non_sse_json"
	case bytes.HasPrefix(prefix, []byte("<")):
		return rest, "html", "non_sse_html"
	default:
		return rest, "other", ""
	}
}
