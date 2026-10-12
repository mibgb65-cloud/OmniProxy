package wails

import (
	"context"
	"fmt"

	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/credential"
)

func (s *AppService) accountService() (*account.Service, error) {
	store, err := credential.New(s.prof)
	if err != nil {
		return nil, fmt.Errorf("account: initialize credential store: %w", err)
	}
	return account.NewService(s.db, store), nil
}

// ListAccounts returns metadata for all saved accounts.
func (s *AppService) ListAccounts() ([]account.Item, error) {
	service, err := s.accountService()
	if err != nil {
		return nil, err
	}
	return service.List(context.Background())
}

// CreateAccount saves a new API Key account.
func (s *AppService) CreateAccount(input account.CreateInput) (account.Item, error) {
	service, err := s.accountService()
	if err != nil {
		return account.Item{}, err
	}
	item, err := service.Create(context.Background(), input)
	if err == nil {
		s.refreshAgentCatalog()
	}
	return item, err
}

// CreateAccounts 一次保存多个自动命名的 API Key 账号。
func (s *AppService) CreateAccounts(input account.BatchCreateInput) ([]account.Item, error) {
	service, err := s.accountService()
	if err != nil {
		return nil, err
	}
	items, err := service.CreateBatch(context.Background(), input)
	if err == nil {
		s.refreshAgentCatalog()
	}
	return items, err
}

// UpdateAccount 更新名称及可选的新密钥，不返回完整密钥。
func (s *AppService) UpdateAccount(input account.UpdateInput) (account.UpdateResult, error) {
	service, err := s.accountService()
	if err != nil {
		return account.UpdateResult{}, err
	}
	result, err := service.Update(context.Background(), input)
	if err == nil {
		s.refreshAgentCatalog()
	}
	return result, err
}

// SetAccountEnabled changes whether an account can later be scheduled.
func (s *AppService) SetAccountEnabled(id string, enabled bool) (account.Item, error) {
	service, err := s.accountService()
	if err != nil {
		return account.Item{}, err
	}
	item, err := service.SetEnabled(context.Background(), id, enabled)
	if err == nil {
		s.refreshAgentCatalog()
	}
	return item, err
}

// DeleteAccount removes account metadata and its protected API Key.
func (s *AppService) DeleteAccount(id string) error {
	service, err := s.accountService()
	if err != nil {
		return err
	}
	item, err := service.Metadata(context.Background(), id)
	if err != nil {
		return err
	}
	if item.AuthMethod == account.AuthSubscription {
		manager, err := s.subscriptionService()
		if err != nil {
			return err
		}
		ctx, cancel := subscriptionContext()
		defer cancel()
		err = manager.Delete(ctx, id)
		if err == nil {
			s.refreshAgentCatalog()
		}
		return err
	}
	err = service.Delete(context.Background(), id)
	if err == nil {
		s.refreshAgentCatalog()
	}
	return err
}
