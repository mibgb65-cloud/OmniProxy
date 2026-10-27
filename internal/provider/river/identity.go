package river

// PublicModel 将已确认的 River 型号映射到稳定的逻辑模型，未知名称不做模糊匹配。
func PublicModel(model string) string {
	switch model {
	case Qwen:
		return "Qwen3.8-27B-FP8"
	case DeepSeekFlash:
		return "deepseek-flash"
	case DeepSeekFlash1017:
		return "DeepSeek-V4.1-Flash-1017"
	case KimiK3:
		return "Kimi-K3"
	case NemotronVL:
		return "NVIDIA-Nemotron-3.5"
	case GLMFlash:
		return "GLM-5.3-Flash"
	default:
		return model
	}
}

// UpstreamModel 只在选择 River 后还原其官方模型 ID。
func UpstreamModel(model string) string {
	for _, upstream := range Models() {
		if PublicModel(upstream) == model {
			return upstream
		}
	}
	return model
}

// ModelName 返回界面和客户端菜单的短名称。
func ModelName(model string) string {
	model = PublicModel(model)
	if model == "deepseek-flash" {
		return "DeepSeek-V4.1-Flash"
	}
	return model
}

// DisplayModels 返回提供商卡片的品牌型号名称，实际请求 ID 仍由 Models 提供。
func DisplayModels() []string {
	models := Models()
	for i, id := range models {
		models[i] = ModelName(id)
	}
	return models
}
