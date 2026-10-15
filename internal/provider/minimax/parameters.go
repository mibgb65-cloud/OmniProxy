package minimax

import (
	"encoding/json"
	"omniproxy/internal/core/proxy"
)

func parameters(fields map[string]json.RawMessage, request proxy.Request) error {
	for name, maximum := range map[string]float64{"temperature": 2, "top_p": 1} {
		if request.Protocol == "responses" && name == "temperature" {
			maximum = 1
		}
		if raw, ok := fields[name]; ok {
			var value *float64
			if json.Unmarshal(raw, &value) != nil || value == nil || *value < 0 || *value > maximum {
				return proxy.Failure("invalid_request", 400)
			}
		}
	}
	for _, name := range []string{"n"} {
		if raw, ok := fields[name]; ok {
			var n int
			if json.Unmarshal(raw, &n) != nil || n != 1 {
				return proxy.Failure("unsupported_content", 400)
			}
		}
	}
	if raw, ok := fields["service_tier"]; ok {
		var tier string
		if json.Unmarshal(raw, &tier) != nil || tier != "standard" && tier != "priority" {
			return proxy.Failure("invalid_request", 400)
		}
	}
	if raw, ok := fields["reasoning_effort"]; ok {
		if err := effort(raw, request.Model); err != nil {
			return err
		}
	}
	for _, name := range []string{"reasoning", "output_config"} {
		if raw, ok := fields[name]; ok {
			var object map[string]json.RawMessage
			if json.Unmarshal(raw, &object) != nil || object == nil {
				return proxy.Failure("invalid_request", 400)
			}
			if e, ok := object["effort"]; ok {
				if err := effort(e, request.Model); err != nil {
					return err
				}
			}
		}
	}
	if raw, ok := fields["thinking"]; ok {
		var thinking struct {
			Type   string `json:"type"`
			Budget *int64 `json:"budget_tokens"`
		}
		if json.Unmarshal(raw, &thinking) != nil || thinking.Type != "disabled" && thinking.Type != "adaptive" && thinking.Type != "enabled" || thinking.Budget != nil && (*thinking.Budget < 1 || *thinking.Budget > 1000000) {
			return proxy.Failure("invalid_request", 400)
		}
		if thinking.Type == "disabled" && request.Model != "MiniMax-M3" {
			return proxy.Failure("unsupported_content", 400)
		}
	}
	if request.Protocol == "responses" {
		if raw, ok := fields["tool_choice"]; ok && string(raw) != `"none"` && string(raw) != `"auto"` {
			return proxy.Failure("unsupported_content", 400)
		}
		if raw, ok := fields["truncation"]; ok && string(raw) != `"disabled"` {
			return proxy.Failure("unsupported_content", 400)
		}
	}
	return nil
}

func effort(raw json.RawMessage, model string) error {
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return proxy.Failure("invalid_request", 400)
	}
	switch value {
	case "low", "medium", "high", "xhigh", "max":
		return nil
	case "none":
		if model == "MiniMax-M3" {
			return nil
		}
	}
	return proxy.Failure("unsupported_content", 400)
}
