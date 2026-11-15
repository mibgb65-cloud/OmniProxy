package antigravity

import (
	"context"
	"sort"
	"strings"
	"time"

	"omniproxy/internal/core/modelcatalog"
)

const tierRefusedFor = 24 * time.Hour

// isFreeTier 判断套餐标识是否为免费/遗留档：Google 拒绝在上报这些值时列出模型。
func isFreeTier(tier string) bool {
	return tier == "free-tier" || tier == "legacy-tier"
}

// entitlementTiers 是 fetchAvailableModels 依次尝试的权益：付费档位优先，
// 24 小时内被拒过的不再尝试，最后是不带权益的请求。
func (c *Client) entitlementTiers(cred Credentials) []string {
	tiers := make([]string, 0, 2)
	tier := strings.TrimSpace(cred.Tier)
	if tier != "" && !isFreeTier(tier) && !c.tierRefused(tierKey(cred, tier)) {
		tiers = append(tiers, tier)
	}
	return append(tiers, "")
}

func tierKey(cred Credentials, tier string) string {
	return cred.Subject + "\x00" + cred.Project + "\x00" + tier
}

func (c *Client) tierRefused(key string) bool {
	c.tiers.Lock()
	defer c.tiers.Unlock()
	until, ok := c.tiers.refused[key]
	if !ok {
		return false
	}
	if !until.After(time.Now()) {
		delete(c.tiers.refused, key)
		return false
	}
	return true
}

// refuseTier 记住一次被拒绝的权益，24 小时内不再用它列模型。
func (c *Client) refuseTier(key string) {
	c.tiers.Lock()
	defer c.tiers.Unlock()
	if c.tiers.refused == nil {
		c.tiers.refused = map[string]time.Time{}
	}
	if len(c.tiers.refused) >= refusalLimit {
		for candidate, until := range c.tiers.refused {
			if !until.After(time.Now()) {
				delete(c.tiers.refused, candidate)
			}
		}
	}
	c.tiers.refused[key] = time.Now().Add(tierRefusedFor)
}

// availableModels 是 fetchAvailableModels 的应答；只保留目录需要的字段。
type availableModels struct {
	Models map[string]struct {
		Name     string `json:"displayName"`
		Internal bool   `json:"isInternal"`
		Disabled bool   `json:"disabled"`
	} `json:"models"`
	Sorts []struct {
		Groups []struct {
			IDs []string `json:"modelIds"`
		} `json:"groups"`
	} `json:"agentModelSorts"`
}

// Models 不使用静态套餐表，也不把未知能力当作已经验证：
// 先按账号套餐尝试权益，被 400 拒绝时降级，最后不带权益。
func (c *Client) Models(ctx context.Context, cred Credentials) ([]Model, error) {
	var value availableModels
	var failure error
	for _, tier := range c.entitlementTiers(cred) {
		payload := map[string]any{"project": cred.Project}
		if tier != "" {
			payload["entitlement"] = map[string]any{"userTier": tier}
		}
		value = availableModels{}
		if err := c.post(ctx, inferenceURL+"/v1internal:fetchAvailableModels", cred.AccessToken, payload, &value); err != nil {
			if tier != "" && ErrorCode(err) == "invalid_request" {
				c.refuseTier(tierKey(cred, tier))
			}
			failure = err
			continue
		}
		failure = nil
		break
	}
	if failure != nil {
		return nil, failure
	}
	if value.Models == nil || len(value.Models) > 1000 {
		return nil, Failure("invalid_response")
	}
	entries := []modelcatalog.Entry{}
	selected := map[string]bool{}
	if len(value.Sorts) > 32 {
		return nil, Failure("invalid_response")
	}
	for _, order := range value.Sorts {
		if len(order.Groups) > 64 {
			return nil, Failure("invalid_response")
		}
		for _, group := range order.Groups {
			if len(group.IDs) > 1000 {
				return nil, Failure("invalid_response")
			}
			for _, id := range group.IDs {
				selected[id] = true
			}
		}
	}
	for id, item := range value.Models {
		// 停用／内部模型不进入目录：请求停用模型会被上游以 404 拒绝。
		if item.Internal || item.Disabled || strings.HasPrefix(id, "chat_") || strings.HasPrefix(id, "tab_") {
			continue
		}
		// Gemini 历史别名保留兼容；新增第三方型号必须由当前官方选择器明确列出。
		if !supportedModel(id) || (!strings.HasPrefix(id, "gemini-") && !selected[id]) {
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
		tools := true
		// Gemini 通道已确认能接收内联图片（用户消息与工具结果都转成 inlineData），
		// 客户端据此才允许附图、才把 agent 读到的图片当像素发送。Claude／GPT-OSS
		// 通道没有经过同样的图片验证，继续按纯文本发布。
		modalities := []string{"text"}
		if strings.HasPrefix(id, "gemini-") {
			modalities = []string{"text", "image"}
		}
		entry.Metadata = modelcatalog.Metadata{InputModalities: modalities, ToolUse: &tools}
		entries = append(entries, entry)
	}
	// 上游用 “(High)／(Low)／(Medium)” 等独立型号表达推理档位；折叠回基础模型，
	// 档位改由客户端原生 effort 选择，网关请求时换成实际存在的变体 ID。
	folded := modelcatalog.FoldEffortVariants(entries)
	models := make([]Model, 0, len(folded))
	for _, entry := range folded {
		models = append(models, Model{ID: entry.ID, Name: entry.Name, Metadata: entry.Metadata})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	return models, nil
}

func supportedModel(id string) bool {
	switch id {
	case "claude-opus-4-6-thinking", "claude-sonnet-4-6", "claude-sonnet-4-6-thinking", "gpt-oss-120b-medium":
		return true
	}
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
