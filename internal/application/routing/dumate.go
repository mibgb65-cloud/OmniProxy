package routing

import (
	"omniproxy/internal/application/account"
	"omniproxy/internal/provider/dumate"
)

// dumateTargetSupports 只承认固定别名与已登录的 DuMate 账号；
// 上游没有目录接口，账号快照也就没有模型行。
func dumateTargetSupports(data snapshot, providerID, accountID, model string) bool {
	if !dumate.SupportsModel(model) {
		return false
	}
	for _, item := range data.accounts {
		if item.ProviderID != providerID || item.AuthMethod != account.AuthDuMateCookie || item.AuthState != "authenticated" {
			continue
		}
		if accountID == "" || item.ID == accountID {
			return true
		}
	}
	return false
}
