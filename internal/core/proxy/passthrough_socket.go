package proxy

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// servePassthroughSocket 与 v1 一致：真正拨到上游 wss，然后双向原样中继帧。
func (s *Server) servePassthroughSocket(w http.ResponseWriter, r *http.Request, path string, access socketAccess) {
	ctx, cancel := s.socketHandshakeContext(r.Context())
	defer cancel()
	finish := BeginSocketStage(ctx, "handshake")
	fail := func(err error) {
		err = safeFailure(ctx, err)
		finish(err)
		s.logSocketFailure(err)
		// 客户端已离开时不再补写一个并未送达的 HTTP 错误。
		if r.Context().Err() == nil {
			writeFailure(w, err)
		}
	}
	policy := func(ctx context.Context) error {
		if err := s.webSocketPolicy(ctx); err != nil {
			return err
		}
		current, err := s.reauthorize(ctx, access)
		if err == nil && current.client != access.client {
			return Failure("client_mismatch", 403)
		}
		return err
	}
	if err := policy(ctx); err != nil {
		fail(err)
		return
	}
	select {
	case s.socketSlots <- struct{}{}:
		defer func() { <-s.socketSlots }()
	default:
		fail(Failure("gateway_busy", 429))
		return
	}
	request := PassthroughRequest{
		Method: r.Method, Path: path, Query: r.URL.RawQuery,
		Client: access.client, Header: cloneHeader(r.Header),
	}
	stage := BeginSocketStage(ctx, "selection")
	session, err := s.opts.Passthrough.OpenPassthrough(ctx, request)
	stage(err)
	if err != nil {
		fail(err)
		return
	}
	subprotocols := requestedSubprotocols(r)
	upstream, attempt, err := s.dialPassthroughSocket(r.WithContext(ctx), session, request, subprotocols)
	if err != nil {
		fail(err)
		return
	}
	defer upstream.CloseNow()
	if err := ctx.Err(); err != nil {
		fail(err)
		return
	}
	if err := policy(ctx); err != nil {
		fail(err)
		return
	}
	stage = BeginSocketStage(ctx, "downstream_accept")
	socket, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// 本机入口已在 allowLocalRequest 校验 Host 与 Origin，不再重复限制来源。
		InsecureSkipVerify: true,
		Subprotocols:       subprotocols,
		CompressionMode:    websocket.CompressionContextTakeover,
	})
	stage(err)
	if err != nil {
		finish(err)
		return
	}
	finish(nil)
	// 上游 Dial 的上下文只用于建连；中继始终使用原始请求生命周期。
	cancel()
	defer socket.CloseNow()
	socket.SetReadLimit(MaxResponsesRequestBytes)
	upstream.SetReadLimit(MaxResponsesRequestBytes)
	relayContext := context.WithValue(r.Context(), socketHandshakeKey{}, ctx.Value(socketHandshakeKey{}))
	stage = BeginSocketStage(relayContext, "relay")
	accounting := newSocketAccounting(s, access, attempt, path)
	relayErr := relaySocket(relayContext, socket, upstream, policy, accounting)
	accounting.close(relayContext, relayErr)
	stage(relayErr)
}

