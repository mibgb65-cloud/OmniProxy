package river

import (
	"encoding/json"
	"omniproxy/internal/core/proxy"
)

func validateTools(raw json.RawMessage, protocol string) error {
	if len(raw) == 0 {
		return nil
	}
	var tools []struct {
		Type     string `json:"type"`
		Name     string `json:"name"`
		Function *struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if json.Unmarshal(raw, &tools) != nil || string(raw) == "null" || len(tools) > 128 {
		return proxy.Failure("invalid_request", 400)
	}
	for _, tool := range tools {
		if tool.Type != "function" {
			return proxy.Failure("unsupported_content", 400)
		}
		name := tool.Name
		if protocol == "chat" {
			if tool.Function == nil {
				return proxy.Failure("invalid_request", 400)
			}
			name = tool.Function.Name
		}
		if name == "" || len(name) > 128 {
			return proxy.Failure("invalid_request", 400)
		}
	}
	return nil
}
