package stepfun

import (
	"encoding/json"
	"omniproxy/internal/core/proxy"
)

func validatedBody(o Options, request proxy.Request) ([]byte, error) {
	path := "/v1/" + request.Protocol
	if request.Protocol == "chat" {
		path = "/v1/chat/completions"
	}
	parsed, err := proxy.ParseProtocolRequest(request.Body, path)
	if err != nil {
		return nil, err
	}
	if request.Model != parsed.Model || request.Stream != parsed.Stream {
		return nil, proxy.Failure("invalid_request", 400)
	}
	if !Supports(o, parsed.Model, request.Protocol) {
		return nil, proxy.Failure("unsupported_model", 400)
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(request.Body, &fields) != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	maxOutput := 1000000
	if parsed.Model == "step-5-preview" {
		maxOutput = 64000
	}
	if parsed.Model == "step-router-v1" {
		maxOutput = 250000
	}
	for _, name := range []string{"max_tokens", "max_completion_tokens", "max_output_tokens"} {
		if raw, exists := fields[name]; exists {
			var value int
			if json.Unmarshal(raw, &value) != nil || value < 1 || value > maxOutput {
				return nil, proxy.Failure("invalid_request", 400)
			}
		}
	}
	for name, maximum := range map[string]float64{"temperature": 2, "top_p": 1} {
		if raw, exists := fields[name]; exists {
			var value float64
			if json.Unmarshal(raw, &value) != nil || value < 0 || value > maximum {
				return nil, proxy.Failure("invalid_request", 400)
			}
		}
	}
	input := fields["messages"]
	if request.Protocol == "responses" {
		input = fields["input"]
		// 文档未确认服务端历史保存；拒绝依赖服务端状态的请求，保留显式完整历史。
		for _, name := range []string{"store", "background"} {
			if raw, exists := fields[name]; exists {
				var value bool
				if json.Unmarshal(raw, &value) != nil || value {
					return nil, proxy.Failure("unsupported_content", 400)
				}
				delete(fields, name)
			}
		}
		if raw, exists := fields["previous_response_id"]; exists && string(raw) != "null" && string(raw) != `""` {
			return nil, proxy.Failure("unsupported_content", 400)
		}
		if raw, exists := fields["context_management"]; exists && string(raw) != "null" {
			return nil, proxy.Failure("unsupported_content", 400)
		}
	}
	if !textInput(input, 0) {
		return nil, proxy.Failure("unsupported_content", 400)
	}
	if raw, exists := fields["system"]; exists && !textInput(raw, 0) {
		return nil, proxy.Failure("unsupported_content", 400)
	}
	if raw, exists := fields["tools"]; exists {
		var tools []struct {
			Type     string `json:"type"`
			Name     string `json:"name"`
			Function *struct {
				Name string `json:"name"`
			} `json:"function"`
		}
		if json.Unmarshal(raw, &tools) != nil || len(tools) > 128 {
			return nil, proxy.Failure("invalid_request", 400)
		}
		for _, tool := range tools {
			if request.Protocol == "messages" {
				if tool.Type != "" && tool.Type != "custom" {
					return nil, proxy.Failure("unsupported_content", 400)
				}
			} else if tool.Type != "function" {
				return nil, proxy.Failure("unsupported_content", 400)
			}
			name := tool.Name
			if tool.Function != nil {
				name = tool.Function.Name
			}
			if len(name) == 0 || len(name) > 128 {
				return nil, proxy.Failure("invalid_request", 400)
			}
		}
	}
	if parsed.Model == "step-router-v1" {
		// Router 会忽略 effort；显式拒绝，不能让客户端误认为档位生效。
		if raw, exists := fields["output_config"]; exists && string(raw) != "null" {
			return nil, proxy.Failure("unsupported_content", 400)
		}
	}
	body, err := json.Marshal(fields)
	if err != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	limit := proxy.MaxRequestBytes
	if request.Protocol == "messages" {
		limit = proxy.MaxMessagesRequestBytes
	}
	if len(body) > limit {
		clear(body)
		return nil, proxy.Failure("request_too_large", 413)
	}
	return body, nil
}

func textInput(raw json.RawMessage, depth int) bool {
	if depth > 8 || len(raw) == 0 {
		return false
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return true
	}
	var items []json.RawMessage
	if json.Unmarshal(raw, &items) == nil && items != nil {
		if len(items) > 4096 {
			return false
		}
		for _, item := range items {
			if !textInput(item, depth+1) {
				return false
			}
		}
		return true
	}
	var block map[string]json.RawMessage
	if json.Unmarshal(raw, &block) != nil || block == nil {
		return false
	}
	var kind string
	if field, exists := block["type"]; exists && json.Unmarshal(field, &kind) != nil {
		return false
	}
	switch kind {
	case "", "message", "text", "input_text", "output_text", "thinking", "redacted_thinking", "reasoning", "tool_use", "tool_result", "function_call", "function_call_output":
	default:
		return false
	}
	for _, name := range []string{"image_url", "video_url", "input_audio", "document"} {
		if _, exists := block[name]; exists {
			return false
		}
	}
	if content, exists := block["content"]; exists && string(content) != "null" {
		return textInput(content, depth+1)
	}
	return true
}
