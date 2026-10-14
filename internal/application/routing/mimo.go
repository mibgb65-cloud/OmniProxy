package routing

import "omniproxy/internal/provider/mimo"

// 未同步时使用明确标记的官方预设；有账号快照后按该账号发现结果限制模型。
func mimoTargetSupports(data snapshot, providerID, accountID, model string) bool {
	if !mimo.SupportsModel(model) {
		return false
	}
	for _, account := range data.accounts {
		if account.ProviderID != providerID || account.AuthMethod != "api_key" || accountID != "" && account.ID != accountID {
			continue
		}
		synced, found := false, false
		for _, item := range data.models {
			if item.providerID == providerID && item.accountID == account.ID {
				synced = true
				found = found || item.model == model
			}
		}
		if !synced || found {
			return true
		}
	}
	return false
}
