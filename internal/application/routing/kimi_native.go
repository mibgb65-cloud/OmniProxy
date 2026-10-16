package routing

import (
	core "omniproxy/internal/core/routing"
	"slices"
)

// 官方订阅原生 ID 也可直接请求，但不能因普通 API 优先而误用按量账号。
func matchKimiNative(model string, data snapshot) (core.Rule, bool) {
	canonical := ""
	switch model {
	case "k3":
		canonical = "kimi-k3"
	case "kimi-for-coding-highspeed":
		canonical = "kimi-k2.7-code-highspeed"
	default:
		return core.Rule{}, false
	}
	data.accounts = slices.Clone(data.accounts)
	for i, a := range data.accounts {
		if a.Kimi == nil || a.Kimi.Mode() != "coding" {
			data.accounts[i].Enabled = false
		}
	}
	decision := evaluate(canonical, core.Rule{}, false, data)
	if !decision.Routable {
		return core.Rule{}, false
	}
	return core.Rule{Alias: model, Model: canonical, ProviderID: decision.ProviderID, AccountID: decision.AccountID, Enabled: true}, true
}
