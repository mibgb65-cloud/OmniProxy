package agentconfig

import (
	"encoding/json"
	"fmt"

	"github.com/pelletier/go-toml/v2"
	catalog "omniproxy/internal/core/modelcatalog"
)

func (s *Service) buildPlan(client string, body, key []byte, plan Plan) ([]byte, string, error) {
	patched, provider, err := build(client, body, key, s.port, plan.DefaultModel)
	if err != nil {
		return nil, provider, err
	}
	if client == "codex" {
		defer clear(patched)
		updates := map[string]string{}
		if plan.Mode == "remote-catalog" {
			updates["model_catalog_json"] = ""
			updates["features.api_key_model_discovery"] = "true"
			updates["model_providers."+provider+".model_catalog_url"] = quote(fmt.Sprintf("http://127.0.0.1:%d/v1/codex-models", s.port))
		} else {
			updates["model_catalog_json"] = quote(s.catalogPath())
		}
		var original map[string]any
		if toml.Unmarshal(body, &original) == nil {
			if effort, ok := original["model_reasoning_effort"].(string); ok {
				supported := false
				confirmed := false
				for _, model := range plan.Models {
					if model.ID == plan.DefaultModel {
						confirmed = len(model.Metadata.ReasoningEfforts) > 0
						for _, item := range model.Metadata.ReasoningEfforts {
							if item.Value == effort {
								supported = true
							}
						}
					}
				}
				if !supported && (confirmed || original["model"] != plan.DefaultModel) {
					updates["model_reasoning_effort"] = ""
				}
			}
		}
		result, err := patchTOML(patched, provider, updates)
		return result, provider, err
	}
	if client != "claude-desktop" {
		if client == "claude" {
			defer clear(patched)
			var config map[string]any
			if json.Unmarshal(patched, &config) != nil {
				return nil, provider, failure("invalid_config")
			}
			claudePickerSettings(config, plan.Models)
			if err := subscriptionCompactSettings(config, plan.Models); err != nil {
				return nil, provider, err
			}
			result, err := json.MarshalIndent(config, "", "  ")
			if err != nil {
				return nil, provider, failure("invalid_config")
			}
			return append(result, '\n'), provider, nil
		}
		return patched, provider, nil
	}
	defer clear(patched)
	var config map[string]any
	if json.Unmarshal(patched, &config) != nil {
		return nil, provider, failure("invalid_config")
	}
	config["inferenceModels"] = catalog.DesktopModels(plan.Models, plan.DefaultModel)
	config["modelDiscoveryEnabled"] = false
	result, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, provider, failure("invalid_config")
	}
	return append(result, '\n'), provider, nil
}
