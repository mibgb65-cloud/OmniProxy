package antigravity

import (
	"context"
	"sort"
	"strings"

	"omniproxy/internal/core/modelcatalog"
)

// AccountModel 来自官方当前选择列表，适配状态不等于已验证所有调用权限。
type AccountModel struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	GatewaySupported bool   `json:"gatewaySupported"`
}

// AccountModels 仅发布官方选择器引用的模型，不把内部补全模型或历史别名当作可选型号。
func (c *Client) AccountModels(ctx context.Context, cred Credentials) ([]AccountModel, error) {
	var raw struct {
		Models map[string]struct {
			Name     string `json:"displayName"`
			Internal bool   `json:"isInternal"`
		} `json:"models"`
		Sorts *[]struct {
			Groups []struct {
				IDs []string `json:"modelIds"`
			} `json:"groups"`
		} `json:"agentModelSorts"`
	}
	if !cred.Valid() {
		return nil, Failure("reauth_required")
	}
	if err := c.post(ctx, inferenceURL+"/v1internal:fetchAvailableModels", cred.AccessToken, map[string]string{"project": cred.Project}, &raw); err != nil {
		return nil, err
	}
	if raw.Models == nil || len(raw.Models) > 1000 || raw.Sorts == nil || len(*raw.Sorts) > 32 {
		return nil, Failure("invalid_response")
	}
	models := []AccountModel{}
	seen := map[string]bool{}
	for _, order := range *raw.Sorts {
		if len(order.Groups) > 64 {
			return nil, Failure("invalid_response")
		}
		for _, group := range order.Groups {
			if len(group.IDs) > 1000 {
				return nil, Failure("invalid_response")
			}
			for _, id := range group.IDs {
				if seen[id] {
					continue
				}
				seen[id] = true
				entry, exists := raw.Models[id]
				if !exists {
					return nil, Failure("invalid_response")
				}
				if entry.Internal {
					continue
				}
				name := strings.TrimSpace(entry.Name)
				if name == "" {
					name = id
				}
				if _, err := modelcatalog.Normalize([]modelcatalog.Entry{{ID: id, Name: name}}); err != nil {
					return nil, Failure("invalid_response")
				}
				models = append(models, AccountModel{ID: id, Name: name, GatewaySupported: supportedModel(id)})
			}
		}
	}
	sort.Slice(models, func(i, j int) bool {
		if models[i].GatewaySupported != models[j].GatewaySupported {
			return models[i].GatewaySupported
		}
		return models[i].ID < models[j].ID
	})
	return models, nil
}
