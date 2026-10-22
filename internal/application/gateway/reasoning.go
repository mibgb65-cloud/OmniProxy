package gateway

import (
	"encoding/json"

	"omniproxy/internal/core/proxy"
	"omniproxy/internal/provider/river"
)

func withClaudeEffort(body []byte, effort string) ([]byte, error) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	config := map[string]json.RawMessage{}
	if raw, ok := fields["output_config"]; ok && (json.Unmarshal(raw, &config) != nil || config == nil) {
		return nil, proxy.Failure("invalid_request", 400)
	}
	config["effort"], _ = json.Marshal(effort)
	fields["output_config"], _ = json.Marshal(config)
	// 菜单档位是用户的明确选择，不能被客户端沿用的旧模型 thinking=disabled 覆盖。
	mode := "adaptive"
	if effort == "none" {
		mode = "disabled"
	}
	fields["thinking"], _ = json.Marshal(map[string]string{"type": mode})
	return json.Marshal(fields)
}

func riverMessagesEffort(body []byte, model string) (json.RawMessage, error) {
	var source map[string]json.RawMessage
	if json.Unmarshal(body, &source) != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	raw, exists := source["output_config"]
	if !exists {
		return nil, nil
	}
	var config map[string]json.RawMessage
	if json.Unmarshal(raw, &config) != nil || config == nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	raw, exists = config["effort"]
	if !exists {
		return nil, nil
	}
	var effort string
	if json.Unmarshal(raw, &effort) != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	for _, candidate := range river.ModelMetadata(model).ReasoningEfforts {
		if candidate.Value == effort {
			return raw, nil
		}
	}
	return nil, proxy.Failure("unsupported_reasoning_effort", 400)
}
