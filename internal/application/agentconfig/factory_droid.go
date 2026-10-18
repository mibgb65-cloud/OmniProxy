package agentconfig

import (
	"strconv"
	"strings"
)

const droidModelPrefix = "OmniProxy "

// Droid 的 CLI 自定义标识包含展示名称和数组下标，不能直接把 API 模型 ID 当作默认值。
func droidModelRef(model map[string]any, index int) string {
	name, _ := model["displayName"].(string)
	return "custom:" + strings.ReplaceAll(name, " ", "-") + "-" + strconv.Itoa(index)
}

func buildFactoryDroid(body, key []byte, root string, plan Plan, owned bool) ([]byte, error) {
	_, config, err := openCodeConfig(body)
	if err != nil {
		return nil, err
	}
	current, ok := config["customModels"].([]any)
	if config["customModels"] != nil && !ok {
		return nil, failure("invalid_config")
	}
	models := []any{}
	for _, raw := range current {
		model, ok := raw.(map[string]any)
		if !ok {
			return nil, failure("invalid_config")
		}
		name, _ := model["displayName"].(string)
		if strings.HasPrefix(name, droidModelPrefix) {
			if !owned {
				return nil, failure("profile_conflict")
			}
			continue
		}
		models = append(models, model)
	}
	// 受管模型追加在原目录之后，保留既有模型下标以及其他设置中的引用。
	for _, entry := range plan.Models {
		model := map[string]any{"model": entry.ID, "displayName": droidModelPrefix + entry.Name, "baseUrl": root, "apiKey": string(key), "provider": "generic-chat-completion-api", "noImageSupport": true}
		if entry.Metadata.OutputTokenLimit != nil {
			model["maxOutputTokens"] = *entry.Metadata.OutputTokenLimit
		}
		if entry.Metadata.ContextWindow != nil {
			contextSize := *entry.Metadata.ContextWindow
			if entry.Metadata.OutputTokenLimit != nil {
				contextSize -= *entry.Metadata.OutputTokenLimit
			}
			if contextSize > 0 {
				model["maxContextLimit"] = contextSize
			}
		}
		if entry.ID == plan.DefaultModel {
			config["model"] = droidModelRef(model, len(models))
		}
		models = append(models, model)
	}
	config["customModels"] = models
	return encodeConfig(config)
}
