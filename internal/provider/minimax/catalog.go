package minimax

import catalog "omniproxy/internal/core/modelcatalog"

// PreviewModel 当前仅面向 M Plan，需订阅账号成功发现后再加入目录。
const PreviewModel = "MiniMax-M3.1-Flash-Preview"

// Models 返回当前首批文本模型的真实 ID，排除语音和生成接口。
func Models() []string {
	return []string{"MiniMax-M3", "MiniMax-M2.7", "MiniMax-M2.7-highspeed", "MiniMax-M2.5", "MiniMax-M2.5-highspeed"}
}

// Supports 按账号类型限制预览模型，具体权限仍由官方账号决定。
func Supports(o Options, model, protocol string) bool {
	if !o.Valid() || (protocol != "chat" && protocol != "messages" && protocol != "responses") {
		return false
	}
	if model == PreviewModel {
		return o.BillingMode == "subscription"
	}
	for _, id := range Models() {
		if id == model {
			return true
		}
	}
	return false
}

// AccountModels 预览模型只在订阅账号成功同步后加入，不能替旧套餐假定权限。
func AccountModels(o Options) []string {
	if !o.Valid() {
		return nil
	}
	return Models()
}

// ModelMetadata 区分官方标称窗口和已经验证的能力，不伪造输出长度。
func ModelMetadata(id string) catalog.Metadata {
	if !Supports(Options{Region: "cn", BillingMode: "subscription"}, id, "chat") {
		return catalog.Metadata{}
	}
	window, tools := int64(204800), true
	if id == "MiniMax-M3" || id == PreviewModel {
		window = 1000000
	}
	m := catalog.Metadata{ContextWindow: &window, ToolUse: &tools, InputModalities: []string{"text"}}
	if id == PreviewModel {
		m.ReasoningEfforts = []catalog.Effort{{Value: "low", Description: "低强度推理"}, {Value: "medium", Description: "中等强度推理"}, {Value: "high", Description: "高强度推理"}, {Value: "xhigh", Description: "更高强度推理"}, {Value: "max", Description: "最高强度推理"}}
		m.DefaultEffort = "max"
	} else if id == "MiniMax-M3" {
		m.Description = "支持原生思考开关；不声明多档推理深度"
	} else {
		m.Description = "始终思考；不声明多档推理深度"
	}
	return m
}
