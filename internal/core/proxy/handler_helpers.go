package proxy

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
)

func (s *Server) allowLocalRequest(r *http.Request) bool {
	// Electron 的原生连接探针发送 none，不代表浏览器跨站请求。
	// Origin 仍必须为空，且继续校验 loopback Host、端口及独立接入 Key。
	site := r.Header.Get("Sec-Fetch-Site")
	if r.Header.Get("Origin") != "" || (site != "" && site != "none" && site != "same-origin") {
		return false
	}
	status := s.Status()
	host, port, err := net.SplitHostPort(r.Host)
	return err == nil && (host == "127.0.0.1" || host == "localhost") && port == strconv.Itoa(status.Port)
}

func (s *Server) supportsModel(model string) bool {
	for _, supported := range s.opts.Models {
		if model == supported {
			return true
		}
	}
	return false
}

func safeFailure(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return Failure("request_timeout", 504)
	}
	if ctx.Err() != nil {
		return Failure("request_cancelled", 499)
	}
	var failure *Error
	if errors.As(err, &failure) {
		return failure
	}
	return Failure("upstream_unavailable", 502)
}

func writeFailure(w http.ResponseWriter, err error) {
	var failure *Error
	if !errors.As(err, &failure) {
		failure = &Error{Code: "upstream_unavailable", StatusCode: 502}
	}
	if tracked, ok := w.(*trackedWriter); ok {
		tracked.code = failure.Code
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(failure.StatusCode)
	// 错误只返回稳定代码，禁止携带上游正文、账号名称、URL 或底层错误文本。
	if _, err := io.WriteString(w, `{"error":{"message":"`+failure.Code+`","type":"gateway_error","code":"`+failure.Code+`"}}`); err != nil {
		slog.Debug("gateway error response write failed", "code", "client_write_failed")
	}
}
