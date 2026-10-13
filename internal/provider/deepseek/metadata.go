package deepseek

import catalog "omniproxy/internal/core/modelcatalog"

// ModelMetadata 只补充官方已确认的上下文能力；发现接口没有公布的未知模型保持未知。
// 来源：https://api-docs.deepseek.com/quick_start/pricing/
func ModelMetadata(model string) catalog.Metadata {
	for _, id := range Models() {
		if model == id {
			window := catalog.ClaudeLongContextWindow
			return catalog.Metadata{ContextWindow: &window}
		}
	}
	return catalog.Metadata{}
}
