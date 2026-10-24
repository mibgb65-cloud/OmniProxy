package proxy

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

const passthroughChunkBytes = 32 << 10

// passthroughHeaderTimeout 只限制响应头到达时间，流式正文不受影响。
const passthroughHeaderTimeout = 60 * time.Second

// sendPassthrough 用客户端原始头部发起一次上游请求，只替换鉴权头。
func (s *Server) sendPassthrough(ctx context.Context, request PassthroughRequest, target PassthroughAttempt) (*http.Response, error) {
	targetURL, err := PassthroughURL(target.BaseURL, request.Path, request.Query)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, request.Method, targetURL, bytes.NewReader(request.Body))
	if err != nil {
		return nil, Failure("invalid_request", 400)
	}
	copyPassthroughHeaders(req.Header, request.Header)
	applyAttemptHeaders(req.Header, target.Header)
	req.Host = req.URL.Host
	resp, err := s.passthrough.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, safeFailure(ctx, err)
		}
		return nil, Failure("upstream_unavailable", 502)
	}
	return resp, nil
}

// writePassthroughResponse 把上游状态、头部与正文原样写给客户端。
func (s *Server) writePassthroughResponse(w http.ResponseWriter, r *http.Request, trace *Trace, resp *http.Response, path string) (int, string) {
	defer resp.Body.Close()
	removePassthroughResponseHeaders(resp.Header)
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	body := io.Reader(resp.Body)
	if resp.StatusCode == http.StatusOK {
		observed := ObservePassthroughResponse(r.Context(), trace, Response{
			StatusCode: resp.StatusCode, ContentType: resp.Header.Get("Content-Type"), Body: resp.Body,
		}, passthroughProtocol(path))
		if observed.Body != nil {
			body = observed.Body
		}
	}
	if result := copyPassthroughStream(w, body); result.err != nil {
		trace.markStreamSide(result.side)
		return resp.StatusCode, passthroughOutcome(r, trace, result)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return resp.StatusCode, "upstream_rejected_request"
	}
	return resp.StatusCode, ""
}

// passthroughCopy 是一次流式转发的结束状态：出错侧决定这次失败的真实归属。
type passthroughCopy struct {
	written int64
	// side 记录出错的一侧：client（写客户端）或 upstream（读上游）。
	side string
	err  error
}

// passthroughOutcome 只按可证实的信号分类，不把连接结束一律算成客户端写失败：
// 上游已交付协议终结事件 → 这一轮已经完整交付；客户端连接结束 → 取消；
// 其余按出错侧归类，把上游读流问题标成 upstream_stream_failed。
func passthroughOutcome(r *http.Request, trace *Trace, result passthroughCopy) string {
	if trace.terminalSeen() {
		return ""
	}
	if r.Context().Err() != nil || result.side == "client" {
		return "request_cancelled"
	}
	return "upstream_stream_failed"
}

// copyPassthroughStream 逐块转发并立即刷新，保持 SSE 的首字延迟，
// 同时记录出错侧，避免把上游中断记成客户端写失败。
func copyPassthroughStream(w http.ResponseWriter, body io.Reader) passthroughCopy {
	controller := http.NewResponseController(w)
	buffer := make([]byte, passthroughChunkBytes)
	var result passthroughCopy
	for {
		n, err := body.Read(buffer)
		if n > 0 {
			written, writeErr := w.Write(buffer[:n])
			result.written += int64(written)
			if writeErr != nil {
				result.side, result.err = "client", writeErr
				return result
			}
			if flushErr := controller.Flush(); flushErr != nil && !errors.Is(flushErr, http.ErrNotSupported) {
				result.side, result.err = "client", flushErr
				return result
			}
		}
		if err == io.EOF {
			return result
		}
		if err != nil {
			result.side, result.err = "upstream", err
			return result
		}
	}
}

func hasToken(value, token string) bool {
	for _, part := range strings.Split(value, ",") {
		if equalFoldToken(strings.TrimSpace(part), token) {
			return true
		}
	}
	return false
}

func equalFoldToken(value, token string) bool {
	return strings.EqualFold(strings.TrimSpace(value), token)
}
