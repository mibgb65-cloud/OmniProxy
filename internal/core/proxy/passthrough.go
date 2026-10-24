package proxy

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"
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
// 执行前额度拒绝可能被 HTTP 200／429 包装在首事件里，必须在写给客户端之前判定。
func (s *Server) forwardPassthrough(w http.ResponseWriter, r *http.Request, request PassthroughRequest, session PassthroughSession, trace *Trace) (int, string) {
	ctx := r.Context()
	var lastStatus int
	var lastErr error
	var quotaRejection *PassthroughOutcome
	for attempt := 0; attempt < maxPassthroughAttempts; attempt++ {
		target, err := session.Next(ctx)
		if err != nil {
			return s.failPassthrough(w, ctx, lastStatus, lastErr, quotaRejection)
		}
		attemptRequest := Request{Model: target.Model, Stream: true}
		// 尝试独立记账：被拒绝的账号不能把父记录的归属改写成自己。
		if beginErr := trace.BeginAttempt(ctx, attemptRequest, target.Snapshot); beginErr != nil {
			clearAttemptHeaders(target.Header)
			session.Done(ctx, target, PassthroughOutcome{})
			writeFailure(w, beginErr)
			return http.StatusServiceUnavailable, "accounting_unavailable"
		}
		resp, err := s.sendPassthrough(ctx, request, target)
		clearAttemptHeaders(target.Header)
		if err != nil {
			_ = trace.EndAttempt(http.StatusBadGateway, "upstream_unavailable")
			session.Done(ctx, target, PassthroughOutcome{})
			lastErr, quotaRejection = err, nil
			continue
		}
		outcome := PassthroughOutcome{StatusCode: resp.StatusCode, RetryAfter: resp.Header.Get("Retry-After")}
		if reset, rejected := inspectPassthroughQuota(resp, time.Now()); rejected {
			// 这段字节是执行前拒绝，不能交付给客户端；按真实重置时间冷却后换号。
			_ = resp.Body.Close()
			outcome.UpstreamQuotaRejected, outcome.QuotaResetAt = true, reset
			_ = trace.EndAttempt(resp.StatusCode, "upstream_quota_exhausted")
			session.Done(ctx, target, outcome)
			lastStatus, lastErr, quotaRejection = resp.StatusCode, nil, &outcome
			if session.Switchable() {
				continue
			}
			break
		}
		if !session.Retryable(target, resp.StatusCode) {
			_ = trace.EndAttempt(resp.StatusCode, "")
			if bindErr := trace.Update(ctx, attemptRequest, target.Snapshot); bindErr != nil {
				_ = resp.Body.Close()
				session.Done(ctx, target, outcome)
				writeFailure(w, bindErr)
				return http.StatusServiceUnavailable, "accounting_unavailable"
			}
			session.Done(ctx, target, outcome)
			return s.writePassthroughResponse(w, r, trace, resp, request.Path)
		}
		_ = trace.EndAttempt(resp.StatusCode, "upstream_rejected_request")
		session.Done(ctx, target, outcome)
		lastStatus, lastErr, quotaRejection = resp.StatusCode, nil, nil
		_ = resp.Body.Close()
	}
	return s.failPassthrough(w, ctx, lastStatus, lastErr, quotaRejection)
}

// failPassthrough 复刻 v1：保留最后一次上游状态码，否则返回网关失败码。
// 全部候选都在执行前被额度拒绝时，客户端统一收到受控 429；请求明细保留真实上游状态码。
func (s *Server) failPassthrough(w http.ResponseWriter, ctx context.Context, lastStatus int, err error, rejected *PassthroughOutcome) (int, string) {
	if rejected != nil {
		writeFailure(w, &Error{Code: "upstream_quota_exhausted", StatusCode: http.StatusTooManyRequests, RetryAfter: quotaRetryAfter(rejected.QuotaResetAt, time.Now())})
		return rejected.StatusCode, "upstream_quota_exhausted"
	}
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

// quotaRetryAfter 把重置时间换算成受控的 Retry-After 秒数；未知或越界时留空。
func quotaRetryAfter(reset time.Time, now time.Time) string {
	if !reset.After(now) {
		return ""
	}
	seconds := int((reset.Sub(now) + time.Second - 1) / time.Second)
	if seconds < 1 || seconds > 86400 {
		return ""
	}
	return strconv.Itoa(seconds)
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