// dialPassthroughSocket 复刻 v1：握手被明确拒绝时换同池账号重拨，成功即停止。
// 客户端握手只在拿到上游之后才应答，因此中途失败仍能返回真实状态码。
// 成功时一并交回握手的尝试身份，让逐轮记账绑定真实账号。
//
// 失败必须原样交出原因：账号不可用、凭据借不出、URL 不可用都被折叠成
// 502 upstream_unavailable 时，客户端只会把它当成上游抖动而无限重连。
func (s *Server) dialPassthroughSocket(r *http.Request, session PassthroughSession, request PassthroughRequest, subprotocols []string) (*websocket.Conn, PassthroughAttempt, error) {
	prepareRemaining := 5 * time.Second
	var lastErr error
	for attempt := 0; attempt < maxPassthroughAttempts; attempt++ {
		if err := r.Context().Err(); err != nil {
			return nil, PassthroughAttempt{}, err
		}
		if prepareRemaining <= 0 {
			return nil, PassthroughAttempt{}, Failure("credential_prepare_timeout", 504)
		}
		prepareCtx, prepareCancel := context.WithTimeout(r.Context(), prepareRemaining)
		stage := BeginSocketStage(prepareCtx, "credentials")
		start := time.Now()
		target, err := session.Next(prepareCtx)
		prepareRemaining -= time.Since(start)
		prepareErr := prepareCtx.Err()
		stageErr := err
		if stageErr == nil {
			stageErr = prepareErr
		}
		stage(stageErr)
		prepareCancel()
		if err == nil && (prepareErr != nil || r.Context().Err() != nil) {
			clearAttemptHeaders(target.Header)
			session.Done(r.Context(), target, PassthroughOutcome{})
		}
		if err != nil || prepareErr != nil || r.Context().Err() != nil {
			if r.Context().Err() != nil {
				return nil, PassthroughAttempt{}, r.Context().Err()
			}
			if prepareErr != nil {
				return nil, PassthroughAttempt{}, Failure("credential_prepare_timeout", 504)
			}
			return nil, PassthroughAttempt{}, err
		}
		targetURL, urlErr := PassthroughSocketURL(target.BaseURL, request.Path, request.Query)
		if urlErr != nil {
			clearAttemptHeaders(target.Header)
			session.Done(r.Context(), target, PassthroughOutcome{})
			return nil, PassthroughAttempt{}, urlErr
		}
		dialCtx, dialCancel := context.WithTimeout(r.Context(), 5*time.Second)
		stage = BeginSocketStage(dialCtx, "upstream_dial")
		upstream, upstreamResp, dialErr := websocket.Dial(dialCtx, targetURL, &websocket.DialOptions{
			HTTPHeader:      socketAttemptHeaders(request.Header, target.Header),
			Subprotocols:    subprotocols,
			CompressionMode: websocket.CompressionContextTakeover,
		})
		stage(dialErr)
		dialErrContext := dialCtx.Err()
		dialCancel()
		clearAttemptHeaders(target.Header)
		status := 0
		if upstreamResp != nil {
			status = upstreamResp.StatusCode
			// 握手失败时客户端库可能已消费正文，Body 为空是正常情况。
			if upstreamResp.Body != nil {
				_ = upstreamResp.Body.Close()
			}
		}
		if dialErr == nil && dialErrContext == nil && r.Context().Err() == nil {
			session.Done(r.Context(), target, PassthroughOutcome{StatusCode: http.StatusSwitchingProtocols})
			return upstream, target, nil
		}
		if upstream != nil {
			upstream.CloseNow()
		}
		session.Done(r.Context(), target, PassthroughOutcome{StatusCode: status, RetryAfter: socketRetryAfter(upstreamResp)})
		if r.Context().Err() != nil {
			return nil, PassthroughAttempt{}, r.Context().Err()
		}
		if dialErrContext != nil {
			return nil, PassthroughAttempt{}, Failure("upstream_socket_timeout", 504)
		}
		lastErr = socketDialFailure(status)
		if status == 0 || !session.Retryable(target, status) {
			return nil, PassthroughAttempt{}, lastErr
		}
	}
	if lastErr == nil {
		lastErr = Failure("upstream_unavailable", 502)
	}
	return nil, PassthroughAttempt{}, lastErr
}

// socketDialFailure 把握手结果映射成稳定错误码：有状态码说明上游明确拒绝，
// 没有状态码说明连接根本没建立，两者不能混为一谈。
func socketDialFailure(status int) error {
	if status != 0 {
		return &Error{Code: "upstream_socket_rejected", StatusCode: status}
	}
	return Failure("upstream_unavailable", 502)
}

// socketRetryAfter 只读取握手失败响应里的重试提示，冷却语义仍由宿主决定。
func socketRetryAfter(response *http.Response) string {
	if response == nil {
		return ""
	}
	return response.Header.Get("Retry-After")
}

// logSocketFailure 只写稳定分类与内部错误码，不带 URL、账号或底层错误文本。
func (s *Server) logSocketFailure(err error) {
	if s.opts.Logger == nil {
		return
	}
	code, status := "upstream_unavailable", 502
	var failure *Error
	if errors.As(err, &failure) {
		code, status = failure.Code, failure.StatusCode
	}
	s.opts.Logger.Warn("gateway websocket failed", "module", "gateway", "code", code, "status_code", status)
}

