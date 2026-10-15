package wails

import (
	"context"
	"errors"
	"omniproxy/internal/application/minimaxquota"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/provider/minimax"
)

// MiniMaxQuota 只返回公开订阅资源，真实 Key 始终保留在后端受控回调。
func (s *AppService) MiniMaxQuota(id string) (minimax.Quota, error) {
	accounts, err := s.accountService()
	if err != nil {
		return minimax.Quota{}, err
	}
	return minimaxquota.New(accounts, provider.NewService(s.db), nil).Read(context.Background(), id)
}

// OpenMiniMaxConsole 从已存地区解析固定官方页面，不接受任意 URL。
func (s *AppService) OpenMiniMaxConsole(id string) error {
	accounts, err := s.accountService()
	if err != nil {
		return err
	}
	a, err := accounts.Metadata(context.Background(), id)
	if err != nil || a.MiniMax == nil || !a.MiniMax.Valid() {
		return errors.New("minimax: invalid_configuration")
	}
	providers, err := provider.NewService(s.db).List(context.Background())
	if err != nil {
		return errors.New("minimax: configuration_unavailable")
	}
	allowed := false
	for _, p := range providers {
		if p.ID == a.ProviderID && p.Kind == minimax.Kind {
			allowed = true
			break
		}
	}
	if !allowed {
		return errors.New("minimax: invalid_configuration")
	}
	address := "https://platform.minimax.cn"
	if a.MiniMax.Region == "global" {
		address = "https://platform.minimax.io"
	}
	if s.openAgentURL == nil {
		return errors.New("minimax: browser_unavailable")
	}
	if s.openAgentURL(address) != nil {
		return errors.New("minimax: browser_unavailable")
	}
	return nil
}
