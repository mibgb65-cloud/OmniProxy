package agentconfig

import (
	"context"
	"encoding/json"

	"github.com/pelletier/go-toml/v2"
	"omniproxy/internal/core/agentendpoint"
)

// cleanCodexTransport 仅用于已核对 applied_hash 的受管刷新。
// 原备份有用户值时不能推断归属；不删除它，也不清理外部修改过的文件。
func (s *Service) cleanCodexTransport(ctx context.Context, body []byte) ([]byte, error) {
	var current map[string]any
	if toml.Unmarshal(body, &current) != nil {
		return nil, failure("invalid_config")
	}
	provider, _ := current["model_provider"].(string)
	if provider == "" {
		provider = "openai"
	}
	root, ok := agentendpoint.BaseURL("codex", s.port)
	if !ok {
		return nil, failure("invalid_config")
	}
	base, _ := current["openai_base_url"].(string)
	if provider != "openai" {
		providers, _ := current["model_providers"].(map[string]any)
		entry, _ := providers[provider].(map[string]any)
		base, _ = entry["base_url"].(string)
	}
	if base != root+"/v1" {
		return append([]byte(nil), body...), nil
	}
	var ref string
	if err := s.db.QueryRowContext(ctx, `SELECT credential_ref FROM agent_backups WHERE client='codex'`).Scan(&ref); err != nil {
		return nil, failure("backup_unavailable")
	}
	raw, err := s.store.Get(ref)
	if err != nil {
		return nil, failure("backup_unavailable")
	}
	defer clear(raw)
	var original backup
	if json.Unmarshal(raw, &original) != nil {
		return nil, failure("backup_unavailable")
	}
	defer clear(original.Body)
	var before map[string]any
	if len(original.Body) > 0 && toml.Unmarshal(original.Body, &before) != nil {
		return nil, failure("backup_unavailable")
	}
	updates := map[string]string{}
	features, _ := current["features"].(map[string]any)
	originalFeatures, _ := before["features"].(map[string]any)
	for _, key := range []string{"responses_websockets", "responses_websockets_v2"} {
		if value, present := features[key]; present && value == false {
			if original, userOwned := originalFeatures[key]; !userOwned {
				updates["features."+key] = ""
			} else if original == true {
				// 旧版本会把用户原来的 true 覆盖成 false；受管版本吻合时恢复原意。
				updates["features."+key] = "true"
			}
		}
	}
	if provider != "openai" {
		updates["model_providers."+provider+".supports_websockets"] = "true"
	}
	if len(updates) == 0 {
		return append([]byte(nil), body...), nil
	}
	return patchTOML(body, provider, updates)
}
