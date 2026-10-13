package proxy

import "context"

func (s *Server) webSocketPolicy(ctx context.Context) *Error {
	if s.opts.WebSocketAllowed == nil {
		return nil
	}
	enabled, err := s.opts.WebSocketAllowed(ctx)
	if err != nil {
		return &Error{Code: "configuration_unavailable", StatusCode: 503}
	}
	if !enabled {
		return &Error{Code: "websocket_disabled", StatusCode: 403}
	}
	return nil
}
