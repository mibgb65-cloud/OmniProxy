package proxy

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
)

// ServeHTTP 只开放模型列表和 Chat Completions，不提供通用 URL 转发能力。
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	if !s.allowLocalRequest(r) {
		writeFailure(w, Failure("local_access_only", 403))
		return
	}
	path, pathKey, valid := scopedAgentPath(r)
	if !valid {
		writeFailure(w, Failure("not_found", 404))
		return
	}
	if !validModelQuery(path, r.URL.RawQuery) {
		writeFailure(w, Failure("invalid_request", 400))
		return
	}
	key := pathKey
	if s.opts.Authorize != nil {
		if key == "" {
			key = r.Header.Get("X-Api-Key")
		}
		if key == "" {
			key = r.Header.Get("X-OmniProxy-Key")
		}
		if key == "" {
			key = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		}
		if len(key) > 512 || !s.opts.Authorize(r.Context(), key) {
			writeFailure(w, Failure("unauthorized", 401))
			return
		}
	}
	if path == "/v1/responses" && r.Method == http.MethodGet && strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		s.serveResponsesSocket(w, r, key)
		return
	}
	if path == "/v1/models" && r.Method == http.MethodGet {
		s.writeModelCatalog(w, r, false)
		return
	}
	if path == "/v1/codex-models" && r.Method == http.MethodGet {
		s.writeModelCatalog(w, r, true)
		return
	}
	if path != "/v1/chat/completions" && path != "/v1/responses" && path != "/v1/messages" {
		writeFailure(w, Failure("not_found", 404))
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeFailure(w, Failure("method_not_allowed", 405))
		return
	}
	protocol := strings.TrimPrefix(path, "/v1/")
	if protocol == "chat/completions" {
		protocol = "chat"
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		writeFailure(w, Failure("invalid_content_type", 415))
		return
	}
	if r.Header.Get("Content-Encoding") != "" {
		writeFailure(w, Failure("invalid_content_type", 415))
		return
	}
	// 在读取正文前占用并发名额，避免大量并行正文绕过上游并发限制占用内存。
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	default:
		writeFailure(w, Failure("gateway_busy", 429))
		return
	}
	trace, err := s.beginTrace(r.Context(), protocol, "http")
	if err != nil {
		writeFailure(w, err)
		return
	}
	w.Header().Set("X-Request-ID", trace.ID())
	writer := &trackedWriter{ResponseWriter: w}
	w = writer
	defer func() { trace.completeLogged(writer.status, writer.code, s.opts.Logger) }()
	controller := http.NewResponseController(w)
	if err := controller.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		writeFailure(w, Failure("request_read_failed", 400))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeFailure(w, Failure("request_too_large", 413))
		} else {
			writeFailure(w, Failure("request_read_failed", 400))
		}
		return
	}
	if err := controller.SetReadDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
		writeFailure(w, Failure("request_read_failed", 400))
		return
	}
	request, err := ParseProtocolRequest(body, path)
	if err != nil {
		writeFailure(w, err)
		return
	}
	request.Trace = trace
	if err := trace.Update(r.Context(), request, TargetSnapshot{}); err != nil {
		writeFailure(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.opts.RequestTimeout)
	defer cancel()
	if router, ok := s.opts.Upstream.(RequestRouter); ok {
		request, err = router.RouteRequest(ctx, request)
		if err != nil {
			writeFailure(w, safeFailure(ctx, err))
			return
		}
		if len(request.Body) > MaxRequestBytes {
			writeFailure(w, Failure("request_too_large", 413))
			return
		}
	}
	if !s.supportsModel(request.Model) && request.ProviderID == "" {
		writeFailure(w, Failure("unsupported_model", 400))
		return
	}
	committed := false
	err = s.opts.Upstream.Forward(ctx, request, func(response Response) error {
		return copyResponse(w, response, request.Stream, &committed, request.Protocol)
	})
	if err != nil && !committed {
		writeFailure(w, safeFailure(ctx, err))
	}
	if err != nil {
		writer.code = safeFailure(ctx, err).(*Error).Code
	}
	// 已发送响应头后无法改写 HTTP 状态；中断连接，让客户端明确感知未完成的响应。
	if err != nil && committed {
		panic(http.ErrAbortHandler)
	}
}
