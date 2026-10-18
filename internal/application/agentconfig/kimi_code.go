package agentconfig

import (
	"strings"

	"github.com/pelletier/go-toml/v2"
)

func kimiConfig(body []byte) (map[string]any, error) {
	config := map[string]any{}
	if len(body) > 0 && toml.Unmarshal(body, &config) != nil {
		return nil, failure("invalid_config")
	}
	return config, nil
}

func buildKimiCode(body, key []byte, root string, plan Plan, owned, legacy bool) ([]byte, error) {
	config, err := kimiConfig(body)
	if err != nil {
		return nil, err
	}
	providers, err := configObject(config, "providers")
	if err != nil {
		return nil, err
	}
	if providers["omniproxy"] != nil && !owned {
		return nil, failure("profile_conflict")
	}
	models, err := configObject(config, "models")
	if err != nil {
		return nil, err
	}
	for id, raw := range models {
		model, ok := raw.(map[string]any)
		if !ok {
			return nil, failure("invalid_config")
		}
		if model["provider"] == "omniproxy" || strings.HasPrefix(id, "omniproxy/") {
			if !owned || model["provider"] != "omniproxy" {
				return nil, failure("profile_conflict")
			}
			delete(models, id)
		}
	}
	providers["omniproxy"] = map[string]any{"type": "openai", "base_url": root, "api_key": string(key)}
	if legacy {
		providers["omniproxy"].(map[string]any)["type"] = "openai_legacy"
	}
	for _, entry := range plan.Models {
		// Kimi 强制要求窗口大小；未知时采用保守运行预算，不把预算当作厂商已验证能力。
		contextSize := int64(65536)
		if entry.Metadata.ContextWindow != nil {
			contextSize = *entry.Metadata.ContextWindow
		}
		model := map[string]any{"provider": "omniproxy", "model": entry.ID, "display_name": entry.Name, "max_context_size": contextSize, "capabilities": []string{"tool_use"}}
		if legacy {
			// 旧 Python 版使用 openai_legacy 表示 Chat，能力枚举也不接受新版 tool_use 标签。
			delete(model, "capabilities")
		}
		models["omniproxy/"+entry.ID] = model
	}
	config["default_model"] = "omniproxy/" + plan.DefaultModel
	result, err := toml.Marshal(config)
	if err != nil {
		return nil, failure("invalid_config")
	}
	return result, nil
}
