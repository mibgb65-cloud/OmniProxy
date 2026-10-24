package wails

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/agentconfig"
	"omniproxy/internal/application/provider"
)

// maxLocalCodexAuthBytes 限制从本机读取的登录文件，避免把任意大文件读进内存。
const maxLocalCodexAuthBytes = 128 << 10

// codexLoginSource 返回要写进客户端登录的池账号 auth.json（v1 的池账号语义）。
// 没有可用池账号时返回 nil：此时不接管客户端原有登录。
func (s *AppService) codexLoginSource() func(context.Context) ([]byte, error) {
	return func(ctx context.Context) ([]byte, error) {
		service, err := s.accountService()
		if err != nil {
			return nil, err
		}
		items, err := service.List(ctx)
		if err != nil {
			return nil, err
		}
		enabled, err := s.enabledCodexProviders(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if !item.Enabled || !enabled[item.ProviderID] || item.AuthMethod != account.AuthCodexFile || item.AuthState != "authenticated" {
				continue
			}
			var body []byte
			if err := service.UseCodexAuth(ctx, item.ID, func(raw []byte) error {
				body = append([]byte(nil), raw...)
				return nil
			}); err != nil {
				continue
			}
			return body, nil
		}
		return nil, nil
	}
}

func (s *AppService) enabledCodexProviders(ctx context.Context) (map[string]bool, error) {
	items, err := provider.NewService(s.db).List(ctx)
	if err != nil {
		return nil, err
	}
	enabled := make(map[string]bool, len(items))
	for _, item := range items {
		if item.Enabled && item.Kind == "openai" {
			enabled[item.ID] = true
		}
	}
	return enabled, nil
}

// ImportCodexAuthFromClient 把本机 Codex 登录收进网关自己的池（v1 的导入语义）。
// 重复导入同一官方账号只更新凭据，不新增账号。
func (s *AppService) ImportCodexAuthFromClient() (account.Item, error) {
	service, err := s.accountService()
	if err != nil {
		return account.Item{}, err
	}
	ctx := context.Background()
	enabled, err := s.enabledCodexProviders(ctx)
	if err != nil {
		return account.Item{}, err
	}
	providerID := ""
	for id := range enabled {
		if providerID == "" || id < providerID {
			providerID = id
		}
	}
	if providerID == "" {
		return account.Item{}, errors.New("account: no enabled OpenAI provider")
	}
	raw, err := readLocalCodexAuth()
	if err != nil {
		return account.Item{}, err
	}
	defer clear(raw)
	return service.ImportCodexAuth(ctx, providerID, "", raw)
}

// readLocalCodexAuth 只读取用户自己的 Codex 登录文件，不触碰隔离 profile。
func readLocalCodexAuth() ([]byte, error) {
	paths, err := agentconfig.DefaultPaths()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(filepath.Dir(paths.Codex), "auth.json")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxLocalCodexAuthBytes {
		return nil, errors.New("account: local Codex auth unavailable")
	}
	return os.ReadFile(path)
}
