package gateway

import (
	"context"

	"omniproxy/internal/application/account"
	"omniproxy/internal/core/proxy"
)

// 独立压缩是原生协议能力，不能经过 Chat／Messages 转换伪造或丢弃加密上下文项。
func (s *Service) forwardCompaction(ctx context.Context, target target, request proxy.Request, consume func(proxy.Response) error) error {
	if target.kind != "openai" {
		return proxy.Failure("compaction_not_supported", 501)
	}
	if target.authMethod == account.AuthAPIKey {
		return s.useAPIKey(ctx, target, func(key []byte) error {
			return s.openAI.Forward(ctx, target.baseURL, key, request, func(response proxy.Response) error {
				return consume(proxy.ObserveResponse(ctx, request.Trace, response, "responses"))
			})
		})
	}
	if s.subscription == nil {
		return proxy.Failure("compaction_not_supported", 501)
	}
	return s.useToken(ctx, target, func(key []byte, accountID string) error {
		return s.chatgpt.Compact(ctx, key, accountID, request, func(response proxy.Response) error {
			return consume(proxy.ObserveResponse(ctx, request.Trace, response, "responses"))
		})
	})
}
