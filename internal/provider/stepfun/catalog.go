package stepfun

import catalog "omniproxy/internal/core/modelcatalog"

// Models 返回官方文本预设；预设不代表真实账号权限。
func Models() []string {
	return []string{"step-5-preview", "step-3.7-flash", "step-3.5-flash", "step-3.5-flash-2603"}
}

// Supports 按已确认模型和协议限制转发，不推测 Router 的 Responses 能力。
func Supports(o Options, model, protocol string) bool {
	if !o.Valid() {
		return false
	}
	if protocol == "responses" {
		return model == "step-5-preview" || model == "step-3.7-flash"
	}
	if protocol != "chat" && protocol != "messages" {
		return false
	}
	for _, id := range Models() {
		if model == id {
			return true
		}
	}
	return o.Region == "cn" && o.BillingMode == "step_plan" && model == "step-router-v1"
}

// AccountModels 区分订阅预设和按量目录，海外 Router 尚未确认。
func AccountModels(o Options) []string {
	if !o.Valid() {
		return nil
	}
	ids := Models()
	if o.Region == "cn" && o.BillingMode == "step_plan" {
		ids = append(ids, "step-router-v1")
	}
	return ids
}

// ModelMetadata 仅声明文本与已核对限制，实际长窗口验证保持独立状态。
func ModelMetadata(model string) catalog.Metadata {
	tools := true
	m := catalog.Metadata{InputModalities: []string{"text"}, ToolUse: &tools}
	if model == "step-5-preview" || model == "step-3.7-flash" {
		window := int64(1000000)
		if model == "step-3.7-flash" {
			window = 256000
		}
		m.ContextWindow = &window
		m.ReasoningEfforts = []catalog.Effort{{Value: "low", Description: "低强度推理"}, {Value: "medium", Description: "中等强度推理"}, {Value: "high", Description: "高强度推理"}}
		m.DefaultEffort = "medium"
	}
	if model == "step-5-preview" {
		output := int64(64000)
		m.OutputTokenLimit = &output
	}
	return m
}
