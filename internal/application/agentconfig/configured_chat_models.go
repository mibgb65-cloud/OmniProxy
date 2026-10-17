package agentconfig

import (
	"encoding/json"
	"strings"

	"gopkg.in/yaml.v3"
)

func configuredChat(client string, body []byte) ([]ModelSummary, error) {
	value, _, err := openCodeConfig(body)
	if err != nil {
		return nil, err
	}
	value.Standardize()
	var stored struct {
		Providers map[string]struct {
			Models map[string]struct {
				Name string `json:"name"`
			} `json:"models"`
		} `json:"provider"`
		ModelProviders struct {
			OpenAI []struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				EnvKey string `json:"envKey"`
			} `json:"openai"`
		} `json:"modelProviders"`
	}
	if json.Unmarshal(value.Pack(), &stored) != nil {
		return nil, failure("invalid_config")
	}
	result := []ModelSummary{}
	if client == "qwen-code" {
		for _, model := range stored.ModelProviders.OpenAI {
			if model.EnvKey == qwenCredentialRef {
				result = append(result, ModelSummary{model.ID, strings.TrimPrefix(model.Name, "OmniProxy · ")})
			}
		}
	} else {
		for id, model := range stored.Providers["omniproxy"].Models {
			result = append(result, ModelSummary{id, model.Name})
		}
	}
	return result, nil
}

func configuredHarness(body []byte) ([]ModelSummary, error) {
	var bundle harnessBundle
	if json.Unmarshal(body, &bundle) != nil {
		return nil, failure("invalid_config")
	}
	defer clear(bundle.Config.Body)
	root, err := yamlDocument(bundle.Config.Body, yaml.SequenceNode)
	if err != nil {
		return nil, err
	}
	plugin, err := harnessPlugin(root)
	if err != nil {
		return nil, err
	}
	config := yamlField(plugin, "config")
	providers := yamlField(config, "providers")
	provider := yamlField(providers, "omniproxy")
	models := yamlField(provider, "models")
	if models == nil {
		return []ModelSummary{}, nil
	}
	var stored []struct {
		ID   string `yaml:"id"`
		Name string `yaml:"name"`
	}
	if models.Decode(&stored) != nil {
		return nil, failure("invalid_config")
	}
	result := make([]ModelSummary, 0, len(stored))
	for _, model := range stored {
		result = append(result, ModelSummary{model.ID, model.Name})
	}
	return result, nil
}
