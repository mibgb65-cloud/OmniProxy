package agentconfig

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
)

func harnessPlugin(root *yaml.Node) (*yaml.Node, error) {
	var found *yaml.Node
	for _, entry := range root.Content {
		if entry.Kind != yaml.MappingNode {
			return nil, failure("invalid_config")
		}
		id, name := yamlField(entry, "id"), yamlField(entry, "name")
		if id != nil && id.Value == "llm-pi-ai" || name != nil && name.Value == "@deepseek-ai/dsh-llm-pi-ai" {
			if found != nil {
				return nil, failure("profile_conflict")
			}
			found = entry
		}
	}
	if found == nil {
		found = &yaml.Node{Kind: yaml.MappingNode}
		yamlSet(found, "id", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "llm-pi-ai"})
		root.Content = append(root.Content, found)
	}
	return found, nil
}

func buildHarness(client string, body, key []byte, port int, plan Plan) ([]byte, string, error) {
	var bundle harnessBundle
	if json.Unmarshal(body, &bundle) != nil {
		return nil, "", failure("invalid_config")
	}
	root, err := yamlDocument(bundle.Config.Body, yaml.SequenceNode)
	if err != nil {
		return nil, "", err
	}
	plugin, err := harnessPlugin(root)
	if err != nil {
		return nil, "", err
	}
	config := yamlField(plugin, "config")
	if config == nil {
		config = &yaml.Node{Kind: yaml.MappingNode}
		yamlSet(plugin, "config", config)
	}
	if config.Kind != yaml.MappingNode {
		return nil, "", failure("invalid_config")
	}
	providers := yamlField(config, "providers")
	if providers == nil {
		providers = &yaml.Node{Kind: yaml.MappingNode}
		yamlSet(config, "providers", providers)
	}
	if providers.Kind != yaml.MappingNode {
		return nil, "", failure("invalid_config")
	}
	if prior := yamlField(providers, "omniproxy"); prior != nil {
		ref := yamlField(prior, "apiKeyEnv")
		if ref == nil || ref.Value != harnessRef(client) {
			return nil, "", failure("profile_conflict")
		}
	}
	models := []map[string]any{}
	for _, entry := range plan.Models {
		// pi-ai 的 input 显式非空即优先于安装目录；写死 [text] 会让上游能看图的路由
		// 在 Harness 里无法附图，因此按已确认能力输出。
		model := map[string]any{"id": entry.ID, "name": entry.Name, "input": clientInputModalities(entry)}
		// Harness 只在模型声明 reasoning levels 时显示 Effort 菜单；未确认档位的模型写 false，
		// 保持菜单不出现该项，由上游默认值决定是否思考。
		if efforts := harnessReasoningEfforts(entry); efforts != nil {
			model["reasoningEfforts"] = efforts
			if compat := harnessThinkingFormat(entry, efforts); compat != "" {
				model["compat"] = map[string]any{"thinkingFormat": compat}
			}
		} else {
			model["reasoningEfforts"] = false
		}
		if entry.Metadata.ContextWindow != nil {
			model["contextWindow"] = *entry.Metadata.ContextWindow
		}
		if entry.Metadata.OutputTokenLimit != nil {
			model["maxTokens"] = *entry.Metadata.OutputTokenLimit
		}
		models = append(models, model)
	}
	route, err := yamlValue(map[string]any{"displayName": "OmniProxy", "api": "openai-completions", "apiKeyEnv": harnessRef(client), "baseURL": fmt.Sprintf("http://127.0.0.1:%d/v1", port), "compat": map[string]any{"supportsDeveloperRole": false, "maxTokensField": "max_tokens"}, "models": models})
	if err != nil {
		return nil, "", err
	}
	yamlSet(providers, "omniproxy", route)
	bundle.Config.Body, err = encodeYAML(root)
	if err != nil {
		return nil, "", err
	}
	bundle.Config.Exists = true
	value := string(key)
	bundle.Key = &value
	encoded, err := json.Marshal(bundle)
	if err != nil {
		return nil, "", failure("invalid_config")
	}
	return encoded, "omniproxy", nil
}
