package agentconfig

import catalog "omniproxy/internal/core/modelcatalog"

func claudePickerSettings(config map[string]any, entries []catalog.Entry) {
	options := make([]map[string]string, 0, len(entries))
	for _, entry := range entries {
		options = append(options, map[string]string{"model": entry.ID, "label": entry.Name})
	}
	// 档位映射不会注册模型菜单；显式替换内置列表，避免重复档位和不可路由的官方模型。
	// 不设置 behavesAs，不能把第三方模型伪装为 Claude 并据此推断能力。
	config["modelPicker"] = map[string]any{"options": options, "replaceBuiltInOptions": true}
}
