package gateway

import (
	"context"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/provider/mimo"
)

func (s *Service) forwardMiMo(ctx context.Context, target target, request proxy.Request, consume func(proxy.Response) error) error {
	_, ok := s.accounts.(interface {
		UseAPIKeyAt(context.Context, string, string, func([]byte) error) error
	})
	if !ok || target.mimo == nil {
		return proxy.Failure("invalid_configuration", 503)
	}
	return s.useAPIKey(ctx, target, func(key []byte) error {
		base, err := target.mimo.BaseURL(target.baseURL)
		if err != nil {
			return proxy.Failure("invalid_configuration", 503)
		}
		if !mimo.ValidKeyForOptions(*target.mimo, key) {
			return proxy.Failure("invalid_credential", 503)
		}
		if !mimo.SupportsAccountModel(*target.mimo, request.Model) {
			return proxy.Failure("unsupported_model", 400)
		}
		return s.mimo.Forward(ctx, base, key, request, func(response proxy.Response) error {
			return consume(proxy.ObserveResponse(ctx, request.Trace, response, request.Protocol))
		})
	})
}
