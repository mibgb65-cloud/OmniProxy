// Package minimaxquota 将订阅资源查询绑定当前账号和配置版本。
package minimaxquota

import (
	"context"
	"errors"
	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/provider/minimax"
)

// Accounts 不返回凭据，查询和消费均绑定选中账号快照。
type Accounts interface {
	Metadata(context.Context, string) (account.Item, error)
	UseAPIKeyAt(context.Context, string, string, func([]byte) error) error
}

// Providers 仅用于确认账号实际属于 MiniMax。
type Providers interface {
	List(context.Context) ([]provider.Configuration, error)
}

// Client 允许使用官方端点的契约传输验证资源查询。
type Client interface {
	ReadQuota(context.Context, minimax.Options, []byte) (minimax.Quota, error)
}

// Service 不缓存明文凭证或跨账号共享资源响应。
type Service struct {
	accounts  Accounts
	providers Providers
	client    Client
}

// New 使用现有凭证回调和标准库客户端。
func New(accounts Accounts, providers Providers, client Client) *Service {
	if client == nil {
		client = minimax.New(nil)
	}
	return &Service{accounts, providers, client}
}

// Read 查询完成后再次检查账号版本，切区／换 Key 时不发布旧资源。
func (s *Service) Read(ctx context.Context, id string) (minimax.Quota, error) {
	failure := func(code string) (minimax.Quota, error) { return minimax.Quota{}, errors.New("minimax: " + code) }
	a, err := s.accounts.Metadata(ctx, id)
	if err != nil || a.AuthMethod != account.AuthAPIKey || a.MiniMax == nil || !a.MiniMax.Valid() || a.MiniMax.BillingMode != "subscription" {
		return failure("invalid_configuration")
	}
	providers, err := s.providers.List(ctx)
	if err != nil {
		return failure("configuration_unavailable")
	}
	allowed := false
	for _, p := range providers {
		if p.ID == a.ProviderID && p.Kind == minimax.Kind {
			allowed = true
			break
		}
	}
	if !allowed {
		return failure("invalid_configuration")
	}
	var quota minimax.Quota
	err = s.accounts.UseAPIKeyAt(ctx, id, a.UpdatedAt, func(key []byte) error {
		var err error
		quota, err = s.client.ReadQuota(ctx, *a.MiniMax, key)
		return err
	})
	if err != nil {
		switch err.Error() {
		case "minimax: unauthorized", "minimax: forbidden", "minimax: rate_limited", "minimax: network_failed", "minimax: invalid_response", "minimax: query_failed":
			return minimax.Quota{}, err
		}
		return failure("credential_unavailable")
	}
	current, err := s.accounts.Metadata(ctx, id)
	if err != nil || current.UpdatedAt != a.UpdatedAt {
		return failure("configuration_unavailable")
	}
	return quota, nil
}
