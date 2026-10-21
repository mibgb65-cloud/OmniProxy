// Package modelcatalog 定义与客户端、数据库无关的模型目录及格式适配。
package modelcatalog

import (
	"errors"
	"sort"
	"strings"
	"unicode"
)

const MaxModels = 1000

var ErrInvalid = errors.New("models: invalid_catalog")

// Effort 只保存上游明确公布的推理档位。
type Effort struct {
	Value       string `json:"value"`
	Description string `json:"description"`
}

// Metadata 的空值表示尚未确认，不能当作模型不支持或猜测具体额度。
type Metadata struct {
	Description      string `json:"description,omitempty"`
	ContextWindow    *int64 `json:"contextWindow,omitempty"`
	InputTokenLimit  *int64 `json:"inputTokenLimit,omitempty"`
	OutputTokenLimit *int64 `json:"outputTokenLimit,omitempty"`
	// ContextVerified 表示当前授权版本的订阅线路已完成长输入验证，不是客户端的默认值。
	ContextVerified  bool          `json:"contextVerified,omitempty"`
	ReasoningEfforts []Effort      `json:"reasoningEfforts,omitempty"`
	ServiceTiers     []ServiceTier `json:"serviceTiers,omitempty"`
	DefaultEffort    string        `json:"defaultEffort,omitempty"`
	InputModalities  []string      `json:"inputModalities,omitempty"`
	ToolUse          *bool         `json:"toolUse,omitempty"`
	IsDefault        bool          `json:"isDefault,omitempty"`
}

// Entry 代表网关实际可解析的模型；别名继承其当前目标的元数据。
type Entry struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	TargetModel  string   `json:"targetModel"`
	ProviderID   string   `json:"providerId"`
	ProviderKind string   `json:"providerKind"`
	Alias        bool     `json:"alias"`
	Protocols    []string `json:"protocols,omitempty"`
	Metadata     Metadata `json:"metadata"`
}

func textValid(value string, limit int) bool {
	return len(value) <= limit && !strings.ContainsFunc(value, func(r rune) bool {
		return unicode.IsControl(r) && r != '\n' && r != '\t'
	})
}

// ValidateMetadata 校验持久快照和客户端目录的共同边界。
func ValidateMetadata(value Metadata) error {
	if err := validateServiceTiers(value.ServiceTiers); err != nil {
		return err
	}
	for _, limit := range []*int64{value.InputTokenLimit, value.OutputTokenLimit} {
		if limit != nil && (*limit < 1 || *limit > 10000000) {
			return ErrInvalid
		}
	}
	if value.ContextVerified && (value.ContextWindow == nil || *value.ContextWindow != ClaudeLongContextWindow) {
		return ErrInvalid
	}
	if !textValid(value.Description, 2000) || len(value.ReasoningEfforts) > 16 || len(value.InputModalities) > 3 {
		return ErrInvalid
	}
	if value.ContextWindow != nil && (*value.ContextWindow < 1 || *value.ContextWindow > 10000000) {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, effort := range value.ReasoningEfforts {
		if effort.Value == "" || len(effort.Value) > 32 || strings.ContainsFunc(effort.Value, func(r rune) bool {
			return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-')
		}) || !textValid(effort.Description, 500) || seen[effort.Value] {
			return ErrInvalid
		}
		seen[effort.Value] = true
	}
	if value.DefaultEffort != "" && !seen[value.DefaultEffort] {
		return ErrInvalid
	}
	for _, modality := range value.InputModalities {
		if modality != "text" && modality != "image" && modality != "audio" {
			return ErrInvalid
		}
	}
	return nil
}

// Normalize 校验、去重并稳定排序，官方默认模型优先，避免配置随数据库顺序变化。
func Normalize(entries []Entry) ([]Entry, error) {
	if len(entries) > MaxModels {
		return nil, ErrInvalid
	}
	result := make([]Entry, 0, len(entries))
	seen := map[string]bool{}
	for _, entry := range entries {
		if !ValidClaudeModel(entry.ID) || !textValid(entry.Name, 200) || ValidateMetadata(entry.Metadata) != nil {
			return nil, ErrInvalid
		}
		if seen[entry.ID] {
			continue
		}
		seen[entry.ID] = true
		if entry.Name == "" {
			entry.Name = entry.ID
		}
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Metadata.IsDefault != result[j].Metadata.IsDefault {
			return result[i].Metadata.IsDefault
		}
		return result[i].ID < result[j].ID
	})
	return result, nil
}
