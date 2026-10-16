package googlesubscription

import (
	"context"
	"time"

	"omniproxy/internal/provider/antigravity"
)

// ReadQuota 读取独立 Google 额度；在途重新授权／退出后不得发布旧结果。
func (s *Service) ReadQuota(ctx context.Context, id string) (antigravity.Quota, error) {
	return readGoogleResource(s, ctx, id, s.client.ReadQuota)
}

// ReadAccountModels 独立读取可选模型，失败不会覆盖已成功的额度快照。
func (s *Service) ReadAccountModels(ctx context.Context, id string) ([]antigravity.AccountModel, error) {
	return readGoogleResource(s, ctx, id, s.client.AccountModels)
}

func readGoogleResource[T any](s *Service, ctx context.Context, id string, read func(context.Context, antigravity.Credentials) (T, error)) (T, error) {
	var empty T
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	s.ops.Lock()
	cred, err := s.credentials(ctx, id, false)
	version := ""
	if err == nil {
		version, err = s.accounts.CatalogVersion(ctx, id)
	}
	s.ops.Unlock()
	if err != nil {
		return empty, err
	}
	value, err := read(ctx, cred)
	if err != nil {
		return empty, err
	}
	if ctx.Err() != nil {
		return empty, antigravity.Failure("upstream_unavailable")
	}
	s.ops.Lock()
	defer s.ops.Unlock()
	current, currentError := s.accounts.Metadata(ctx, id)
	latest, versionError := s.accounts.CatalogVersion(ctx, id)
	if currentError != nil || versionError != nil || version != latest || current.AuthState != "authenticated" || current.AuthAccountID != cred.Subject || s.busy(id) {
		return empty, antigravity.Failure("quota_changed")
	}
	return value, nil
}
