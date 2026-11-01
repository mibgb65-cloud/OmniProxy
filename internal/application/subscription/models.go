package subscription

import (
	"context"
	"strings"

	"omniproxy/internal/application/account"
	"omniproxy/internal/core/netproxy"
	"omniproxy/internal/infrastructure/codex"
)

// Models 只使用本应用管理的授权目录，保持用户官方登录不变。
func (s *Service) Models(ctx context.Context, id string) ([]codex.Model, error) {
	models, item, err := s.listModels(ctx, id)
	if err != nil {
		return nil, err
	}
	version, err := s.codexVersion(ctx)
	if err != nil || !strings.HasPrefix(version, "codex-cli ") {
		return nil, failure("models_unavailable")
	}
	var windows map[string]int64
	// 先释放发现锁，再走统一凭据入口；仍校验同一账号版本，避免重入锁或混用重新登录的身份。
	err = s.UseTokenAt(ctx, id, item.UpdatedAt, func(key []byte, officialID string) error {
		var readErr error
		windows, readErr = s.modelWindows(netproxy.WithProvider(ctx, item.ProviderID, "openai"), key, officialID, strings.TrimPrefix(version, "codex-cli "))
		return readErr
	})
	if err != nil {
		return nil, failure("models_unavailable")
	}
	for index := range models {
		window, found := windows[models[index].ID]
		if !found || window < 1 || window > 10000000 {
			// 不发布容量未知的订阅目录；同步失败保留数据库内上一次完整快照。
			return nil, failure("models_unavailable")
		}
		models[index].Metadata.ContextWindow = &window
	}
	return models, nil
}

func (s *Service) listModels(ctx context.Context, id string) ([]codex.Model, account.Item, error) {
	if err := s.operations.LockAccount(ctx, id); err != nil {
		return nil, account.Item{}, failure("operation_cancelled")
	}
	defer s.operations.UnlockAccount(id)
	item, profile, err := s.accounts.SubscriptionProfile(ctx, id)
	if err != nil || item.AuthState != "authenticated" || s.accountBusy(id) {
		return nil, account.Item{}, failure("invalid_account")
	}
	client, err := s.open(ctx, profile)
	if err != nil {
		return nil, account.Item{}, err
	}
	defer client.Close()
	identity, err := client.AccountID(ctx)
	if err != nil || item.AuthAccountID != "" && item.AuthAccountID != identity {
		return nil, account.Item{}, failure("identity_changed")
	}
	if err := verifyMember(ctx, client, item); err != nil {
		return nil, account.Item{}, err
	}
	catalog, ok := client.(interface {
		Models(context.Context) ([]codex.Model, error)
	})
	if !ok {
		return nil, account.Item{}, failure("models_unavailable")
	}
	models, err := catalog.Models(ctx)
	return models, item, err
}
