package agentconfig

import "gopkg.in/yaml.v3"

func buildGoose(body, key []byte, root string, plan Plan, owned bool) ([]byte, error) {
	bundle, err := decodeBundle(body)
	if err != nil {
		return nil, err
	}
	defer bundle.clear()
	config, err := yamlDocument(bundle.Files[0].Body, yaml.MappingNode)
	if err != nil {
		return nil, err
	}
	providers := yamlField(config, "providers")
	if providers == nil {
		providers = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		yamlSet(config, "providers", providers)
	}
	if providers.Kind != yaml.MappingNode {
		return nil, failure("invalid_config")
	}
	if !owned && (bundle.Files[1].Exists || yamlField(providers, "omniproxy") != nil) {
		return nil, failure("profile_conflict")
	}
	provider, err := yamlValue(map[string]any{"enabled": true, "configured": true, "model": plan.DefaultModel})
	if err != nil {
		return nil, err
	}
	yamlSet(providers, "omniproxy", provider)
	active := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "omniproxy"}
	yamlSet(config, "active_provider", active)
	// 同步已有旧版选择字段，避免它覆盖新 provider；不新增旧格式字段触发迁移。
	if yamlField(config, "GOOSE_PROVIDER") != nil || yamlField(config, "GOOSE_MODEL") != nil {
		value := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "omniproxy"}
		yamlSet(config, "GOOSE_PROVIDER", value)
		value = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: plan.DefaultModel}
		yamlSet(config, "GOOSE_MODEL", value)
	}
	models := []any{}
	for _, entry := range plan.Models {
		model := map[string]any{"name": entry.ID, "supports_vision": false}
		if entry.Metadata.ContextWindow != nil {
			model["context_limit"] = *entry.Metadata.ContextWindow
		}
		models = append(models, model)
	}
	// 固定鉴权头局限于本地专用 provider，不覆盖 Goose 的 OpenAI 密钥或系统凭据。
	custom := map[string]any{"name": "omniproxy", "display_name": "OmniProxy", "engine": "openai", "base_url": root + "/chat/completions", "models": models, "headers": map[string]string{"Authorization": "Bearer " + string(key)}, "requires_auth": false, "supports_streaming": true, "dynamic_models": false}
	configBody, err := encodeYAML(config)
	if err != nil {
		return nil, err
	}
	defer clear(configBody)
	providerBody, err := encodeConfig(custom)
	if err != nil {
		return nil, err
	}
	defer clear(providerBody)
	bundle.Files = []backup{{configBody, true}, {providerBody, true}}
	return encodeBundle(bundle)
}
