package gateway

import (
	"context"

	"omniproxy/internal/application/account"
	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/core/proxy"
)

// defaultCodexUpstreamBase 是 v1 的默认上游基址；可在设置里改成自建反代。
const defaultCodexUpstreamBase = "https://chatgpt.com/backend-api/codex"

// SetCodexUpstreamBase 只在启动前安装设置读取器，每次请求即时生效。
func (s *Service) SetCodexUpstreamBase(read func(context.Context) (string, error)) {
	s.upstreamBase = read
}

// codexUpstreamBase 读取当前上游基址；读取失败不能猜一个地址。
func (s *Service) codexUpstreamBase(ctx context.Context) (string, error) {
	if s.upstreamBase == nil {
		return defaultCodexUpstreamBase, nil
	}
	value, err := s.upstreamBase(ctx)
	if err != nil || value == "" {
		return "", proxy.Failure("configuration_unavailable", 503)
	}
	return value, nil
}

// Transparent 报告这次请求能否按字节透明转发。
//
// 只有 OpenAI 通道与 Codex 共用同一套 Responses 契约（路径、正文与凭据都一致）；
// 其他服务商需要网关把公共模型 ID 换成官方 ID、还原推理档位并做适配器校验，
// 必须回到协议入口处理。无法判定时保持透明转发，不改变既有失败语义。
func (s *Service) Transparent(ctx context.Context, request proxy.PassthroughRequest) bool {
	if s.router == nil || s.accounts == nil || s.providers == nil {
		return true
	}
	model, _ := passthroughModel(request)
	if model == "" {
		return true
	}
	decision, _, _, err := s.resolvePreview(ctx, model)
	if err != nil || !decision.Routable {
		return true
	}
	providers, err := s.providers.List(ctx)
	if err != nil {
		return true
	}
	for _, item := range providers {
		if item.ID == decision.ProviderID {
			return item.Kind == "openai"
		}
	}
	return true
}

// OpenPassthrough 把透明转发请求解析成同池候选账号序列（v1 的取号语义）。
func (s *Service) OpenPassthrough(ctx context.Context, request proxy.PassthroughRequest) (proxy.PassthroughSession, error) {
	if s.accounts == nil {
		return nil, proxy.Failure("invalid_configuration", 503)
	}
	model, effort := passthroughModel(request)
	if model == "" || s.router == nil {
		return s.openFallbackSession(ctx, model, effort, "")
	}
	decision, candidates, fixed, err := s.resolvePreview(ctx, model)
	if err != nil {
		return nil, proxy.Failure("routing_configuration_unavailable", 503)
	}
	if !decision.Routable {
		if decision.ErrorCode == "unsupported_model" {
			return s.openFallbackSession(ctx, model, effort, "")
		}
		return nil, proxy.Failure(routeFailureCode(decision), 503)
	}
	plan := proxy.Request{
		ProviderID: decision.ProviderID, AccountID: decision.AccountID,
		AccountUpdatedAt: decision.AccountUpdatedAt, Model: decision.Model,
		Effort: effort, Candidates: candidates, FixedAccount: fixed, Protocol: "responses",
	}
	initial, err := s.selectRoutedTarget(ctx, plan)
	if err != nil {
		return nil, err
	}
	plan.Candidates, err = s.rotateAccounts(ctx, initial, plan)
	if err != nil {
		return nil, err
	}
	return s.newPassthroughSession(ctx, initial, plan, decision.Model, "", fixed)
}

// openFallbackSession 在正文没有可用模型时（例如 WebSocket 握手没有正文）按 v1 的取号方式
// 取第一个可用订阅账号，但保留完整候选序列：只认第一个账号会让一个借不出令牌的账号
// 直接掐断整条通路，而带模型的 HTTP 请求仍能轮换成功。
func (s *Service) openFallbackSession(ctx context.Context, model, effort, base string) (proxy.PassthroughSession, error) {
	targets, err := s.passthroughFallbackTargets(ctx)
	if err != nil {
		return nil, err
	}
	initial := targets[0]
	candidates := make([]proxy.AccountCandidate, 0, len(targets))
	for _, item := range targets {
		candidates = append(candidates, proxy.AccountCandidate{AccountID: item.accountID, UpdatedAt: item.updatedAt})
	}
	for _, item := range targets[1:] {
		if !sameTargetPool(initial, item) {
			return nil, proxy.Failure("invalid_configuration", 503)
		}
	}
	plan := proxy.Request{
		ProviderID: initial.snapshot.ProviderID, AccountID: initial.accountID,
		AccountUpdatedAt: initial.updatedAt, Model: model, Effort: effort, Protocol: "responses",
		Candidates: candidates,
	}
	plan.Candidates, err = s.rotateFallbackCandidates(ctx, initial, plan)
	if err != nil {
		return nil, err
	}
	return s.newPassthroughSession(ctx, initial, plan, model, base, false)
}

// passthroughFallbackTargets 只选已授权且启用的订阅账号，不猜接入方式。
// 顺序即候选顺序：第一个账号仍排在最前，其余作为可轮换的备选。
func (s *Service) passthroughFallbackTargets(ctx context.Context) ([]target, error) {
	if s.providers == nil || s.accounts == nil {
		return nil, proxy.Failure("invalid_configuration", 503)
	}
	providers, err := s.providers.List(ctx)
	if err != nil {
		return nil, proxy.Failure("configuration_unavailable", 503)
	}
	accounts, err := s.accounts.List(ctx)
	if err != nil {
		return nil, proxy.Failure("configuration_unavailable", 503)
	}
	subscriptions := make([]target, 0, len(accounts))
	apiKeys := make([]target, 0, len(accounts))
	for _, provider := range providers {
		if !provider.Enabled || provider.Kind != "openai" {
			continue
		}
		for _, item := range accounts {
			if item.ProviderID != provider.ID || !item.Enabled {
				continue
			}
			entry := target{
				authMethod: item.AuthMethod, accountID: item.ID, updatedAt: item.UpdatedAt,
				baseURL: provider.BaseURL, kind: provider.Kind, snapshot: snapshotTarget(provider, item),
			}
			switch {
			case codexLoginMethod(item.AuthMethod) && item.AuthState == "authenticated":
				subscriptions = append(subscriptions, entry)
			case item.AuthMethod == account.AuthAPIKey:
				apiKeys = append(apiKeys, entry)
			}
		}
	}
	// 订阅与按量是两个计费池；握手没有模型可解析时优先保留原有订阅行为，
	// 没有订阅账号时才使用 API Key 池，不能把两种凭据混在同一条连接里。
	targets := subscriptions
	if len(targets) == 0 {
		targets = apiKeys
	}
	if len(targets) == 0 {
		return nil, proxy.Failure("route_provider_unavailable", 503)
	}
	return targets, nil
}

// passthroughModel 从正文尽力读取模型与推理档位；读不到交给取号回退，不能因此拒绝请求。
func passthroughModel(request proxy.PassthroughRequest) (string, string) {
	parsed, err := proxy.ParseProtocolRequest(request.Body, "/v1/responses")
	if err != nil {
		return "", ""
	}
	return parsed.Model, parsed.Effort
}

func routeFailureCode(decision routes.Preview) string {
	if decision.ErrorCode == "" {
		return "route_unavailable"
	}
	return "route_" + decision.ErrorCode
}
