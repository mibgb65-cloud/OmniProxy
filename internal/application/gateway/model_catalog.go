package gateway

import (
	"context"

	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/provider/deepseek"
)

// Catalog 只读取可解析的模型配置，网关停止时仍可用于客户端接入。
func (s *Service) Catalog(ctx context.Context) ([]catalog.Entry, error) {
	if source, ok := s.router.(interface {
		Catalog(context.Context) ([]catalog.Entry, error)
	}); ok {
		return source.Catalog(ctx)
	}
	ids, err := s.AvailableModels(ctx)
	if err != nil {
		return nil, err
	}
	entries := make([]catalog.Entry, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, catalog.Entry{ID: id, Name: id, Metadata: deepseek.ModelMetadata(id)})
	}
	return catalog.Normalize(entries)
}
