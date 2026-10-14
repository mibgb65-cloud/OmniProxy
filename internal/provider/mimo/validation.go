package mimo

import (
	"encoding/json"
	"math"

	"omniproxy/internal/core/proxy"
)

func validatedBody(request proxy.Request) ([]byte, error) {
	path := "/v1/" + request.Protocol
	if request.Protocol == "chat" {
		path = "/v1/chat/completions"
	}
	parsed, err := proxy.ParseProtocolRequest(request.Body, path)
	if err != nil {
		return nil, err
	}
	if !SupportsModel(parsed.Model) || request.Model != parsed.Model {
		return nil, proxy.Failure("unsupported_model", 400)
	}
	if parsed.Stream != request.Stream {
		return nil, proxy.Failure("invalid_request", 400)
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(request.Body, &fields) != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	for _, name := range []string{"max_tokens", "max_completion_tokens", "max_output_tokens"} {
		if raw, exists := fields[name]; exists && string(raw) != "null" {
			var limit int
			if json.Unmarshal(raw, &limit) != nil || limit < 1 || limit > maxOutputTokens {
				return nil, proxy.Failure("invalid_request", 400)
			}
		}
	}
	for name, bounds := range map[string][2]float64{"temperature": {0, 1.5}, "top_p": {0.01, 1}} {
		if raw, exists := fields[name]; exists && string(raw) != "null" {
			var value float64
			if json.Unmarshal(raw, &value) != nil || math.IsNaN(value) || value < bounds[0] || value > bounds[1] {
				return nil, proxy.Failure("invalid_request", 400)
			}
		}
	}
	if request.Protocol == "chat" {
		if raw, exists := fields["max_tokens"]; exists && string(raw) != "null" {
			if other, present := fields["max_completion_tokens"]; present && string(other) != "null" {
				return nil, proxy.Failure("invalid_request", 400)
			}
			fields["max_completion_tokens"] = raw
			delete(fields, "max_tokens")
		}
	}
	if request.Protocol == "messages" {
		if raw, exists := fields["system"]; exists && !textContent(raw, 0) {
			return nil, proxy.Failure("unsupported_content", 400)
		}
		if !textInputs(fields["messages"], false) {
			return nil, proxy.Failure("unsupported_content", 400)
		}
	}
	if request.Protocol == "responses" {
		if !textInputs(fields["input"], true) {
			return nil, proxy.Failure("unsupported_content", 400)
		}
		// 上游不保存响应上下文，不能静默丢弃引用并让客户端误以为历史仍然存在。
		if err := statelessResponses(fields); err != nil {
			return nil, err
		}
	}
	if err := validateTools(fields["tools"], request.Protocol); err != nil {
		return nil, err
	}
	if err := normalizeThinking(fields, request.Protocol); err != nil {
		return nil, err
	}
	if request.Protocol == "chat" && request.Stream {
		options := map[string]json.RawMessage{}
		if raw, exists := fields["stream_options"]; exists && string(raw) != "null" {
			if json.Unmarshal(raw, &options) != nil || options == nil {
				return nil, proxy.Failure("invalid_request", 400)
			}
		}
		options["include_usage"] = json.RawMessage("true")
		fields["stream_options"], err = json.Marshal(options)
		if err != nil {
			return nil, proxy.Failure("invalid_request", 400)
		}
	}
	body, err := json.Marshal(fields)
	limit := proxy.MaxRequestBytes
	if request.Protocol == "messages" {
		limit = proxy.MaxMessagesRequestBytes
	}
	if err != nil || len(body) > limit {
		return nil, proxy.Failure("request_too_large", 413)
	}
	return body, nil
}

func statelessResponses(fields map[string]json.RawMessage) error {
	for _, name := range []string{"background", "store"} {
		if raw, exists := fields[name]; exists {
			var enabled bool
			if json.Unmarshal(raw, &enabled) != nil {
				return proxy.Failure("invalid_request", 400)
			}
			if enabled {
				return proxy.Failure("unsupported_content", 400)
			}
			delete(fields, name)
		}
	}
	if raw, exists := fields["previous_response_id"]; exists && string(raw) != "null" {
		var id string
		if json.Unmarshal(raw, &id) != nil {
			return proxy.Failure("invalid_request", 400)
		}
		if id != "" {
			return proxy.Failure("unsupported_content", 400)
		}
		delete(fields, "previous_response_id")
	}
	if raw, exists := fields["context_management"]; exists && string(raw) != "null" {
		return proxy.Failure("unsupported_content", 400)
	}
	return nil
}

func normalizeThinking(fields map[string]json.RawMessage, protocol string) error {
	if raw, exists := fields["thinking"]; exists {
		var thinking struct {
			Type   string `json:"type"`
			Budget *int   `json:"budget_tokens"`
		}
		if json.Unmarshal(raw, &thinking) != nil || thinking.Budget != nil && (*thinking.Budget < 1 || *thinking.Budget > maxOutputTokens) {
			return proxy.Failure("invalid_request", 400)
		}
		if protocol == "messages" && thinking.Type == "adaptive" {
			thinking.Type = "enabled"
		}
		if thinking.Type != "enabled" && thinking.Type != "disabled" {
			return proxy.Failure("invalid_request", 400)
		}
		// MiMo 只有开关；Claude 的 adaptive／预算不能伪装成上游支持的推理档位。
		fields["thinking"] = json.RawMessage(`{"type":"` + thinking.Type + `"}`)
	}
	if raw, exists := fields["reasoning"]; exists {
		var reasoning struct {
			Effort string `json:"effort"`
		}
		if json.Unmarshal(raw, &reasoning) != nil {
			return proxy.Failure("invalid_request", 400)
		}
		switch reasoning.Effort {
		case "", "none", "minimal", "low", "medium", "high", "xhigh":
		default:
			return proxy.Failure("invalid_request", 400)
		}
	}
	return nil
}
