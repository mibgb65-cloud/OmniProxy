package agentconfig

import (
	"context"

	catalog "omniproxy/internal/core/modelcatalog"
)

// clientEntries 仅排除已经确认不适合当前客户端的模型，未知能力保持未知。
func (s *Service) clientEntries(ctx context.Context, client string) ([]catalog.Entry, error) {
	entries, err := s.entries(ctx)
	if err != nil {
		return nil, err
	}
	protocol := "messages"
	if client == "codex" {
		protocol = "responses"
	}
	entries = catalog.ForProtocol(entries, protocol)
	if protocol == "messages" {
		return catalog.ClaudeEntries(entries)
	}
	return entries, nil
}
