package proxy

import (
	"context"
	"io"
	"net/http"
)

// maxPassthroughAttempts 是单次透明转发的账号尝试上限，与 v1 同类设置对齐。
const maxPassthroughAttempts = 3

// servePassthrough 按 v1 语义透明转发：路径、头部与正文原样，状态码原样回传。
func (s *Server) servePassthrough(w http.ResponseWriter, r *http.Request, path string, access socketAccess) {
	if socketUpgradeRequested(r) {
		s.servePassthroughSocket(w, r, path, access)
		return
	}
	body, ok := s.readPassthroughBody(w, r)
	if !ok {
		return
	}
	defer clearBytes(body)
	request := PassthroughRequest{
		Method: r.Method, Path: path, Query: r.URL.RawQuery,
		Client: access.client, Header: cloneHeader(r.Header), Body: body,
	}
	session, err := s.opts.Passthrough.OpenPassthrough(r.Context(), request)
	if err != nil {
		writeFailure(w, safeFailure(r.Context(), err))
		return
	}
	trace, err := s.beginTrace(r.Context(), passthroughProtocol(path), "http", recordedClient(access.client, r.Header))
	if err != nil {
		writeFailure(w, err)
		return
	}
	w.Header().Set("X-Request-ID", trace.ID())
	status, code := s.forwardPassthrough(w, r, request, session, trace)
	trace.completeLogged(status, code, s.opts.Logger)
}

// readPassthroughBody 只做容量保护，不解压也不解析：压缩正文按原编码转发。
func (s *Server) readPassthroughBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxPassthroughBytes+1))
	if err != nil {
		writeFailure(w, Failure("request_read_failed", 400))
		return nil, false
	}
	if len(body) > MaxPassthroughBytes {
		writeFailure(w, Failure("request_too_large", 413))
		return nil, false
	}
	return body, true
}

// forwardPassthrough 顺序尝试候选账号；只有宿主判定为明确拒绝才换号。
func (s *Server) forwardPassthrough(w http.ResponseWriter, r *http.Request, request PassthroughRequest, session PassthroughSession, trace *Trace) (int, string) {
	ctx := r.Context()
	var lastStatus int
	var lastErr error
	for attempt := 0; attempt < maxPassthroughAttempts; attempt++ {
		target, err := session.Next(ctx)
		if err != nil {
			return s.failPassthrough(w, ctx, lastStatus, err)
		}
		updateErr := trace.Update(ctx, Request{Model: target.Model, Stream: true}, target.Snapshot)
		if updateErr != nil {
			clearAttemptHeaders(target.Header)
			session.Done(ctx, target, 0)
			writeFailure(w, updateErr)
			return http.StatusServiceUnavailable, "accounting_unavailable"
		}
		resp, err := s.sendPassthrough(ctx, request, target)
		clearAttemptHeaders(target.Header)
		if err != nil {
			session.Done(ctx, target, 0)
			lastErr = err
			continue
		}
		if !session.Retryable(target, resp.StatusCode) {
			session.Done(ctx, target, resp.StatusCode)
			return s.writePassthroughResponse(w, r, trace, resp, request.Path)
		}
		session.Done(ctx, target, resp.StatusCode)
		lastStatus = resp.StatusCode
		_ = resp.Body.Close()
	}
	return s.failPassthrough(w, ctx, lastStatus, lastErr)
}

// failPassthrough 复刻 v1：保留最后一次上游状态码，否则返回网关失败码。
func (s *Server) failPassthrough(w http.ResponseWriter, ctx context.Context, lastStatus int, err error) (int, string) {
	if lastStatus != 0 {
		http.Error(w, http.StatusText(lastStatus), lastStatus)
		return lastStatus, "upstream_rejected_request"
	}
	failure, ok := safeFailure(ctx, err).(*Error)
	if !ok {
		failure = &Error{Code: "upstream_unavailable", StatusCode: http.StatusBadGateway}
	}
	writeFailure(w, failure)
	return failure.StatusCode, failure.Code
}

// clearBytes 抹掉正文副本中的请求内容。
func clearBytes(body []byte) {
	for index := range body {
		body[index] = 0
	}
}

// socketUpgradeRequested 只认 Upgrade: websocket，与客户端实际协商一致。
func socketUpgradeRequested(r *http.Request) bool {
	return hasToken(r.Header.Get("Connection"), "upgrade") && equalFoldToken(r.Header.Get("Upgrade"), "websocket")
}
