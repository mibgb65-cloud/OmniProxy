package gateway

import (
	"context"
	"encoding/json"
	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/core/proxy"
)

func snapshotTarget(p provider.Configuration, a account.Item) proxy.TargetSnapshot {
	target := proxy.TargetSnapshot{ProviderID: p.ID, ProviderName: p.Name, ProviderKind: p.Kind, AccountID: a.ID, AccountName: a.Name, AuthMethod: a.AuthMethod, Email: a.AuthEmail, WorkspaceID: a.AuthAccountID}
	if a.StepFun != nil {
		target.ProviderRegion = a.StepFun.Region
		target.BillingMode = a.StepFun.BillingMode
	}
	if a.MiniMax != nil {
		target.ProviderRegion = a.MiniMax.Region
		target.BillingMode = a.MiniMax.BillingMode
	}
	if a.Kimi != nil {
		target.ProviderRegion = a.Kimi.Region
	}
	return target
}
func (s *Service) forwardDeepseek(ctx context.Context, target target, key []byte, request proxy.Request, consume func(proxy.Response) error) error {
	if request.Stream {
		var fields map[string]json.RawMessage
		if json.Unmarshal(request.Body, &fields) != nil {
			return proxy.Failure("invalid_request", 400)
		}
		options := map[string]json.RawMessage{}
		if raw, exists := fields["stream_options"]; exists && string(raw) != "null" {
			if json.Unmarshal(raw, &options) != nil || options == nil {
				return proxy.Failure("invalid_request", 400)
			}
		}
		options["include_usage"] = json.RawMessage("true")
		fields["stream_options"], _ = json.Marshal(options)
		body, err := json.Marshal(fields)
		if err != nil || len(body) > proxy.MaxRequestBytes {
			return proxy.Failure("request_too_large", 413)
		}
		request.Body = body
	}
	return s.client.Forward(ctx, target.baseURL, key, request, func(response proxy.Response) error {
		return consume(proxy.ObserveResponse(ctx, request.Trace, response, "chat"))
	})
}
