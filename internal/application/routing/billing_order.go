package routing

import (
	"omniproxy/internal/application/account"
	core "omniproxy/internal/core/routing"
)

func preferredBilling(data snapshot, a account.Item) (mode, preferred string) {
	if a.MiMo != nil && a.MiMo.Valid() {
		return a.MiMo.BillingMode, data.mimoSettings.PreferredBillingMode
	}
	if a.StepFun != nil && a.StepFun.Valid() {
		return a.StepFun.BillingMode, data.stepSettings.PreferredBillingMode
	}
	return "", ""
}

// 各服务商先保留有合格账号的首选计费方式，再复用核心的服务商／创建时间顺序。
// 固定账号及旧显式通道不受默认设置覆盖；这里只筛候选，不引入失败重试。
func prioritizeBillingAccounts(data snapshot, candidates []core.Account, allowed map[string]bool, fixed string) {
	if fixed != "" {
		return
	}
	modes := map[string]string{}
	for index, a := range data.accounts {
		mode, preferred := preferredBilling(data, a)
		if !candidates[index].Enabled || !allowed[a.ProviderID] || mode == "" {
			continue
		}
		if modes[a.ProviderID] == "" || mode == preferred {
			modes[a.ProviderID] = mode
		}
	}
	for index, a := range data.accounts {
		mode, _ := preferredBilling(data, a)
		if mode != "" && modes[a.ProviderID] != "" && mode != modes[a.ProviderID] {
			candidates[index].Enabled = false
		}
	}
}
