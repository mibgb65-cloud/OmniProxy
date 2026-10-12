// Package balance 查询账号的公开余额，不保存或返回 API Key。
package balance

import (
	"context"
	"errors"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/provider/deepseek"
)

type AccountAccess interface {
	Metadata(context.Context, string) (account.Item, error)
	UseAPIKey(context.Context, string, func([]byte) error) error
}

type ProviderReader interface {
	List(context.Context) ([]provider.Configuration, error)
}
type Client interface {
	Balance(context.Context, []byte) (deepseek.Balance, error)
}

type Info struct {
	Available bool                   `json:"available"`
	Entries   []deepseek.BalanceInfo `json:"entries"`
	FetchedAt string                 `json:"fetchedAt"`
}

var officialClient = deepseek.New(nil)

type Service struct {
	accounts  AccountAccess
	providers ProviderReader
	client    Client
}

func New(accounts AccountAccess, providers ProviderReader, client Client) *Service {
	if client == nil {
		client = officialClient
	}
	return &Service{accounts: accounts, providers: providers, client: client}
}

func failure(code string) error { return errors.New("balance: " + code) }

// Read 复用受控凭据回调，停用账号也可做只读查询，不影响转发资格。
func (s *Service) Read(ctx context.Context, id string) (Info, error) {
	readCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	item, err := s.accounts.Metadata(readCtx, id)
	if err != nil {
		return Info{}, failure("invalid_account")
	}
	if item.AuthMethod != account.AuthAPIKey {
		return Info{}, failure("unsupported")
	}
	providers, err := s.providers.List(readCtx)
	if err != nil {
		return Info{}, failure("configuration_unavailable")
	}
	allowed := false
	for _, value := range providers {
		if value.ID == item.ProviderID && value.Kind == "deepseek" {
			allowed = true
			break
		}
	}
	if !allowed {
		return Info{}, failure("unsupported")
	}
	var result deepseek.Balance
	called := false
	err = s.accounts.UseAPIKey(readCtx, id, func(key []byte) error {
		called = true
		var queryErr error
		result, queryErr = s.client.Balance(readCtx, key)
		return queryErr
	})
	if err != nil {
		if !called {
			return Info{}, failure("credential_unavailable")
		}
		switch err.Error() {
		case "balance: unauthorized", "balance: forbidden", "balance: rate_limited", "balance: service_unavailable", "balance: network_failed", "balance: invalid_response", "balance: credential_unavailable":
			return Info{}, err
		}
		return Info{}, failure("query_failed")
	}
	return Info{Available: result.Available, Entries: result.Entries, FetchedAt: time.Now().UTC().Format(time.RFC3339)}, nil
}