// relaySocket 任一方向结束即关闭整对连接；解码与协议语义完全交给两端。
//
// guard 在转发每个客户端帧之前复核当前策略：关掉 WebSocket 开关后，
// 已经建立的连接也不能继续创建新的响应，与协议路径的逐帧复核一致。
// accounting 只在已转发的帧上旁路记账：客户端帧先记账再转发，上游帧先转发再统计。
func relaySocket(ctx context.Context, client, upstream *websocket.Conn, guard func(context.Context) error, accounting *socketAccounting) error {
	ctx, cancel := context.WithCancel(ctx)
	var once sync.Once
	closeBoth := func() {
		once.Do(func() {
			cancel()
			client.CloseNow()
			upstream.CloseNow()
		})
	}
	defer closeBoth()
	results := make(chan error, 1)
	go func() {
		defer closeBoth()
		for {
			kind, data, err := client.Read(ctx)
			if err != nil {
				results <- socketRelayFailure(ctx, err, "client_disconnected")
				return
			}
			if guard != nil {
				if err := guard(ctx); err != nil {
					results <- err
					return
				}
			}
			// 记账失败就不把这一轮送给上游：未记录的上游消费不能发生。
			if err := accounting.clientFrame(ctx, kind, data); err != nil {
				results <- err
				return
			}
			if err := upstream.Write(ctx, kind, data); err != nil {
				results <- socketRelayFailure(ctx, err, "upstream_write_failed")
				return
			}
		}
	}()
	for {
		kind, data, err := upstream.Read(ctx)
		if err != nil {
			select {
			case err := <-results:
				return err
			default:
				return socketRelayFailure(ctx, err, "upstream_disconnected")
			}
		}
		if err := client.Write(ctx, kind, data); err != nil {
			return socketRelayFailure(ctx, err, "client_write_failed")
		}
		// 字节已经交给客户端，统计失败不能再改变这一轮的交付结果。
		accounting.upstreamFrame(ctx, kind, data)
	}
}

func socketRelayFailure(ctx context.Context, err error, code string) error {
	status := websocket.CloseStatus(err)
	event := &socketRelayError{direction: code, closeCode: status}
	if status == websocket.StatusNormalClosure || status == websocket.StatusGoingAway {
		return event
	}
	if ctx.Err() != nil {
		event.failure = ctx.Err()
	} else {
		event.failure = Failure(code, 502)
	}
	return event
}

type socketRelayError struct {
	failure   error
	direction string
	closeCode websocket.StatusCode
}

func (e *socketRelayError) Error() string {
	if e.failure == nil {
		return "websocket: normal_close"
	}
	return e.failure.Error()
}

func (e *socketRelayError) Unwrap() error { return e.failure }

// requestedSubprotocols 只读取客户端声明的子协议，不新增或协商自定义协议。
func requestedSubprotocols(r *http.Request) []string {
	values := r.Header.Values("Sec-Websocket-Protocol")
	protocols := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				protocols = append(protocols, trimmed)
			}
		}
	}
	return protocols
}

// socketAttemptHeaders 复刻 v1：客户端头整体带到上游握手，只去掉由握手自身管理的头；
// 子协议改由 DialOptions 协商，Origin 与 Host 不随客户端透传。
func socketAttemptHeaders(client, attempt http.Header) http.Header {
	header := make(http.Header, len(client)+len(attempt))
	for key, values := range client {
		if skipPassthroughHeader(key) || socketManagedHeader(key) {
			continue
		}
		for _, value := range values {
			header.Add(key, value)
		}
	}
	applyAttemptHeaders(header, attempt)
	return header
}

func socketManagedHeader(key string) bool {
	switch http.CanonicalHeaderKey(key) {
	case "Host", "Origin",
		"Sec-Websocket-Accept", "Sec-Websocket-Extensions", "Sec-Websocket-Key",
		"Sec-Websocket-Protocol", "Sec-Websocket-Version":
		return true
	default:
		return false
	}
}
