package routing

import (
	"context"

	catalog "omniproxy/internal/core/modelcatalog"
)

// TargetMetadata 按实际选中账号读取限制，避免同名模型或别名使用其他账号的元数据。
func (s *Service) TargetMetadata(ctx context.Context, providerID, accountID, model string) (catalog.Metadata, error) {
	models, err := s.discovered(ctx)
	if err != nil {
		return catalog.Metadata{}, err
	}
	for _, item := range models {
		if item.providerID == providerID && item.accountID == accountID && item.model == model {
			return item.metadata, nil
		}
	}
	return catalog.Metadata{}, catalog.ErrInvalid
}
