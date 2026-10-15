package minimax

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
	maxOutput := int(*ModelMetadata(parsed.Model).ContextWindow)
	for _, name := range []string{"max_tokens", "max_completion_tokens", "max_output_tokens"} {
		if raw, exists := fields[name]; exists {
			var value int
			if json.Unmarshal(raw, &value) != nil || value < 1 || value > maxOutput {
				return nil, proxy.Failure("invalid_request", 400)
			}
		}
	}
	if err := parameters(fields, request); err != nil {
		return nil, err
	}
	input := fields["messages"]
	if request.Protocol == "responses" {
		input = fields["input"]
		// 文档未确认服务端历史保存；拒绝依赖服务端状态的请求，保留显式完整历史。
		for _, name := range []string{"store", "background"} {
			if raw, exists := fields[name]; exists {
				var value *bool
				if json.Unmarshal(raw, &value) != nil || value == nil || *value {
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
		if json.Unmarshal(raw, &tools) != nil || tools == nil || len(tools) > 128 {
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
			if !validToolName(name) {
				return nil, proxy.Failure("invalid_request", 400)
			}
		}
	}
	if request.Protocol == "chat" && request.Stream {
		options := map[string]json.RawMessage{}
		if raw, ok := fields["stream_options"]; ok && (json.Unmarshal(raw, &options) != nil || options == nil) {
			return nil, proxy.Failure("invalid_request", 400)
		}
		options["include_usage"] = json.RawMessage("true")
		fields["stream_options"], err = json.Marshal(options)
		if err != nil {
			return nil, proxy.Failure("invalid_request", 400)
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
