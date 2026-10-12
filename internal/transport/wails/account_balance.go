package wails

import (
	"context"

	"omniproxy/internal/application/balance"
	"omniproxy/internal/application/provider"
)

// AccountBalance 仅返回公开余额，凭据在后端回调内使用和清理。
func (s *AppService) AccountBalance(id string) (balance.Info, error) {
	accounts, err := s.accountService()
	if err != nil {
		return balance.Info{}, err
	}
	return balance.New(accounts, provider.NewService(s.db), nil).Read(context.Background(), id)
}
