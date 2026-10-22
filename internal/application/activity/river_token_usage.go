package activity

import (
	"context"

	"omniproxy/internal/application/account"
)

// RiverTokenUsage 读取当前 River API Key 账号的本机留存用量，不访问凭据或官方接口。
func (s *Service) RiverTokenUsage(ctx context.Context, item account.Item) (*AccountTokenUsage, error) {
	if item.ID == "" || item.ProviderID != "builtin_river" || item.AuthMethod != "api_key" {
		return nil, ErrInvalid
	}
	return s.tokenUsage(ctx, item, "river")
}
