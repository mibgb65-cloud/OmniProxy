package agentconfig

import (
	"context"

	"omniproxy/internal/application/modeleffort"
	catalog "omniproxy/internal/core/modelcatalog"
)

// effortClient 表示使用客户端原生 effort 控件的受管客户端。
func effortClient(client string) bool {
	return client == "claude" || client == "claude-desktop"
}

// validateEfforts 只接受所选模型中已确认的档位。
// 空档位表示清除该模型的固定值；未确认档位的模型直接拒绝，不猜测、不降级。
func validateEfforts(entries []catalog.Entry, efforts map[string]string) error {
	for model, effort := range efforts {
		if model == "" || len(model) > 128 || len(effort) > 16 {
			return failure("invalid_effort")
		}
		found := false
		for _, entry := range entries {
			if entry.ID != model {
				continue
			}
			found = true
			if effort != "" && !catalog.ClaudeEffort(entry, effort) {
				return failure("invalid_effort")
			}
			break
		}
		if !found {
			return failure("invalid_model")
		}
	}
	return nil
}

// effortPlan 合并已保存的固定档位与本次选择，供界面显示当前的生效档位。
// 本次显式选择（含空值）优先；已保存但当前模型不再确认的档位不显示。
func (s *Service) effortPlan(ctx context.Context, entries []catalog.Entry, chosen map[string]string) (map[string]string, error) {
	saved, err := modeleffort.Pins(ctx, s.db)
	if err != nil {
		return nil, failure("effort_unavailable")
	}
	plan := make(map[string]string, len(entries))
	for _, entry := range entries {
		if effort, ok := chosen[entry.ID]; ok {
			if effort != "" {
				plan[entry.ID] = effort
			}
			continue
		}
		if effort := saved[entry.ID]; effort != "" && catalog.ClaudeEffort(entry, effort) {
			plan[entry.ID] = effort
		}
	}
	return plan, nil
}
