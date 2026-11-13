package gateway

import (
	"context"

	"omniproxy/internal/application/account"
	"omniproxy/internal/core/agentprotocol"
	"omniproxy/internal/core/netproxy"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/provider/dumate"
	"omniproxy/internal/provider/oxionai"
)

type SubscriptionAccess interface {
	UseToken(context.Context, string, func([]byte, string) error) error
}

func (s *Service) SetSubscription(service SubscriptionAccess) { s.subscription = service }

// forwardAgent 订阅 Responses 原样流式转发，跨协议保留文本及工具调用关联。
func (s *Service) forwardAgent(ctx context.Context, target target, request proxy.Request, consume func(proxy.Response) error) error {
	// 把提供商身份写进 ctx：出站代理按它决定直连还是走代理。
	ctx = netproxy.WithProvider(ctx, target.snapshot.ProviderID, target.kind)
	if request.Compaction {
		return s.forwardCompaction(ctx, target, request, consume)
	}
	if target.kind == "minimax" {
		return s.forwardMiniMax(ctx, target, request, consume)
	}
	if target.kind == "kimi" {
		return s.forwardKimi(ctx, target, request, consume)
	}
	if target.kind == "stepfun" {
		return s.forwardStep(ctx, target, request, consume)
	}
	if target.kind == "openai" && target.authMethod != account.AuthAPIKey && s.subscription != nil {
		return s.useToken(ctx, target, func(key []byte, accountID string) error {
			upstream := request
			if request.Protocol != "responses" {
				chat, _, err := agentprotocol.ToChat(request.Body, request.Protocol)
				if err != nil {
					return protocolContentFailure(err)
				}
				body, err := agentprotocol.ChatToResponses(chat)
				if err != nil {
					return proxy.Failure("unsupported_content", 400)
				}
				upstream.Body = body
			}
			return s.chatgpt.Forward(ctx, key, accountID, upstream, func(response proxy.Response) error {
				response = proxy.ObserveResponse(ctx, request.Trace, response, "responses")
				if response.PreExecutionQuotaRejected || response.StatusCode != 200 || request.Protocol == "responses" && request.Stream {
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
	if target.kind == "openai" && target.authMethod == account.AuthAPIKey {
		return s.forwardOpenAIAPIKey(ctx, target, request, consume)
	}
	if target.kind == "openai" {
		return proxy.Failure("adapter_unavailable", 503)
	}
	if target.kind == "river" {
		return s.forwardRiver(ctx, target, request, consume)
	}
	if target.kind == oxionai.Kind {
		return s.forwardOxionai(ctx, target, request, consume)
	}
	if target.kind == "atria" {
		return s.useAPIKey(ctx, target, func(key []byte) error {
			return s.atria.Forward(ctx, target.baseURL, key, request, func(response proxy.Response) error {
				return consume(proxy.ObserveResponse(ctx, request.Trace, response, request.Protocol))
			})
		})
	}
	if target.kind == "google" {
		return s.forwardGoogle(ctx, target, request, consume)
	}
	if target.kind == "mimo" {
		return s.forwardMiMo(ctx, target, request, consume)
	}
	if target.kind == dumate.Kind {
		return s.forwardDuMate(ctx, target, request, consume)
	}
	if target.kind != "" && target.kind != "deepseek" {
		return proxy.Failure("adapter_unavailable", 503)
	}
	if request.Protocol == "" || request.Protocol == "chat" {
		return s.useAPIKey(ctx, target, func(key []byte) error { return s.forwardDeepseek(ctx, target, key, request, consume) })
	}
	body, kinds, err := agentprotocol.ToChat(request.Body, request.Protocol)
	if err != nil {
		return protocolContentFailure(err)
	}
	upstream := request
	upstream.Body = body
	upstream.Stream = false
	return s.useAPIKey(ctx, target, func(key []byte) error {
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

// forwardOpenAIAPIKey 把 Responses 原样送到 api.openai.com，其他协议先转成
// Responses 再按客户端协议回包；不使用订阅的 ChatGPT 端点与账号头。
func (s *Service) forwardOpenAIAPIKey(ctx context.Context, target target, request proxy.Request, consume func(proxy.Response) error) error {
	if s.openAI == nil {
		return proxy.Failure("adapter_unavailable", 503)
	}
	upstream := request
	if request.Protocol != "responses" {
		chat, _, err := agentprotocol.ToChat(request.Body, request.Protocol)
		if err != nil {
			return protocolContentFailure(err)
		}
		body, err := agentprotocol.ChatToResponses(chat)
		if err != nil {
			return proxy.Failure("unsupported_content", 400)
		}
		upstream.Body, upstream.Protocol, upstream.Stream = body, "responses", true
	}
	return s.useAPIKey(ctx, target, func(key []byte) error {
		return s.openAI.Forward(ctx, target.baseURL, key, upstream, func(response proxy.Response) error {
			response = proxy.ObserveResponse(ctx, request.Trace, response, "responses")
			if response.PreExecutionQuotaRejected || response.StatusCode != 200 || request.Protocol == "responses" {
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
