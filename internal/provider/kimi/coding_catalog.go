package kimi

import catalog "omniproxy/internal/core/modelcatalog"

// AccountModels 共享型号仅保留一个逻辑 ID，独有的 K2.8 Preview 不冒充 K2.7。
func AccountModels(o Options) []string {
	if !o.Valid() {
		return nil
	}
	if o.Mode() == "coding" {
		return []string{"kimi-k3", "k3-256k", "kimi-for-coding", "kimi-k2.7-code-highspeed"}
	}
	return Models()
}

// UpstreamModel 只映射已确认的同款模型，普通 API 仍使用原 ID。
func UpstreamModel(o Options, id string) string {
	if o.Mode() == "coding" {
		switch id {
		case "kimi-k3":
			return "k3"
		case "kimi-k2.7-code-highspeed":
			return "kimi-for-coding-highspeed"
		}
	}
	return id
}

// SupportsAccount 根据账号通道校验文本协议，不把订阅能力扩散到普通 API。
func SupportsAccount(o Options, id, protocol string) bool {
	if !o.Valid() {
		return false
	}
	if o.Mode() != "coding" {
		return Supports(id, protocol)
	}
	for _, model := range AccountModels(o) {
		if model == id {
			return protocol == "chat" || protocol == "messages" || protocol == "responses"
		}
	}
	return false
}

// AccountModelMetadata 按实际计费通道提供能力，K3 未核实会员档位时只公布 256K。
func AccountModelMetadata(o Options, id string) catalog.Metadata {
	if o.Mode() != "coding" {
		return ModelMetadata(id)
	}
	if !SupportsAccount(o, id, "chat") {
		return catalog.Metadata{}
	}
	tools, window := true, int64(262144)
	if id == "kimi-for-coding" {
		window = 1048576
	}
	m := catalog.Metadata{ContextWindow: &window, ToolUse: &tools, InputModalities: []string{"text"}}
	if id != "kimi-k2.7-code-highspeed" {
		m.ReasoningEfforts = []catalog.Effort{{Value: "low", Description: "低强度推理"}, {Value: "high", Description: "高强度推理"}, {Value: "max", Description: "最高强度推理"}}
		m.DefaultEffort = "high"
		if id == "kimi-for-coding" {
			m.DefaultEffort = "max"
		}
	}
	m.Description = "Kimi Code 订阅预设，模型权限及 K3 1M 权益以实际会员套餐为准"
	return m
}
