package antigravity

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"omniproxy/internal/core/modelcatalog"
)

// Model 仅包含上游实际返回且本适配器支持的生成模型。
type Model struct {
	ID       string
	Name     string
	Metadata modelcatalog.Metadata
}

type projectResponse struct {
	Project     json.RawMessage `json:"cloudaicompanionProject"`
	CurrentTier struct {
		ID string `json:"id"`
	} `json:"currentTier"`
	PaidTier struct {
		ID string `json:"id"`
	} `json:"paidTier"`
	AllowedTiers []struct {
		ID      string `json:"id"`
		Default bool   `json:"isDefault"`
	} `json:"allowedTiers"`
}

func projectID(raw json.RawMessage) string {
	var id string
	if json.Unmarshal(raw, &id) == nil {
		return id
	}
	var value struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(raw, &value) == nil {
		return value.ID
	}
	return ""
}

// Project 查询官方项目与套餐；未初始化时只选择官方标记的默认档位。
// Tier 保存付费档位（其次当前档位），作为模型目录的权益参数。
func (c *Client) Project(ctx context.Context, cred *Credentials) error {
	var value projectResponse
	metadata := map[string]string{"ideType": "ANTIGRAVITY"}
	if err := c.post(ctx, controlURL+"/v1internal:loadCodeAssist", cred.AccessToken, map[string]any{"metadata": metadata}, &value); err != nil {
		return err
	}
	tier := strings.TrimSpace(value.PaidTier.ID)
	if tier == "" {
		tier = strings.TrimSpace(value.CurrentTier.ID)
	}
	if tier != "" && !safeString(tier, 80) {
		return Failure("invalid_response")
	}
	cred.Project, cred.Plan, cred.Tier = projectID(value.Project), tier, tier
	if safeString(cred.Project, 256) {
		return nil
	}
	defaultTier := ""
	for _, candidate := range value.AllowedTiers {
		if candidate.Default && safeString(candidate.ID, 80) {
			defaultTier = candidate.ID
			break
		}
	}
	if defaultTier == "" {
		return Failure("project_unavailable")
	}
	for range 5 {
		var result struct {
			Done     bool            `json:"done"`
			Response projectResponse `json:"response"`
		}
		payload := map[string]any{
			"tier_id":  defaultTier,
			"metadata": map[string]string{"ide_type": "ANTIGRAVITY", "ide_version": protocolVersion(), "ide_name": "antigravity"},
		}
		// 与官方客户端一致：初始化请求带 Node 客户端标识。
		headers := map[string]string{
			"X-Goog-Api-Client": "gl-node/22.21.1",
			"User-Agent":        userAgent() + " google-api-nodejs-client/10.3.0",
		}
		if err := c.postWith(ctx, inferenceURL+"/v1internal:onboardUser", cred.AccessToken, payload, &result, headers); err != nil {
			return err
		}
		if result.Done {
			cred.Project = projectID(result.Response.Project)
			if safeString(cred.Project, 256) {
				return nil
			}
			return Failure("project_unavailable")
		}
		select {
		case <-ctx.Done():
			return Failure("upstream_unavailable")
		case <-time.After(time.Second):
		}
	}
	return Failure("project_unavailable")
}
