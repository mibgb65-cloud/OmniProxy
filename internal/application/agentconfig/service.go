package agentconfig

import (
	"context"
	"github.com/pelletier/go-toml/v2"
)

func (s *Service) Inspect(ctx context.Context) ([]Item, error) {
	items := []Item{}
	for _, client := range []string{"codex", "claude", "claude-desktop"} {
		path, _ := s.path(client)
		i := Item{Client: client, Path: path}
		body, exists, err := read(path)
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
				if hash != i.Revision {
					i.ErrorCode = "config_changed"
				}
				if client == "codex" && s.checkCatalogFile(ctx) != nil {
					i.ErrorCode = "config_changed"
				}
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
