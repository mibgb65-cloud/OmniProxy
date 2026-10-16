package kimi

import (
	"encoding/json"
	"omniproxy/internal/core/proxy"
)

func accountChatParameters(o Options, fields map[string]json.RawMessage, model string) error {
	if o.Mode() != "coding" {
		return chatParameters(fields, model)
	}
	if raw, ok := fields["reasoning_effort"]; ok {
		var effort string
		if json.Unmarshal(raw, &effort) != nil || (effort != "low" && effort != "high" && effort != "max") {
			return proxy.Failure("invalid_request", 400)
		}
	}
	return nil
}

func codingParameters(fields map[string]json.RawMessage, request proxy.Request) error {
	// 关闭思考可能由订阅服务换成另一模型，首版不静默放宽模型身份。
	if raw, ok := fields["thinking"]; ok {
		var value struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(raw, &value) != nil || (value.Type != "enabled" && value.Type != "adaptive") {
			return proxy.Failure("unsupported_content", 400)
		}
	}
	if request.Model == "kimi-k2.7-code-highspeed" {
		for _, key := range []string{"reasoning_effort", "reasoning", "output_config"} {
			if _, ok := fields[key]; ok {
				return proxy.Failure("unsupported_content", 400)
			}
		}
	}
	return nil
}
