package agentconfig

import "strings"

func buildPi(body, key []byte, root string, plan Plan, owned bool) ([]byte, error) {
	bundle, err := decodeBundle(body)
	if err != nil {
		return nil, err
	}
	defer bundle.clear()
	_, config, err := openCodeConfig(bundle.Files[0].Body)
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
	models := []any{}
	for _, entry := range plan.Models {
		// 与 Harness 同为 pi-ai 配置：input 显式非空即优先，需按已确认能力声明。
		model := map[string]any{"id": entry.ID, "name": entry.Name, "input": clientInputModalities(entry)}
		if entry.Metadata.ContextWindow != nil {
			model["contextWindow"] = *entry.Metadata.ContextWindow
		}
		if entry.Metadata.OutputTokenLimit != nil {
			model["maxTokens"] = *entry.Metadata.OutputTokenLimit
		}
		models = append(models, model)
	}
	providers["omniproxy"] = map[string]any{"name": "OmniProxy", "api": "openai-completions", "baseUrl": root, "apiKey": string(key), "authHeader": true, "models": models}
	_, settings, err := openCodeConfig(bundle.Files[1].Body)
	if err != nil {
		return nil, err
	}
	settings["defaultProvider"], settings["defaultModel"] = "omniproxy", plan.DefaultModel
	if err := piScopedModels(settings, plan, owned); err != nil {
		return nil, err
	}
	modelBody, err := encodeConfig(config)
	if err != nil {
		return nil, err
	}
	defer clear(modelBody)
	settingsBody, err := encodeConfig(settings)
	if err != nil {
		return nil, err
	}
	defer clear(settingsBody)
	bundle.Files = []backup{{modelBody, true}, {settingsBody, true}}
	return encodeBundle(bundle)
}

func piScopedModels(settings map[string]any, plan Plan, owned bool) error {
	current, ok := settings["enabledModels"].([]any)
	if settings["enabledModels"] != nil && !ok {
		return failure("invalid_config")
	}
	if len(current) == 0 {
		return nil
	}
	patterns := []any{}
	for _, raw := range current {
		pattern, ok := raw.(string)
		if !ok {
			return failure("invalid_config")
		}
		if !owned || !strings.HasPrefix(pattern, "omniproxy/") {
			patterns = append(patterns, pattern)
		}
	}
	for _, model := range plan.Models {
		patterns = append(patterns, "omniproxy/"+model.ID)
	}
	settings["enabledModels"] = patterns
	return nil
}
