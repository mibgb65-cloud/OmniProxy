package antigravity

import (
	"context"
	"encoding/json"
	"sort"
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
func (c *Client) Project(ctx context.Context, cred *Credentials) error {
	var value projectResponse
	metadata := map[string]string{"ideType": "ANTIGRAVITY"}
	if err := c.post(ctx, controlURL+"/v1internal:loadCodeAssist", cred.AccessToken, map[string]any{"metadata": metadata}, &value); err != nil {
		return err
	}
	cred.Project, cred.Plan = projectID(value.Project), value.PaidTier.ID
	if cred.Plan == "" {
		cred.Plan = value.CurrentTier.ID
	}
	if cred.Plan != "" && !safeString(cred.Plan, 80) {
		return Failure("invalid_response")
	}
	if safeString(cred.Project, 256) {
		return nil
	}
	tier := ""
	for _, candidate := range value.AllowedTiers {
		if candidate.Default && safeString(candidate.ID, 80) {
			tier = candidate.ID
			break
		}
	}
	if tier == "" {
		return Failure("project_unavailable")
	}
	for range 5 {
		var result struct {
			Done     bool            `json:"done"`
			Response projectResponse `json:"response"`
		}
		if err := c.post(ctx, inferenceURL+"/v1internal:onboardUser", cred.AccessToken, map[string]any{"tier_id": tier, "metadata": map[string]string{"ide_type": "ANTIGRAVITY", "ide_version": "2.23.0", "ide_name": "antigravity"}}, &result); err != nil {
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

// Models 不使用静态套餐表，也不把未知能力当作已经验证。
func (c *Client) Models(ctx context.Context, cred Credentials) ([]Model, error) {
	var value struct {
		Models map[string]struct {
			Name string `json:"displayName"`
		} `json:"models"`
	}
	if err := c.post(ctx, inferenceURL+"/v1internal:fetchAvailableModels", cred.AccessToken, map[string]string{"project": cred.Project}, &value); err != nil {
		return nil, err
	}
	if value.Models == nil || len(value.Models) > 1000 {
		return nil, Failure("invalid_response")
	}
	models := []Model{}
	for id, item := range value.Models {
		if !supportedModel(id) {
			continue
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = id
		}
		entry := modelcatalog.Entry{ID: id, Name: name}
		if _, err := modelcatalog.Normalize([]modelcatalog.Entry{entry}); err != nil {
			return nil, Failure("invalid_response")
		}
		models = append(models, Model{ID: id, Name: name})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	return models, nil
}

func supportedModel(id string) bool {
	if !strings.HasPrefix(id, "gemini-") || len(id) > 128 {
		return false
	}
	for _, part := range []string{"image", "embedding", "tts", "audio", "robotics", "computer-use"} {
		if strings.Contains(id, part) {
			return false
		}
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '.' || r == '_') {
			return false
		}
	}
	return true
}
