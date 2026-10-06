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
	return service.Create(context.Background(), input)
}

// CreateAccounts 一次保存多个自动命名的 API Key 账号。
func (s *AppService) CreateAccounts(input account.BatchCreateInput) ([]account.Item, error) {
	service, err := s.accountService()
	if err != nil {
		return nil, err
	}
	return service.CreateBatch(context.Background(), input)
}

// SetAccountEnabled changes whether an account can later be scheduled.
func (s *AppService) SetAccountEnabled(id string, enabled bool) (account.Item, error) {
	service, err := s.accountService()
	if err != nil {
		return account.Item{}, err
	}
	return service.SetEnabled(context.Background(), id, enabled)
}

// DeleteAccount removes account metadata and its protected API Key.
func (s *AppService) DeleteAccount(id string) error {
	service, err := s.accountService()
	if err != nil {
		return err
	}
	return service.Delete(context.Background(), id)
}
