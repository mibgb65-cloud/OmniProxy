package gateway

import (
	"context"
	"errors"

	"omniproxy/internal/application/account"
	"omniproxy/internal/core/agentprotocol"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/provider/dumate"
)

// forwardDuMate 用 Cookie 登录态换取 gateway token 后转发；上游只确认 Chat，
// 其余协议经既有转换链路接入并在本进程缓冲完整结果。
func (s *Service) forwardDuMate(ctx context.Context, target target, request proxy.Request, consume func(proxy.Response) error) error {
	if s.accounts == nil || s.dumateClient == nil || s.dumateTokens == nil || s.dumateConsole == nil {
		return proxy.Failure("adapter_unavailable", 503)
	}
	return s.accounts.UseDuMateCredentials(ctx, target.accountID, func(item account.Item, credentials dumate.Credentials) error {
		return s.forwardWithDuMateToken(ctx, request, consume, item, credentials)
	})
}

func (s *Service) forwardWithDuMateToken(ctx context.Context, request proxy.Request, consume func(proxy.Response) error, item account.Item, credentials dumate.Credentials) error {
	cacheKey := dumate.TokenCacheKey(item.ID, item.UpdatedAt)
	token, err := s.dumateTokens.Get(ctx, cacheKey, func(fetchCtx context.Context) (dumate.Token, error) {
		return s.dumateConsole.GatewayToken(fetchCtx, credentials)
	})
	if err != nil {
		return dumateTokenFailure(err)
	}
	forwardErr := s.forwardDuMateChat(ctx, request, consume, token.Value)
	// gateway token 被上游拒绝时只清除 token 缓存；Cookie 是否失效由控制台判定。
	if reauthFailure(forwardErr) {
		s.dumateTokens.Invalidate(cacheKey)
	}
	return forwardErr
}

func (s *Service) forwardDuMateChat(ctx context.Context, request proxy.Request, consume func(proxy.Response) error, token string) error {
	if request.Protocol == "" || request.Protocol == "chat" {
		return s.dumateClient.Forward(ctx, token, request, func(response proxy.Response) error {
			return consume(proxy.ObserveResponse(ctx, request.Trace, response, "chat"))
		})
	}
	body, kinds, err := agentprotocol.ToChat(request.Body, request.Protocol)
	if err != nil {
		return protocolContentFailure(err)
	}
	upstream := request
	upstream.Body, upstream.Protocol, upstream.Stream = body, "chat", false
	return s.dumateClient.Forward(ctx, token, upstream, func(response proxy.Response) error {
		if response.StatusCode != 200 {
			return consume(response)
		}
		response = proxy.ObserveResponse(ctx, request.Trace, response, "chat")
		result, err := agentprotocol.ReadChat(response.Body, kinds)
		if err != nil {
			return proxy.Failure("invalid_upstream_response", 502)
		}
		reader, kind := agentprotocol.Encode(result, request.Protocol, request.Stream)
		return consume(proxy.Response{StatusCode: 200, ContentType: kind, Body: reader})
	})
}

// dumateTokenFailure 把控制台换取失败映射为稳定的网关错误码。
func dumateTokenFailure(err error) error {
	if dumate.SignInExpired(err) {
		return proxy.Failure("dumate_sign_in_expired", 401)
	}
	switch dumate.ErrorCode(err) {
	case dumate.CodeRateLimited:
		return proxy.Failure("dumate_rate_limited", 429)
	case dumate.CodeCredentialUnavailable:
		return proxy.Failure("dumate_reauth_required", 401)
	}
	return proxy.Failure("dumate_token_unavailable", 502)
}

// reauthFailure 报告 gateway token 是否被上游拒绝。
func reauthFailure(err error) bool {
	var failure *proxy.Error
	return errors.As(err, &failure) && failure.Code == dumate.CodeReauthRequired
}
