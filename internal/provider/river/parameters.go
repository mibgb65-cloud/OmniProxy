package river

import (
	"encoding/json"
	"omniproxy/internal/core/proxy"
)

func validateParameters(fields map[string]json.RawMessage, model, protocol string) error {
	for name, bounds := range map[string][2]float64{"temperature": {0, 2}, "top_p": {0, 1}} {
		if raw, exists := fields[name]; exists {
			var value float64
			if string(raw) == "null" || json.Unmarshal(raw, &value) != nil || value < bounds[0] || value > bounds[1] || name == "top_p" && protocol == "responses" && value == 0 {
				return proxy.Failure("invalid_request", 400)
			}
		}
	}
	if raw, exists := fields["reasoning_effort"]; exists {
		var effort string
		if json.Unmarshal(raw, &effort) != nil || !validEffort(model, effort) || protocol != "chat" {
			return proxy.Failure("invalid_request", 400)
		}
	}
	if raw, exists := fields["reasoning"]; exists {
		var reasoning struct {
			Effort string `json:"effort"`
		}
		if string(raw) == "null" || json.Unmarshal(raw, &reasoning) != nil || protocol != "responses" {
			return proxy.Failure("invalid_request", 400)
		}
		if reasoning.Effort != "" && !validEffort(model, reasoning.Effort) {
			return proxy.Failure("invalid_request", 400)
		}
	}
	if protocol != "responses" {
		return nil
	}
	// River 不保存会话，拒绝续接参数，防止历史被静默丢失。
	for _, name := range []string{"store", "background"} {
		if raw, exists := fields[name]; exists {
			var enabled bool
			if string(raw) == "null" || json.Unmarshal(raw, &enabled) != nil || enabled {
				return proxy.Failure("unsupported_content", 400)
			}
		}
	}
	for _, name := range []string{"previous_response_id", "conversation", "context_management"} {
		if raw, exists := fields[name]; exists && string(raw) != "null" {
			return proxy.Failure("unsupported_content", 400)
		}
	}
	return nil
}

func validEffort(model, effort string) bool {
	for _, candidate := range ModelMetadata(model).ReasoningEfforts {
		if effort == candidate.Value {
			return true
		}
	}
	return false
}
