package modelcatalog

import (
	"context"
	"database/sql"
	"encoding/json"

	core "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/infrastructure/codex"
	"time"
)

// ConfirmContext 为已真实验证的订阅模型保存 1M；必须绑定验证时的账号授权版本。
// 这些官方模型的总窗口为 1050000；约 900K 输入验证留出输出空间，不把百万输入失败误判为模型不支持长窗口。
func (s *Service) ConfirmContext(ctx context.Context, accountID, model, expectedVersion string, measuredInput int64) error {
	known := map[string]bool{"gpt-6.1-sol": true, "gpt-6-sol": true, "gpt-6-luna": true, "gpt-6-astra": true, "gpt-5.6-sol": true, "gpt-5.6-terra": true, "gpt-5.6-luna": true}
	if len(accountID) == 0 || len(accountID) > 128 || !known[model] || expectedVersion == "" || len(expectedVersion) > 512 || measuredInput < 900000 || measuredInput > 1050000 {
		return ErrUnavailable
	}
	var raw, version string
	var found sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(m.metadata_json,'{}'),a.credential_ref || ':' || a.auth_account_id,m.model FROM accounts a
 LEFT JOIN account_models m ON m.account_id=a.id AND m.model=? AND (a.credential_ref || ':' || a.auth_account_id)=m.account_version
 JOIN providers p ON p.id=a.provider_id
 WHERE a.id=? AND a.enabled=1 AND a.auth_method='subscription_login'
 AND a.auth_state='authenticated' AND p.enabled=1 AND p.kind='openai'`, model, accountID).Scan(&raw, &version, &found)
	var metadata core.Metadata
	if err != nil || version != expectedVersion || len(raw) > 16384 || json.Unmarshal([]byte(raw), &metadata) != nil || core.ValidateMetadata(metadata) != nil {
		return ErrUnavailable
	}
	// 本机旧官方核心尚未列出 6.1，但相同账号直接调用已完整完成；补充明确验证的新模型，不能复制用户缓存。
	if !found.Valid {
		if model != "gpt-6.1-sol" {
			return ErrUnavailable
		}
		metadata.DefaultEffort = "medium"
		for _, effort := range []string{"low", "medium", "high", "xhigh", "max"} {
			metadata.ReasoningEfforts = append(metadata.ReasoningEfforts, core.Effort{Value: effort})
		}
		metadata.InputModalities = []string{"text", "image"}
	}
	window := core.ClaudeLongContextWindow
	metadata.ContextWindow, metadata.ContextVerified = &window, true
	body, err := json.Marshal(metadata)
	if err != nil {
		return ErrUnavailable
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO account_models(account_id,model,name,fetched_at,account_version,metadata_json)
 SELECT ?,?,?,?,?,? WHERE EXISTS(SELECT 1 FROM accounts a JOIN providers p ON p.id=a.provider_id
 WHERE a.id=? AND (a.credential_ref || ':' || a.auth_account_id)=? AND a.enabled=1
 AND a.auth_method='subscription_login' AND a.auth_state='authenticated' AND p.enabled=1 AND p.kind='openai')
 ON CONFLICT(account_id,model) DO UPDATE SET metadata_json=excluded.metadata_json,account_version=excluded.account_version`, accountID, model, "GPT-6.1-Sol", time.Now().UTC().Format(time.RFC3339Nano), expectedVersion, string(body), accountID, expectedVersion)
	if err != nil {
		return ErrUnavailable
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return ErrUnavailable
	}
	return nil
}

func confirmedModels(ctx context.Context, tx *sql.Tx, accountID, version string) (map[string]codex.Model, error) {
	rows, err := tx.QueryContext(ctx, `SELECT model,name,metadata_json FROM account_models WHERE account_id=? AND account_version=?`, accountID, version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]codex.Model{}
	for rows.Next() {
		var model codex.Model
		var raw string
		if rows.Scan(&model.ID, &model.Name, &raw) != nil || len(raw) > 16384 || json.Unmarshal([]byte(raw), &model.Metadata) != nil || core.ValidateMetadata(model.Metadata) != nil {
			return nil, ErrUnavailable
		}
		if model.Metadata.ContextVerified {
			result[model.ID] = model
		}
	}
	return result, rows.Err()
}
