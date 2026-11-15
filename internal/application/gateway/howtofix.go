package gateway

import (
	"context"

	"omniproxy/internal/core/agentprotocol"
	"omniproxy/internal/core/proxy"
)

// forwardHowToFix 只把 Chat Completions 原生透传给上游；Responses／Messages 先转换成
// Chat 再转发，并按调用方原始协议编码回包。handbook 虽称 Anthropic／Gemini 路由为原生直出，
// 但在真实 Key 逐项验证前不把转换当成原生协议。
func (s *Service) forwardHowToFix(ctx context.Context, target target, request proxy.Request, consume func(proxy.Response) error) error {
	if request.Protocol == "" || request.Protocol == "chat" {
		return s.useAPIKey(ctx, target, func(key []byte) error {
			return s.howtofix.Forward(ctx, target.baseURL, key, request, func(response proxy.Response) error {
				return consume(proxy.ObserveResponse(ctx, request.Trace, response, "chat"))
			})
		})
	}
	body, kinds, err := agentprotocol.ToChat(request.Body, request.Protocol)
	if err != nil {
		return protocolContentFailure(err)
	}
	upstream := request
	upstream.Protocol = "chat"
	upstream.Body = body
	upstream.Stream = false
	return s.useAPIKey(ctx, target, func(key []byte) error {
		return s.howtofix.Forward(ctx, target.baseURL, key, upstream, func(response proxy.Response) error {
			if response.StatusCode != 200 {
				return consume(response)
			}
			result, err := agentprotocol.ReadChat(response.Body, kinds)
			if err != nil {
				return proxy.Failure("invalid_upstream_response", 502)
			}
			reader, kind := agentprotocol.Encode(result, request.Protocol, request.Stream)
			return consume(proxy.Response{StatusCode: 200, ContentType: kind, Body: reader})
		})
	})
}
