package agentconfig

import (
	"context"
	"database/sql"
	"encoding/json"

	catalog "omniproxy/internal/core/modelcatalog"
)

// Claude 目录可能为每个源模型生成一个额外的百万上下文选项。
const maxSelectedModels = 2 * catalog.MaxModels

func filterSelectedModels(entries []catalog.Entry, ids []string, strict bool) ([]catalog.Entry, error) {
	if ids == nil {
		return entries, nil
	}
	if len(ids) == 0 || len(ids) > maxSelectedModels {
		return nil, failure("invalid_model_selection")
	}
	selected := make(map[string]bool, len(ids))
	for _, id := range ids {
		if len(id) == 0 || len(id) > 128 || selected[id] {
			return nil, failure("invalid_model_selection")
		}
		selected[id] = true
	}
	result := make([]catalog.Entry, 0, len(ids))
	for _, entry := range entries {
		if selected[entry.ID] {
			result = append(result, entry)
		}
	}
	if strict && len(result) != len(ids) {
		return nil, failure("invalid_model_selection")
	}
	if len(result) == 0 {
		return nil, failure("no_models")
	}
	return result, nil
}

func (s *Service) selectedCatalog(ctx context.Context, client string, entries []catalog.Entry) ([]catalog.Entry, error) {
	var stored sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT model_ids FROM agent_backups WHERE client=?`, client).Scan(&stored); err != nil {
		return nil, failure("backup_unavailable")
	}
	if !stored.Valid {
		return entries, nil
	}
	var ids []string
	if len(stored.String) > maxSelectedModels*132 || json.Unmarshal([]byte(stored.String), &ids) != nil || ids == nil {
		return nil, failure("invalid_model_selection")
	}
	// 临时不可用的选择继续保留；恢复可用后重新加入，绝不回退到用户未选的模型。
	return filterSelectedModels(entries, ids, false)
}
