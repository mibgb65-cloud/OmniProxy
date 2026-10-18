package gateway

import (
	"context"
	"omniproxy/internal/core/proxy"
)

func (s *Service) forwardKimi(ctx context.Context, target target, request proxy.Request, consume func(proxy.Response) error) error {
	_, ok := s.accounts.(interface {
		UseAPIKeyAt(context.Context, string, string, func([]byte) error) error
	})
	if !ok || target.kimi == nil {
		return proxy.Failure("invalid_configuration", 503)
	}
	return s.useAPIKey(ctx, target, func(key []byte) error {
		return s.kimi.Forward(ctx, *target.kimi, key, request, func(response proxy.Response) error {
			return consume(proxy.ObserveResponse(ctx, request.Trace, response, request.Protocol))
		})
	})
}
