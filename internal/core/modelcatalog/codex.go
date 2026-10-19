package modelcatalog

import (
	"encoding/json"
	"slices"
)

// 兼容模型使用通用编码说明，不复制某个官方模型的能力或专属指令。
const codingInstructions = "You are a coding assistant. Follow the user's request and project instructions. Use available tools when needed, and report results and limitations accurately."

type codexEffort struct {
	Effort      string `json:"effort"`
	Description string `json:"description"`
}

type codexModel struct {
	Slug                     string         `json:"slug"`
	DisplayName              string         `json:"display_name"`
	Description              string         `json:"description"`
	DefaultReasoningLevel    *string        `json:"default_reasoning_level"`
	ReasoningLevels          []codexEffort  `json:"supported_reasoning_levels"`
	ShellType                string         `json:"shell_type"`
	Visibility               string         `json:"visibility"`
	SupportedInAPI           bool           `json:"supported_in_api"`
	Priority                 int            `json:"priority"`
	AvailabilityNUX          any            `json:"availability_nux"`
	Upgrade                  any            `json:"upgrade"`
	BaseInstructions         string         `json:"base_instructions"`
	ModelMessages            map[string]any `json:"model_messages"`
	SupportsReasoningSummary bool           `json:"supports_reasoning_summary_parameter"`
	DefaultReasoningSummary  string         `json:"default_reasoning_summary"`
	SupportVerbosity         bool           `json:"support_verbosity"`
	DefaultVerbosity         any            `json:"default_verbosity"`
	ApplyPatchToolType       *string        `json:"apply_patch_tool_type"`
	TruncationPolicy         truncation     `json:"truncation_policy"`
	ContextWindow            *int64         `json:"context_window,omitempty"`
	ExperimentalTools        []string       `json:"experimental_supported_tools"`
	InputModalities          []string       `json:"input_modalities"`
}

type truncation struct {
	Mode  string `json:"mode"`
	Limit int    `json:"limit"`
}

// CodexJSON 输出 Codex ModelsResponse，而不是普通 OpenAI 模型 ID 列表。
func CodexJSON(entries []Entry) ([]byte, error) {
	entries, err := Normalize(entries)
	if err != nil {
		return nil, err
	}
	entries = ForProtocol(entries, "responses")
	models := make([]codexModel, 0, len(entries))
	for index, entry := range entries {
		efforts := make([]codexEffort, 0, len(entry.Metadata.ReasoningEfforts))
		for _, effort := range entry.Metadata.ReasoningEfforts {
			efforts = append(efforts, codexEffort{effort.Value, effort.Description})
		}
		var defaultEffort *string
		if entry.Metadata.DefaultEffort != "" {
			value := entry.Metadata.DefaultEffort
			defaultEffort = &value
		}
		instructions := codingInstructions
		var patchType *string
		if entry.ProviderKind == "openai" {
			// 原生订阅支持自定义工具；缺失此能力会让核心不注册补丁工具，文件写入无法形成结构化变更项。
			instructions += " Prefer the native apply_patch tool for manual file edits when available, so the client can track structured file changes."
			value := "freeform"
			patchType = &value
		}
		models = append(models, codexModel{
			Slug: entry.ID, DisplayName: entry.Name, Description: entry.Metadata.Description,
			DefaultReasoningLevel: defaultEffort, ReasoningLevels: efforts,
			ApplyPatchToolType: patchType, ShellType: "unified_exec", Visibility: "list", SupportedInAPI: true, Priority: index,
			BaseInstructions: instructions, ModelMessages: map[string]any{"instructions_template": instructions},
			SupportsReasoningSummary: false, DefaultReasoningSummary: "none",
			TruncationPolicy: truncation{"bytes", 10000}, ContextWindow: entry.Metadata.ContextWindow,
			ExperimentalTools: []string{}, InputModalities: codexInputModalities(entry),
		})
	}
	data, err := json.Marshal(struct {
		Models []codexModel `json:"models"`
	}{models})
	if err != nil || len(data) > 512<<10 {
		return nil, ErrInvalid
	}
	return append(data, '\n'), nil
}

// 只有原生订阅 Responses 直通图片；其他适配器的多模态元数据不等于跨协议图片转换能力。
func codexInputModalities(entry Entry) []string {
	modalities := []string{"text"}
	if entry.ProviderKind == "openai" && slices.Contains(entry.Metadata.InputModalities, "image") {
		modalities = append(modalities, "image")
	}
	return modalities
}
