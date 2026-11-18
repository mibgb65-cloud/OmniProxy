package gateway

import (
	"context"
	"strings"

	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/proxy"
	core "omniproxy/internal/core/routing"
)

// familyEffortOrder 是家族名回退时的档位优先级：目录默认档位来自上游命名，
// 客户端没指定档位时先试 high，再退到 medium／low。
var familyEffortOrder = []string{"high", "medium", "low", "minimal", "xhigh", "ultra"}

// applyFamilyVariant 把客户端按家族发送的模型名解析到目录里的档位变体。
//
// Antigravity 桌面端会用自带的模型注册表把 “Gemini 4 Argon (High)” 折叠成家族名
// gemini-4-argon 再加档位发送；目录里只有上游真实存在的 …-high／…-medium 变体。
// 这里只在精确匹配失败后触发，按客户端档位挑对应变体，没有档位时取 high。
func (s *Service) applyFamilyVariant(ctx context.Context, request proxy.Request) (proxy.Request, bool, error) {
	if s.router == nil || request.Model == "" {
		return request, false, nil
	}
	entries, err := s.Catalog(ctx)
	if err != nil {
		return request, false, nil
	}
	matches := familyVariants(entries, request.Model)
	if len(matches) == 0 {
		return request, false, nil
	}
	variant := pickFamilyVariant(matches, proxy.ReasoningEffort(request.Protocol, request.Body))
	if variant == "" {
		return request, false, nil
	}
	body, err := core.RewriteModel(request.Body, variant)
	if err != nil {
		return request, false, proxy.Failure("invalid_request", 400)
	}
	request.Body, request.Model = body, variant
	return request, true, nil
}

// applyRequestedFamilyVariant 在选服务商之前把「家族名 + 明确档位」换成上游真实变体。
//
// applyFamilyVariant 只在精确匹配失败后触发；一旦有服务商发布了同名裸型号（中转站），
// 路由就会停在裸型号上，自家订阅的档位变体永远选不中。桌面端按自带注册表发送家族名
// 并带上档位，因此这里在路由前先解析。没有明确档位时不猜（保持精确匹配），同族确认
// 不足两个档位时不改写（后缀可能只是型号的一部分）。
func (s *Service) applyRequestedFamilyVariant(ctx context.Context, request proxy.Request) (proxy.Request, error) {
	if s.router == nil || request.Model == "" {
		return request, nil
	}
	effort := proxy.ReasoningEffort(request.Protocol, request.Body)
	if effort == "" {
		return request, nil
	}
	// 已经是档位变体的型号不必再读目录：桌面端绝大多数请求都是这种形态。
	if _, variantShaped := catalog.EffortVariantSuffix(request.Model); variantShaped {
		return request, nil
	}
	entries, err := s.Catalog(ctx)
	if err != nil {
		// 目录不可用时不改写请求；后面的路由解析仍会给出明确失败码。
		return request, nil
	}
	if siblingEffortLevels(entries, request.Model) < 2 {
		return request, nil
	}
	variant := exactFamilyVariant(entries, request.Model, effort)
	if variant == "" {
		return request, nil
	}
	body, err := core.RewriteModel(request.Body, variant)
	if err != nil {
		return request, proxy.Failure("invalid_request", 400)
	}
	request.Body, request.Model = body, variant
	return request, nil
}

// exactFamilyVariant 只认客户端明确要的档位，不做就近替换：路由前的改写必须可解释，
// 不能把 high 请求悄悄降成 medium。
func exactFamilyVariant(entries []catalog.Entry, family, effort string) string {
	suffix := "-" + effort
	for _, entry := range familyVariants(entries, family) {
		if strings.HasSuffix(entry.ID, suffix) {
			return entry.ID
		}
	}
	return ""
}

// familyVariants 收集 ID 以「家族名-」开头的目录条目；只认同一个提供商，
// 避免把别家的同名前缀当成变体。
func familyVariants(entries []catalog.Entry, family string) []catalog.Entry {
	prefix := family + "-"
	result := make([]catalog.Entry, 0, 4)
	kind := ""
	for _, entry := range entries {
		if !strings.HasPrefix(entry.ID, prefix) || entry.ID == family {
			continue
		}
		if kind == "" {
			kind = entry.ProviderKind
		}
		if entry.ProviderKind != kind {
			continue
		}
		result = append(result, entry)
	}
	return result
}

// pickFamilyVariant 优先用客户端档位，其次按固定档位顺序，最后接受唯一候选。
func pickFamilyVariant(matches []catalog.Entry, effort string) string {
	if effort != "" {
		suffix := "-" + effort
		for _, entry := range matches {
			if strings.HasSuffix(entry.ID, suffix) {
				return entry.ID
			}
		}
	}
	for _, level := range familyEffortOrder {
		suffix := "-" + level
		for _, entry := range matches {
			if strings.HasSuffix(entry.ID, suffix) {
				return entry.ID
			}
		}
	}
	if len(matches) == 1 {
		return matches[0].ID
	}
	return ""
}
