package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"mime"
	"strings"

	"github.com/coder/websocket"
)

type socketResult struct {
	ID     string            `json:"id"`
	Output []json.RawMessage `json:"output"`
}

func copySocketResponse(ctx context.Context, socket *websocket.Conn, response Response, trace *Trace) (socketResult, error) {
	if response.PreExecutionQuotaRejected || response.StatusCode != 200 {
		return socketResult{}, upstreamFailure(response)
	}
	contentType, _, err := mime.ParseMediaType(response.ContentType)
	if err != nil || contentType != "text/event-stream" || response.Body == nil {
		return socketResult{}, Failure("invalid_upstream_response", 502)
	}
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 4096), MaxResponsesEventBytes)
	total := 0
	for scanner.Scan() {
		line := scanner.Text()
		total += len(line) + 1
		if total > maxStreamBytes {
			return socketResult{}, Failure("response_too_large", 502)
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := []byte(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		if string(data) == "[DONE]" {
			continue
		}
		var event struct {
			Type     string       `json:"type"`
			Response socketResult `json:"response"`
		}
		if json.Unmarshal(data, &event) != nil || hasUpstreamError(data) {
			return socketResult{}, Failure("upstream_stream_failed", 502)
		}
		if err := socket.Write(ctx, websocket.MessageText, data); err != nil {
			return socketResult{}, Failure("client_write_failed", 502)
		}
		if trace.hasFirstContent(data) {
			trace.markFirstToken()
		}
		// 完成事件就是流结束边界，不能等待上游关闭连接后才允许下一轮请求。
		if event.Type == "response.completed" {
			return event.Response, nil
		}
	}
	return socketResult{}, streamReadFailure(scanner.Err())
}

func socketError(ctx context.Context, socket *websocket.Conn, code string) {
	status := 400
	switch code {
	case "unauthorized":
		status = 401
	case "websocket_disabled":
		status = 403
	case "gateway_busy":
		status = 429
	case "request_too_large":
		status = 413
	case "accounting_unavailable":
		status = 503
	}
	socketFailure(ctx, socket, &Error{Code: code, StatusCode: status})
}

func (s *Server) rejectSocketRequest(ctx context.Context, socket *websocket.Conn, err error) {
	failure := safeFailure(ctx, err).(*Error)
	if s.opts.Logger != nil {
		s.opts.Logger.Warn("gateway request rejected", "protocol", "responses", "transport", "websocket", "status_code", failure.StatusCode, "code", failure.Code)
	}
	socketFailure(ctx, socket, failure)
}

func socketFailure(ctx context.Context, socket *websocket.Conn, failure *Error) {
	event := map[string]any{"type": "error", "status": failure.StatusCode, "error": map[string]string{"type": "gateway_error", "code": failure.Code, "message": failure.Code}}
	if seconds := validatedRetryAfter(failure.RetryAfter); seconds != "" {
		event["headers"] = map[string]string{"retry-after": seconds}
	}
	data, err := json.Marshal(event)
	if err == nil {
		_ = socket.Write(ctx, websocket.MessageText, data)
	}
}
