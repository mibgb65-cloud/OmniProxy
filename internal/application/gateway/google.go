package gateway

import (
	"context"
	"encoding/json"
	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/provider/antigravity"
	"omniproxy/internal/provider/google"
)

// GoogleSubscriptionAccess 只允许在后端回调内使用订阅令牌。
type GoogleSubscriptionAccess interface {
	Use(context.Context, string, func(antigravity.Credentials) error) error
}

// SetGoogleSubscription 安装独立订阅通道。
func (s *Service) SetGoogleSubscription(service GoogleSubscriptionAccess) {
	s.googleAuth = service
}

type GoogleResourceObserver interface {
	Record(context.Context, string, string, string, google.Failure) error
}

func (s *Service) SetGoogleResources(observer GoogleResourceObserver) { s.googleResources = observer }

func (s *Service) forwardGoogle(ctx context.Context, target target, request proxy.Request, consume func(proxy.Response) error) error {
	if target.authMethod == "subscription_login" {
		if s.googleAuth == nil {
			return proxy.Failure("adapter_unavailable", 503)
		}
		return s.useGoogleToken(ctx, target, func(cred antigravity.Credentials) error {
			return s.forwardGoogleAgent(ctx, cred, request, consume)
		})
	}
	if request.Protocol != "chat" {
		return proxy.Failure("unsupported_protocol", 400)
	}
	if err := s.googleOutputLimit(ctx, target, request); err != nil {
		return err
	}
	return s.useAPIKey(ctx, target, func(key []byte) error {
		return s.google.Forward(ctx, target.baseURL, key, request, func(failure google.Failure) error {
			if s.googleResources == nil {
				return nil
			}
			if err := s.googleResources.Record(ctx, target.accountID, target.updatedAt, request.Model, failure); err != nil {
				if s.log == nil {
					return proxy.Failure("resource_observation_unavailable", 503)
				}
				s.log.Warn("Google resource observation unavailable", "module", "googlequota", "code", "observation_failed")
			}
			return nil
		}, func(response proxy.Response) error {
			return consume(proxy.ObserveResponse(ctx, request.Trace, response, "chat"))
		})
	})
}

func (s *Service) googleOutputLimit(ctx context.Context, target target, request proxy.Request) error {
	source, ok := s.router.(interface {
		TargetMetadata(context.Context, string, string, string) (catalog.Metadata, error)
	})
	if !ok {
		return nil
	}
	metadata, err := source.TargetMetadata(ctx, request.ProviderID, target.accountID, request.Model)
	if err != nil {
		return proxy.Failure("model_catalog_unavailable", 503)
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(request.Body, &fields) != nil {
		return proxy.Failure("invalid_request", 400)
	}
	if metadata.OutputTokenLimit != nil {
		for _, name := range []string{"max_tokens", "max_completion_tokens"} {
			if raw, exists := fields[name]; exists {
				var limit int64
				if json.Unmarshal(raw, &limit) != nil || limit < 1 || limit > *metadata.OutputTokenLimit {
					return proxy.Failure("invalid_request", 400)
				}
			}
		}
	}
	return nil
}
