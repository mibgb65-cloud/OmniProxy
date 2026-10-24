package proxy

import (
	"context"
	"encoding/json"
	"github.com/coder/websocket"
	"net/http"
	"time"
)

// socketAccess 记录握手时实际使用的授权方式，供连接内逐帧复核复用。
//
// byPrefix 为真表示该连接由固定入口前缀放行（Codex 保留官方登录令牌，客户端不写接入 Key）；
// 这种情况下 key 是官方登录令牌，既不是接入 Key 也不能进入接入 Key 比较。
type socketAccess struct {
	key      string
	prefix   string
	client   string
	byPrefix bool
}

// reauthorize 按与握手一致的判定复核当前连接，返回最新的客户端归属。
func (s *Server) reauthorize(ctx context.Context, access socketAccess) (socketAccess, error) {
	if access.byPrefix {
		if s.opts.PrefixAccess == nil {
			return access, Failure("unauthorized", 401)
		}
		client, ok := s.opts.PrefixAccess(ctx, access.prefix)
		if !ok {
			return access, Failure("unauthorized", 401)
		}
		access.client = client
		return access, nil
	}
	if s.opts.Authorize == nil {
		return access, nil
	}
	// 超过上限的凭据绝不进入比较，与握手阶段保持一致。
	if len(access.key) > 512 {
		return access, Failure("unauthorized", 401)
	}
	client, ok := s.opts.Authorize(ctx, access.key)
	if !ok {
		return access, Failure("unauthorized", 401)
	}
	access.client = client
	return access, nil
}

// serveResponsesSocket 支持内置 Codex 的 WebSocket，连接上下文仅在内存中保存。
func (s *Server) serveResponsesSocket(w http.ResponseWriter, r *http.Request, access socketAccess, metadata CodexMetadata) {
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
	socket.SetReadLimit(MaxResponsesRequestBytes)
	var history []json.RawMessage
	lastID := ""
	for {
		readCtx, readCancel := context.WithTimeout(r.Context(), 60*time.Second)
		_, body, err := socket.Read(readCtx)
		readCancel()
		if err != nil {
			return
		}
		// 连接期间授权状态仍可能变化（接入 Key 停用／替换、固定入口停用），
		// 每帧按握手时的同一判定复核并同步客户端归属。
		renewed, reauthErr := s.reauthorize(r.Context(), access)
		if reauthErr != nil {
			socketFailure(r.Context(), socket, safeFailure(r.Context(), reauthErr).(*Error))
			return
		}
		access = renewed
		// 已建立连接也必须检查当前策略，关闭开关后不能继续通过旧连接创建响应。
		if err := s.webSocketPolicy(r.Context()); err != nil {
			socketFailure(r.Context(), socket, err)
			return
		}
		if err := validateJSONSize(body, MaxResponsesRequestBytes); err != nil {
			s.rejectSocketRequest(r.Context(), socket, err)
			return
		}
		var fields map[string]json.RawMessage
		if json.Unmarshal(body, &fields) != nil || string(fields["type"]) != `"response.create"` {
			socketError(r.Context(), socket, "invalid_request")
			return
		}
		generate := true
		if raw, ok := fields["generate"]; ok && (string(raw) == "null" || json.Unmarshal(raw, &generate) != nil) {
			socketError(r.Context(), socket, "invalid_request")
			return
		}
		delete(fields, "generate")
		delete(fields, "type")
		fields["stream"] = json.RawMessage("true")
		var previous string
		if raw, ok := fields["previous_response_id"]; ok && string(raw) != "null" && json.Unmarshal(raw, &previous) != nil {
			socketError(r.Context(), socket, "invalid_request")
			return
		}
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
		if err != nil || len(body) > MaxResponsesRequestBytes {
			socketError(r.Context(), socket, "request_too_large")
			return
		}
		request, err := ParseProtocolRequest(body, "/v1/responses")
		if err != nil {
			s.rejectSocketRequest(r.Context(), socket, err)
			return
		}
		if !generate {
			ctx, cancel := context.WithTimeout(r.Context(), s.opts.RequestTimeout)
			_, err = s.routeSocketRequest(ctx, request)
			id := ""
			if err == nil {
				id, err = socketWarmup(ctx, socket)
			}
			if err != nil {
				failure := safeFailure(ctx, err).(*Error)
				cancel()
				socketFailure(r.Context(), socket, failure)
				return
			}
			cancel()
			history, lastID = input, id
			continue
		}
		request.CodexMetadata, err = frameCodexMetadata(metadata, fields["client_metadata"])
		metadata.TurnMetadata = ""
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
		ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout(request))
		trace, traceErr := s.beginTrace(ctx, "responses", "websocket", access.client)
		if traceErr != nil {
			cancel()
			<-s.slots
			socketError(r.Context(), socket, "accounting_unavailable")
			return
		}
		request.Trace = trace
		err = trace.Update(ctx, request, TargetSnapshot{})
		if err == nil {
			request, err = s.routeSocketRequest(ctx, request)
		}
		completed := false
		var result socketResult
		if err == nil {
			err = s.opts.Upstream.Forward(ctx, request, func(response Response) error {
				var copyErr error
				result, copyErr = copySocketResponse(ctx, socket, response, trace)
				completed = copyErr == nil
				return copyErr
			})
		}
		code := ""
		status := 200
		var failure *Error
		if err != nil {
			failure = safeFailure(ctx, err).(*Error)
			code, status = failure.Code, failure.StatusCode
		} else if !completed {
			code, status = "upstream_stream_failed", 502
			failure = &Error{Code: code, StatusCode: status}
		}
		trace.completeLogged(status, code, s.opts.Logger)
		cancel()
		<-s.slots
		if err != nil || !completed {
			socketFailure(r.Context(), socket, failure)
			return
		}
		history = append(input, result.Output...)
		lastID = result.ID
	}
}
func (s *Server) routeSocketRequest(ctx context.Context, request Request) (Request, error) {
	if router, ok := s.opts.Upstream.(RequestRouter); ok {
		var err error
		request, err = router.RouteRequest(ctx, request)
		if err != nil {
			return request, err
		}
	} else if !s.supportsModel(request.Model) {
		return request, Failure("unsupported_model", 400)
	}
	// 与 HTTP 一样，别名改写后的最终正文也必须受限，预热和正式请求共用此边界。
	if len(request.Body) > MaxResponsesRequestBytes {
		return request, Failure("request_too_large", 413)
	}
	return request, nil
}
