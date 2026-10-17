package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"github.com/coder/websocket"
	"net/http"
	"strings"
	"time"
)

// serveResponsesSocket 支持内置 Codex 的 WebSocket，连接上下文仅在内存中保存。
func (s *Server) serveResponsesSocket(w http.ResponseWriter, r *http.Request, key string) {
	select {
	case s.socketSlots <- struct{}{}:
		defer func() { <-s.socketSlots }()
	default:
		writeFailure(w, Failure("gateway_busy", 429))
		return
	}
	socket, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer socket.CloseNow()
	socket.SetReadLimit(MaxRequestBytes)
	var history []json.RawMessage
	lastID := ""
	for {
		readCtx, readCancel := context.WithTimeout(r.Context(), 60*time.Second)
		_, body, err := socket.Read(readCtx)
		readCancel()
		if err != nil {
			return
		}
		if s.opts.Authorize != nil && !s.opts.Authorize(r.Context(), key) {
			socketError(r.Context(), socket, "unauthorized")
			return
		}
		// 已建立连接也必须检查当前策略，关闭开关后不能继续通过旧连接创建响应。
		if err := s.webSocketPolicy(r.Context()); err != nil {
			socketError(r.Context(), socket, err.Code)
			return
		}
		var fields map[string]json.RawMessage
		if json.Unmarshal(body, &fields) != nil || string(fields["type"]) != `"response.create"` {
			socketError(r.Context(), socket, "invalid_request")
			return
		}
		delete(fields, "type")
		fields["stream"] = json.RawMessage("true")
		var previous string
		json.Unmarshal(fields["previous_response_id"], &previous)
		var input []json.RawMessage
		if json.Unmarshal(fields["input"], &input) != nil {
			socketError(r.Context(), socket, "invalid_messages")
			return
		}
		if previous != "" {
			if previous != lastID {
				socketError(r.Context(), socket, "unknown_previous_response")
				return
			}
			input = append(append([]json.RawMessage{}, history...), input...)
		}
		delete(fields, "previous_response_id")
		fields["input"], err = json.Marshal(input)
		if err != nil {
			return
		}
		body, err = json.Marshal(fields)
		if err != nil || len(body) > MaxRequestBytes {
			socketError(r.Context(), socket, "request_too_large")
			return
		}
		request, err := ParseProtocolRequest(body, "/v1/responses")
		if err != nil {
			socketError(r.Context(), socket, "invalid_request")
			return
		}
		select {
		case s.slots <- struct{}{}:
		default:
			socketError(r.Context(), socket, "gateway_busy")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), s.opts.RequestTimeout)
		trace, traceErr := s.beginTrace(ctx, "responses", "websocket")
		if traceErr != nil {
			cancel()
			<-s.slots
			socketError(r.Context(), socket, "accounting_unavailable")
			return
		}
		request.Trace = trace
		err = trace.Update(ctx, request, TargetSnapshot{})
		if router, ok := s.opts.Upstream.(RequestRouter); ok && err == nil {
			request, err = router.RouteRequest(ctx, request)
		} else if !s.supportsModel(request.Model) {
			err = Failure("unsupported_model", 400)
		}
		completed := false
		nextID := ""
		var output []json.RawMessage
		if err == nil {
			err = s.opts.Upstream.Forward(ctx, request, func(response Response) error {
				if response.StatusCode != 200 {
					return Failure("upstream_rejected_request", 502)
				}
				scanner := bufio.NewScanner(response.Body)
				scanner.Buffer(make([]byte, 4096), 2<<20)
				total := 0
				for scanner.Scan() {
					line := scanner.Text()
					total += len(line) + 1
					if total > maxStreamBytes {
						return Failure("response_too_large", 502)
					}
					if !strings.HasPrefix(line, "data:") {
						continue
					}
					data := []byte(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
					if string(data) == "[DONE]" {
						continue
					}
					var event struct {
						Type     string `json:"type"`
						Response struct {
							ID     string            `json:"id"`
							Output []json.RawMessage `json:"output"`
						} `json:"response"`
					}
					if json.Unmarshal(data, &event) != nil || hasUpstreamError(data) {
						return Failure("upstream_stream_failed", 502)
					}
					if event.Type == "response.completed" {
						completed = true
						nextID = event.Response.ID
						output = event.Response.Output
					}
					if err := socket.Write(ctx, websocket.MessageText, data); err != nil {
						return err
					}
					if trace.hasFirstContent(data) {
						trace.markFirstToken()
					}
				}
				return scanner.Err()
			})
		}
		code := ""
		status := 200
		if err != nil {
			failure := safeFailure(ctx, err).(*Error)
			code, status = failure.Code, failure.StatusCode
		} else if !completed {
			code, status = "upstream_stream_failed", 502
		}
		trace.completeLogged(status, code, s.opts.Logger)
		cancel()
		<-s.slots
		if err != nil || !completed {
			socketError(r.Context(), socket, "upstream_stream_failed")
			return
		}
		history = append(input, output...)
		lastID = nextID
	}
}
func socketError(ctx context.Context, socket *websocket.Conn, code string) {
	data, _ := json.Marshal(map[string]any{"type": "error", "error": map[string]string{"type": "invalid_request_error", "code": code, "message": code}})
	_ = socket.Write(ctx, websocket.MessageText, data)
}
