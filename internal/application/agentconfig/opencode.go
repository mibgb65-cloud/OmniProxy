package agentconfig

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tailscale/hujson"
	catalog "omniproxy/internal/core/modelcatalog"
)

func openCodeConfig(body []byte) (hujson.Value, map[string]any, error) {
	if len(body) == 0 {
		body = []byte("{}")
	}
	value, err := hujson.Parse(body)
	if err != nil {
		return value, nil, failure("invalid_config")
	}
	count := 0
	for part := range value.All() {
		count++
		if count > 20000 {
			return value, nil, failure("invalid_config")
		}
		if object, ok := part.Value.(*hujson.Object); ok {
			seen := map[string]bool{}
			for _, member := range object.Members {
				var key string
				if json.Unmarshal(member.Name.Pack(), &key) != nil || seen[key] {
					return value, nil, failure("invalid_config")
				}
				seen[key] = true
			}
		}
	}
	standard := value.Clone()
	standard.Standardize()
	config := map[string]any{}
	if json.Unmarshal(standard.Pack(), &config) != nil || config == nil {
		return value, nil, failure("invalid_config")
	}
	return value, config, nil
}

func buildOpenCode(body, key []byte, port int, plan Plan, owned bool) ([]byte, string, error) {
	value, config, err := openCodeConfig(body)
	if err != nil {
		return nil, "", err
	}
	providers, ok := config["provider"].(map[string]any)
	if config["provider"] != nil && !ok {
		return nil, "", failure("invalid_config")
	}
	root := fmt.Sprintf("http://127.0.0.1:%d/v1", port)
	if prior, exists := providers["omniproxy"]; exists {
		p, _ := prior.(map[string]any)
		options, _ := p["options"].(map[string]any)
		if options["baseURL"] != root && !owned {
			return nil, "", failure("profile_conflict")
		}
	}
	models := map[string]any{}
	for _, entry := range plan.Models {
		model := map[string]any{"name": entry.Name, "modalities": map[string]any{"input": []string{"text"}, "output": []string{"text"}}}
		limits := map[string]int64{}
		if entry.Metadata.ContextWindow != nil {
			limits["context"] = *entry.Metadata.ContextWindow
		}
		if entry.Metadata.OutputTokenLimit != nil {
			limits["output"] = *entry.Metadata.OutputTokenLimit
		}
		if len(limits) > 0 {
			model["limit"] = limits
		}
		models[entry.ID] = model
	}
	provider := map[string]any{"name": "OmniProxy", "npm": "@ai-sdk/openai-compatible", "options": map[string]any{"baseURL": root, "apiKey": string(key)}, "models": models}
	patch := []map[string]any{}
	if providers == nil {
		patch = append(patch, map[string]any{"op": "add", "path": "/provider", "value": map[string]any{}})
	}
	patch = append(patch, map[string]any{"op": "add", "path": "/provider/omniproxy", "value": provider}, map[string]any{"op": "add", "path": "/model", "value": "omniproxy/" + plan.DefaultModel})
	encoded, err := json.Marshal(patch)
	defer clear(encoded)
	if err != nil || value.Patch(encoded) != nil {
		return nil, "", failure("invalid_config")
	}
	value.Format()
	return value.Pack(), "omniproxy", nil
}

func openCodeModel(body []byte) string {
	_, config, err := openCodeConfig(body)
	if err != nil {
		return ""
	}
	model, _ := config["model"].(string)
	return strings.TrimPrefix(model, "omniproxy/")
}

func gatewayEntries(entries []catalog.Entry) []catalog.Entry {
	return catalog.ForProtocol(entries, "chat_completions")
}
