package workbuddyai

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"sort"
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

// defaultEfforts 是上游只给 effort、不给档位列表（onlyReasoning）时补的通用档位。
// 依据：社区插件 0.2.36 的 effortsOf() 对这类模型请求 low/high/max，并在真实账号上可用。
var defaultEfforts = []string{"low", "high", "max"}

// effortOrder 是补档位时的排序顺序（与社区插件的 EFFORT_ORDER 一致）。
var effortOrder = []string{"low", "medium", "high", "xhigh", "max"}

// Model 是账号自己的 /v3/config 里可路由的一条模型记录。
type Model struct {
	ID   string
	Name string
	// Context 与 Output 是上游声明的输入/输出上限；0 表示未声明。
	Context int
	Output  int
	// Images 是上游声明的图片输入能力；glm-* 如实为 false。
	Images bool
	// Efforts 已去掉 "none"；上游只给 effort 的模型补 low/high/max 加它自己的档位。
	// DefaultEffort 不在 Efforts 里时为空。
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

// normalizeModel 只做字段映射与边界收敛：缺失字段保持未知，不补默认值。唯一的补全
// 是推理档位：上游声明支持推理却只给 effort 时，按实测口径补上通用档位。
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
	model.Efforts = cleanEfforts(item.Reasoning.SupportedEfforts)
	if len(model.Efforts) == 0 && item.SupportsReasoning != nil && *item.SupportsReasoning {
		// 上游对 max-model、gemini-4-*、kimi-k3 这类模型只给 reasoning.effort，
		// 不给 supportedEfforts；它们在真实账号上接受 low/high/max 加自己的档位。
		model.Efforts = cleanEfforts(append(append([]string{}, defaultEfforts...), item.Reasoning.Effort))
		sortEfforts(model.Efforts)
	}
	seen := make(map[string]bool, len(model.Efforts))
	for _, effort := range model.Efforts {
		seen[effort] = true
	}
	for _, candidate := range []string{item.Reasoning.DefaultEffort, item.Reasoning.Effort} {
		if candidate != "" && seen[candidate] {
			model.DefaultEffort = candidate
			break
		}
	}
	return model
}

// cleanEfforts 去掉 "none"、重复值与非法拼写，只保留上游真正可请求的档位。
func cleanEfforts(raw []string) []string {
	efforts := make([]string, 0, len(raw))
	seen := map[string]bool{}
	for _, effort := range raw {
		if effort == "none" || seen[effort] || !effortValue(effort) {
			continue
		}
		seen[effort] = true
		efforts = append(efforts, effort)
		if len(efforts) >= maxModelEfforts {
			break
		}
	}
	return efforts
}

// sortEfforts 按 effortOrder 排序；未知档位保持原有相对顺序排在已知档位之后。
func sortEfforts(efforts []string) {
	rank := func(effort string) int {
		for index, known := range effortOrder {
			if effort == known {
				return index
			}
		}
		return len(effortOrder)
	}
	sort.SliceStable(efforts, func(left, right int) bool {
		return rank(efforts[left]) < rank(efforts[right])
	})
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
