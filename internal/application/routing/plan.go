package routing

import (
	"context"
	"omniproxy/internal/application/account"
	core "omniproxy/internal/core/routing"
	"sort"
	"strings"
	"time"
)

// Plan 保留配置解析结果与同池候选，不受短期冷却状态影响。
type Plan struct {
	Decision   Preview
	Candidates []Preview
	Fixed      bool
}

// ResolvePlan 复用 evaluate，避免备用账号绕过原有能力、计费与地区边界。
func (s *Service) ResolvePlan(ctx context.Context, model string) (Plan, error) {
	if !core.ValidModel(model) {
		return Plan{}, ErrInvalidInput
	}
	rules, err := s.rules(ctx)
	if err != nil {
		return Plan{}, err
	}
	data, err := s.snapshot(ctx)
	if err != nil {
		return Plan{}, err
	}
	rule, matched := matchRoute(model, rules, data)
	decision := evaluate(model, rule, matched, data)
	plan := Plan{Decision: decision, Candidates: []Preview{}}
	if !decision.Routable {
		return plan, nil
	}
	manual, explicit := core.Match(model, rules)
	plan.Fixed = explicit && manual.AccountID != ""
	plan.Candidates = append(plan.Candidates, decision)
	if plan.Fixed {
		return plan, nil
	}
	logical := model
	if matched {
		logical = rule.Model
	}
	var selected account.Item
	for _, a := range data.accounts {
		if a.ID == decision.AccountID {
			selected = a
			break
		}
	}
	ordered := append([]account.Item(nil), data.accounts...)
	// evaluate 已校验创建时间；这里只对同池备用账号沿用稳定排序。
	sort.SliceStable(ordered, func(i, j int) bool {
		left, _ := time.Parse(time.RFC3339Nano, ordered[i].CreatedAt)
		right, _ := time.Parse(time.RFC3339Nano, ordered[j].CreatedAt)
		if !left.Equal(right) {
			return left.Before(right)
		}
		return ordered[i].ID < ordered[j].ID
	})
	for _, a := range ordered {
		if a.ID == selected.ID || !SameAccountPool(selected, a) {
			continue
		}
		if selected.AuthMethod == account.AuthSubscription && a.AuthAccountID == selected.AuthAccountID && strings.EqualFold(strings.TrimSpace(a.AuthEmail), strings.TrimSpace(selected.AuthEmail)) {
			continue
		}
		if !a.Enabled || !eligible(data, a, logical) && !uncataloguedSubscriptionBackup(data, a, logical) {
			continue
		}
		// 服务商和通道已冻结；复用资格校验，不为每个备用账号重复解析整张路由表。
		candidate := decision
		candidate.AccountID, candidate.AccountName, candidate.AccountUpdatedAt = a.ID, a.Name, a.UpdatedAt
		plan.Candidates = append(plan.Candidates, candidate)
	}
	return plan, nil
}

// SameAccountPool 只比较计费与地区身份，不把配置版本或同名提供商当作同一账号池。
func SameAccountPool(a, b account.Item) bool {
	if a.ID == "" || b.ID == "" || a.ProviderID != b.ProviderID || a.AuthMethod != b.AuthMethod {
		return false
	}
	if (a.MiMo == nil) != (b.MiMo == nil) || (a.StepFun == nil) != (b.StepFun == nil) || (a.Kimi == nil) != (b.Kimi == nil) || (a.MiniMax == nil) != (b.MiniMax == nil) {
		return false
	}
	if a.MiMo != nil && !a.MiMo.SamePool(*b.MiMo) {
		return false
	}
	if a.StepFun != nil && !a.StepFun.SamePool(*b.StepFun) {
		return false
	}
	if a.Kimi != nil && !a.Kimi.SamePool(*b.Kimi) {
		return false
	}
	return a.MiniMax == nil || a.MiniMax.SamePool(*b.MiniMax)
}

// 新登录订阅尚未同步目录时，仅作为已知模型的备用尝试；有明确目录却不支持的账号仍排除。
// 不扩大模型发现或主路由，实际模型权限仍由备用账号的官方响应确认。
func uncataloguedSubscriptionBackup(data snapshot, a account.Item, model string) bool {
	if a.AuthMethod != account.AuthSubscription || a.AuthState != "authenticated" {
		return false
	}
	openai := false
	for _, p := range data.providers {
		if p.ID == a.ProviderID && p.Enabled && p.Kind == "openai" {
			openai = true
		}
	}
	if !openai || !targetSupports(data, a.ProviderID, "", model) {
		return false
	}
	for _, m := range data.models {
		if m.accountID == a.ID {
			return false
		}
	}
	return true
}
