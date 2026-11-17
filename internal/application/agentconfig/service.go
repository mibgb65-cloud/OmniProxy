package agentconfig

import (
	"context"
	"github.com/pelletier/go-toml/v2"
)

func (s *Service) Inspect(ctx context.Context) ([]Item, error) {
	items := []Item{}
	for _, client := range Clients() {
		path, _ := s.path(client)
		i := Item{Client: client, Path: path}
		body, exists, err := s.readClient(client, path)
		if err != nil {
			i.ErrorCode = "unsafe_config"
		} else {
			i.Exists = exists
			i.Revision = digest(body, exists)
			var config map[string]any
			if client == "codex" {
				i.Provider = "openai"
				if len(body) > 0 && toml.Unmarshal(body, &config) != nil {
					i.ErrorCode = "invalid_config"
				} else {
					if v, ok := config["model_provider"].(string); ok {
						i.Provider = v
					}
				}
			}
			var enabled bool
			var ref string
			if s.db.QueryRowContext(ctx, `SELECT credential_ref,enabled FROM agent_connections WHERE client=?`, client).Scan(&ref, &enabled) == nil {
				i.Configured = enabled
			}
			var hash string
			if s.db.QueryRowContext(ctx, `SELECT applied_hash FROM agent_backups WHERE client=?`, client).Scan(&hash) == nil {
				i.CanRestore = true
				// Antigravity 会持续改写自己的设置；整文件变化不等于受管字段漂移，
				// 断开时按字段撤销，因此不在这里阻断恢复。
				if hash != i.Revision && client != "antigravity" {
					i.ErrorCode = "config_changed"
				}
				if client == "codex" && s.checkCatalogFile(ctx) != nil {
					i.ErrorCode = "config_changed"
				}
			}
			// 后台刷新失败的原因（写失败、配置被占用、目录为空…）在这里交给界面；
			// 已经有更具体的 config_changed 时保留它（行动方式相同）。
			if code := s.issue(client); code != "" && i.ErrorCode == "" {
				i.ErrorCode = code
			}
		}
		clear(body)
		items = append(items, i)
	}
	return items, nil
}

// Preview 没有写入客户端文件，展示的配置不包含任何原有密码或密钥。
func (s *Service) Preview(ctx context.Context, input Input) (string, error) {
	plan, err := s.PreviewPlan(ctx, input)
	return plan.Summary, err
}

type backup struct {
	Body   []byte `json:"body"`
	Exists bool   `json:"exists"`
}
