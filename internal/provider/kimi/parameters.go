package kimi

import (
	"encoding/json"
	"omniproxy/internal/core/proxy"
)

func chatParameters(fields map[string]json.RawMessage, model string) error {
	thinking := true
	if raw, exists := fields["thinking"]; exists {
		if model == "kimi-k3" {
			return proxy.Failure("unsupported_content", 400)
		}
		var value struct {
			Type string  `json:"type"`
			Keep *string `json:"keep"`
		}
		if json.Unmarshal(raw, &value) != nil || value.Type != "enabled" && value.Type != "disabled" || value.Keep != nil && *value.Keep != "all" {
			return proxy.Failure("invalid_request", 400)
		}
		thinking = value.Type == "enabled"
		if model != "kimi-k2.6" && !thinking {
			return proxy.Failure("unsupported_content", 400)
		}
	}
	if raw, exists := fields["reasoning_effort"]; exists {
		if model != "kimi-k3" {
			return proxy.Failure("unsupported_content", 400)
		}
		var effort string
		if json.Unmarshal(raw, &effort) != nil || effort != "low" && effort != "high" && effort != "max" {
			return proxy.Failure("invalid_request", 400)
		}
	}
	temperature := 1.0
	if !thinking {
		temperature = 0.6
	}
	for name, expected := range map[string]float64{"temperature": temperature, "top_p": 0.95, "n": 1, "presence_penalty": 0, "frequency_penalty": 0} {
		if raw, exists := fields[name]; exists {
			var value float64
			if json.Unmarshal(raw, &value) != nil || value != expected {
				return proxy.Failure("unsupported_content", 400)
			}
		}
	}
	if raw, exists := fields["tool_choice"]; exists && model != "kimi-k3" && string(raw) == `"required"` {
		return proxy.Failure("unsupported_content", 400)
	}
	return nil
}
