package workbuddyai

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"

	"omniproxy/internal/core/proxy"
)

const (
	// maxCatalogBytes 限制 /v3/config 正文，避免异常上游拖垮解析。
	maxCatalogBytes = 1 << 20
	// maxCatalogModels 与 core/modelcatalog 的上限保持一致。
	maxCatalogModels = 1000
	// maxModelIDBytes 与 core/routing 的模型名上限一致。
	maxModelIDBytes = 128
	// maxModelNameBytes 限制展示名长度。
	maxModelNameBytes = 200
	// maxModelEfforts 限制单模型的推理档位数量。
	maxModelEfforts = 16
)

// Model 是账号自己的 /v3/config 里可路由的一条模型记录。
type Model struct {
	ID   string
	Name string
	// Context 与 Output 是上游声明的输入/输出上限；0 表示未声明。
	Context int
	Output  int
	// Images 是上游声明的图片输入能力；glm-* 如实为 false。
	Images bool
	// Efforts 已去掉 "none"；DefaultEffort 不在 Efforts 里时为空。
	Efforts       []string
	DefaultEffort string
	// Free 需要价格促销信息；本适配器不读 modelPromotions，因此恒为 false（未知）。
	Free bool
}

// rawModel 是 /v3/config 里 data.models[] 的一条记录。
type rawModel struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	MaxInputTokens    json.RawMessage `json:"maxInputTokens"`
	MaxOutputTokens   json.RawMessage `json:"maxOutputTokens"`
	SupportsImages    *bool           `json:"supportsImages"`
	SupportsReasoning *bool           `json:"supportsReasoning"`
	Reasoning         *rawReasoning   `json:"reasoning"`
}

type rawReasoning struct {
	SupportedEfforts   []string `json:"supportedEfforts"`
	DefaultEffort      string   `json:"defaultEffort"`
	Effort             string   `json:"effort"`
	CanDisableThinking *bool    `json:"canDisableThinking"`
}

// DiscoverModels 有界读取账号自己的模型目录。目录按套餐与企业发放，因此不能
// 用静态清单替代：空目录、超限或非 200 都失败，调用方保留旧快照。
func (c *Client) DiscoverModels(ctx context.Context, cred Credentials) (models []Model, result error) {
	if !cred.Valid() {
		return nil, Failure(CodeCredentialUnavailable)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+configPath, nil)
	if err != nil {
		return nil, proxy.UpstreamFailure(err)
	}
	c.applySign(request, cred)
	defer request.Header.Del("Authorization")
	response, err := c.http.Do(request)
	if err != nil {
		return nil, proxy.UpstreamFailure(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil && result == nil {
			result = Failure(CodeUpstreamUnavailable)
		}
	}()
	body, err := readBounded(response.Body, maxCatalogBytes)
	if err != nil {
		return nil, err
	}
	if failure := businessStatus(response.StatusCode, body); failure != nil {
		return nil, failure
	}
	data, err := decodeBusiness(body)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Models []rawModel `json:"models"`
	}
	if json.Unmarshal(data, &payload) != nil {
		return nil, Failure(CodeInvalidResponse)
	}
	if len(payload.Models) > maxCatalogModels {
		return nil, Failure(CodeInvalidResponse)
	}
	models = make([]Model, 0, len(payload.Models))
	seen := map[string]bool{}
	for _, item := range payload.Models {
		if !modelID(item.ID) || seen[item.ID] || !safeText(item.Name, maxModelNameBytes, true) {
			return nil, Failure(CodeInvalidResponse)
		}
		seen[item.ID] = true
		models = append(models, normalizeModel(item))
	}
	if len(models) == 0 {
		return nil, Failure(CodeInvalidResponse)
	}
	return models, nil
}

// normalizeModel 只做字段映射与边界收敛：缺失字段保持未知，不补默认值。
func normalizeModel(item rawModel) Model {
	model := Model{
		ID:     item.ID,
		Name:   item.Name,
		Images: item.SupportsImages != nil && *item.SupportsImages && !strings.HasPrefix(item.ID, "glm-"),
	}
	if model.Name == "" {
		model.Name = item.ID
	}
	model.Context = tokenLimit(item.MaxInputTokens)
	model.Output = tokenLimit(item.MaxOutputTokens)
	if item.SupportsReasoning != nil && !*item.SupportsReasoning {
		return model
	}
	if item.Reasoning == nil {
		return model
	}
	seen := map[string]bool{}
	for _, effort := range item.Reasoning.SupportedEfforts {
		if effort == "none" || seen[effort] || !effortValue(effort) {
			continue
		}
		seen[effort] = true
		model.Efforts = append(model.Efforts, effort)
		if len(model.Efforts) >= maxModelEfforts {
			break
		}
	}
	for _, candidate := range []string{item.Reasoning.DefaultEffort, item.Reasoning.Effort} {
		if candidate != "" && seen[candidate] {
			model.DefaultEffort = candidate
			break
		}
	}
	return model
}

// tokenLimit 解析正整数的 token 上限；非法、负数或超限按未知（0）处理。
func tokenLimit(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return 0
	}
	var number float64
	switch typed := value.(type) {
	case float64:
		number = typed
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0
		}
		number = parsed
	default:
		return 0
	}
	if math.IsNaN(number) || math.IsInf(number, 0) || number < 1 || number > 10000000 {
		return 0
	}
	return int(number)
}

// modelID 接受上游的模型 ID：非空、≤128、可打印且不含空格（企业自定义模型带
// "custom:" 前缀与 "|" 分隔符，必须原样保留）。
func modelID(id string) bool {
	return safeASCII(id, maxModelIDBytes, false)
}

// effortValue 只接受 modelcatalog 允许的推理档位形态。
func effortValue(effort string) bool {
	if effort == "" || len(effort) > 32 {
		return false
	}
	for _, char := range effort {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '_' || char == '-' {
			continue
		}
		return false
	}
	return true
}
