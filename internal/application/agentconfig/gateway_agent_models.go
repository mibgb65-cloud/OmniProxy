package agentconfig

import (
	"strings"

	"gopkg.in/yaml.v3"
)

func gatewayAgentModel(client string, body []byte) string {
	state, err := gatewayAgentState(client, body)
	if err != nil {
		return ""
	}
	return state.DefaultModel
}

func configuredGatewayAgent(client string, body []byte) ([]ModelSummary, error) {
	state, err := gatewayAgentState(client, body)
	return state.Models, err
}

func gatewayAgentState(client string, body []byte) (ConfiguredCatalog, error) {
	state := ConfiguredCatalog{Models: []ModelSummary{}}
	if client == "kimi-code" {
		config, err := kimiConfig(body)
		if err != nil {
			return state, err
		}
		models, err := configObject(config, "models")
		if err != nil {
			return state, err
		}
		selected, _ := config["default_model"].(string)
		for alias, raw := range models {
			model, ok := raw.(map[string]any)
			if !ok || model["provider"] != "omniproxy" {
				continue
			}
			id, _ := model["model"].(string)
			name, _ := model["display_name"].(string)
			state.Models = append(state.Models, ModelSummary{id, name})
			if alias == selected {
				state.DefaultModel = id
			}
		}
		return state, nil
	}
	if client == "factory-droid" {
		_, config, err := openCodeConfig(body)
		if err != nil {
			return state, err
		}
		models, _ := config["customModels"].([]any)
		selected, _ := config["model"].(string)
		for index, raw := range models {
			model, ok := raw.(map[string]any)
			if !ok {
				return state, failure("invalid_config")
			}
			name, _ := model["displayName"].(string)
			if !strings.HasPrefix(name, droidModelPrefix) {
				continue
			}
			id, _ := model["model"].(string)
			state.Models = append(state.Models, ModelSummary{id, strings.TrimPrefix(name, droidModelPrefix)})
			if selected == droidModelRef(model, index) {
				state.DefaultModel = id
			}
		}
		return state, nil
	}
	bundle, err := decodeBundle(body)
	if err != nil {
		return state, err
	}
	defer bundle.clear()
	var provider map[string]any
	idField, nameField := "id", "name"
	if client == "pi" {
		_, config, err := openCodeConfig(bundle.Files[0].Body)
		if err != nil {
			return state, err
		}
		providers, err := configObject(config, "providers")
		if err != nil {
			return state, err
		}
		provider, _ = providers["omniproxy"].(map[string]any)
		_, settings, err := openCodeConfig(bundle.Files[1].Body)
		if err != nil {
			return state, err
		}
		if settings["defaultProvider"] == "omniproxy" {
			state.DefaultModel, _ = settings["defaultModel"].(string)
		}
	} else {
		config, err := yamlDocument(bundle.Files[0].Body, yaml.MappingNode)
		if err != nil {
			return state, err
		}
		active := yamlField(config, "active_provider")
		selected := yamlField(yamlField(yamlField(config, "providers"), "omniproxy"), "model")
		if active != nil && active.Value == "omniproxy" && selected != nil {
			state.DefaultModel = selected.Value
		} else if legacy := yamlField(config, "GOOSE_PROVIDER"); legacy != nil && legacy.Value == "omniproxy" {
			if model := yamlField(config, "GOOSE_MODEL"); model != nil {
				state.DefaultModel = model.Value
			}
		}
		_, provider, err = openCodeConfig(bundle.Files[1].Body)
		if err != nil {
			return state, err
		}
		idField, nameField = "name", "name"
	}
	models, ok := provider["models"].([]any)
	if provider["models"] != nil && !ok {
		return state, failure("invalid_config")
	}
	for _, raw := range models {
		model, ok := raw.(map[string]any)
		if !ok {
			return state, failure("invalid_config")
		}
		id, _ := model[idField].(string)
		name, _ := model[nameField].(string)
		state.Models = append(state.Models, ModelSummary{id, name})
	}
	return state, nil
}
