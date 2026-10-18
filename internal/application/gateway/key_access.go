package gateway

import (
	"context"
	"crypto/sha256"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/provider/antigravity"
)

// useAPIKey 保留已有测试适配口，生产调用将每次尝试绑定到已校验的凭据版本。
func (s *Service) useAPIKey(ctx context.Context, t target, consume func([]byte) error) error {
	guarded := func(key []byte) error {
		if claims, ok := ctx.Value(credentialClaimKey{}).(map[[32]byte]bool); ok {
			digest := sha256.Sum256(key)
			if claims[digest] {
				return proxy.Failure("duplicate_credential", 503)
			}
			claims[digest] = true
		}
		return consume(key)
	}
	if access, ok := s.accounts.(interface {
		UseAPIKeyAt(context.Context, string, string, func([]byte) error) error
	}); ok {
		return access.UseAPIKeyAt(ctx, t.accountID, t.updatedAt, guarded)
	}
	return s.accounts.UseAPIKey(ctx, t.accountID, guarded)
}

// 摘要只存在于单个顺序请求的内存中，不跨请求缓存、不写日志或返回前端。
type credentialClaimKey struct{}

func (s *Service) useToken(ctx context.Context, t target, consume func([]byte, string) error) error {
	if access, ok := s.subscription.(interface {
		UseTokenAt(context.Context, string, string, func([]byte, string) error) error
	}); ok {
		return access.UseTokenAt(ctx, t.accountID, t.updatedAt, consume)
	}
	return s.subscription.UseToken(ctx, t.accountID, consume)
}

func (s *Service) useGoogleToken(ctx context.Context, t target, consume func(antigravity.Credentials) error) error {
	if access, ok := s.googleAuth.(interface {
		UseAt(context.Context, string, string, func(antigravity.Credentials) error) error
	}); ok {
		return access.UseAt(ctx, t.accountID, t.updatedAt, consume)
	}
	return s.googleAuth.Use(ctx, t.accountID, consume)
}
