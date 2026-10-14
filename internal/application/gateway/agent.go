package gateway

import (
	"context"
	"omniproxy/internal/core/agentprotocol"
	"omniproxy/internal/core/proxy"
)

type SubscriptionAccess interface {
	UseToken(context.Context, string, func([]byte, string) error) error
}

func (s *Service) SetSubscription(service SubscriptionAccess) { s.subscription = service }

// forwardAgent 订阅 Responses 原样流式转发，跨协议保留文本及工具调用关联。
func (s *Service) forwardAgent(ctx context.Context, target target, request proxy.Request, consume func(proxy.Response) error) error {
	if target.kind == "openai" && s.subscription != nil {
		return s.subscription.UseToken(ctx, target.accountID, func(key []byte, accountID string) error {
			upstream := request
			if request.Protocol != "responses" {
				chat, _, err := agentprotocol.ToChat(request.Body, request.Protocol)
				if err != nil {
					return proxy.Failure("unsupported_content", 400)
				}
				body, err := agentprotocol.ChatToResponses(chat)
				if err != nil {
					return proxy.Failure("unsupported_content", 400)
				}
				upstream.Body = body
			}
			return s.chatgpt.Forward(ctx, key, accountID, upstream, func(response proxy.Response) error {
				response = proxy.ObserveResponse(ctx, request.Trace, response, "responses")
				if response.StatusCode != 200 || request.Protocol == "responses" && request.Stream {
					return consume(response)
				}
				result, err := agentprotocol.ReadResponses(response.Body)
				if err != nil {
					return proxy.Failure("invalid_upstream_response", 502)
				}
				reader, kind := agentprotocol.Encode(result, request.Protocol, request.Stream)
				return consume(proxy.Response{StatusCode: 200, ContentType: kind, Body: reader})
			})
		})
	}
	if target.kind == "openai" {
		return proxy.Failure("adapter_unavailable", 503)
	}
	if target.kind == "atria" {
		return s.accounts.UseAPIKey(ctx, target.accountID, func(key []byte) error {
			return s.atria.Forward(ctx, target.baseURL, key, request, func(response proxy.Response) error {
				return consume(proxy.ObserveResponse(ctx, request.Trace, response, request.Protocol))
			})
		})
	}
	if target.kind == "google" {
		return s.forwardGoogle(ctx, target, request, consume)
	}
	if target.kind == "mimo" {
		return s.accounts.UseAPIKey(ctx, target.accountID, func(key []byte) error {
			return s.mimo.Forward(ctx, target.baseURL, key, request, func(response proxy.Response) error {
				return consume(proxy.ObserveResponse(ctx, request.Trace, response, request.Protocol))
			})
		})
	}
	if target.kind != "" && target.kind != "deepseek" {
		return proxy.Failure("adapter_unavailable", 503)
	}
	if request.Protocol == "" || request.Protocol == "chat" {
		return s.accounts.UseAPIKey(ctx, target.accountID, func(key []byte) error { return s.forwardDeepseek(ctx, target, key, request, consume) })
	}
	body, kinds, err := agentprotocol.ToChat(request.Body, request.Protocol)
	if err != nil {
		return proxy.Failure("unsupported_content", 400)
	}
	upstream := request
	upstream.Body = body
	upstream.Stream = false
	return s.accounts.UseAPIKey(ctx, target.accountID, func(key []byte) error {
		return s.forwardDeepseek(ctx, target, key, upstream, func(response proxy.Response) error {
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
