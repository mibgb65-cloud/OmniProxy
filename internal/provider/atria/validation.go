package atria

import (
	"encoding/json"

	"omniproxy/internal/core/proxy"
)

func validatedBody(request proxy.Request, path string) ([]byte, error) {
	parsed, err := proxy.ParseProtocolRequest(request.Body, "/v1"+path)
	if err != nil {
		return nil, err
	}
	if parsed.Model != ModelID || request.Model != ModelID || parsed.Stream != request.Stream {
		return nil, proxy.Failure("unsupported_model", 400)
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(request.Body, &fields) != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	for _, name := range []string{"max_tokens", "max_completion_tokens", "max_output_tokens"} {
		if raw, exists := fields[name]; exists {
			var limit int
			if json.Unmarshal(raw, &limit) != nil || limit < 1 || limit > maxOutputTokens {
				return nil, proxy.Failure("invalid_request", 400)
			}
		}
	}
	if request.Protocol == "messages" {
		if raw, exists := fields["system"]; exists && !textContent(raw) {
			return nil, proxy.Failure("unsupported_content", 400)
		}
		if !textMessages(fields["messages"], false) {
			return nil, proxy.Failure("unsupported_content", 400)
		}
	}
	if request.Protocol == "responses" && !textMessages(fields["input"], true) {
		return nil, proxy.Failure("unsupported_content", 400)
	}
	// Chat usage 默认可能缺失；使用标准 include_usage 请求最终用量快照。
	if request.Protocol == "chat" && request.Stream {
		options := map[string]json.RawMessage{}
		if raw, exists := fields["stream_options"]; exists && string(raw) != "null" {
			if json.Unmarshal(raw, &options) != nil || options == nil {
				return nil, proxy.Failure("invalid_request", 400)
			}
		}
		options["include_usage"] = json.RawMessage("true")
		encoded, err := json.Marshal(options)
		if err != nil {
			return nil, proxy.Failure("invalid_request", 400)
		}
		fields["stream_options"] = encoded
		body, err := json.Marshal(fields)
		if err != nil || len(body) > proxy.MaxRequestBytes {
			return nil, proxy.Failure("request_too_large", 413)
		}
		return body, nil
	}
	return request.Body, nil
}

func textMessages(raw json.RawMessage, responses bool) bool {
	var text string
	if responses && json.Unmarshal(raw, &text) == nil && string(raw) != "null" {
		return true
	}
	var items []map[string]json.RawMessage
	if json.Unmarshal(raw, &items) != nil || items == nil || len(items) == 0 || len(items) > 512 {
		return false
	}
	for _, item := range items {
		if item == nil {
			return false
		}
		if responses {
			var kind string
			if value, exists := item["type"]; exists && json.Unmarshal(value, &kind) != nil {
				return false
			}
			switch kind {
			case "function_call", "custom_tool_call", "reasoning", "item_reference":
				continue
			case "function_call_output", "custom_tool_call_output":
				if !textContent(item["output"]) {
					return false
				}
				continue
			case "", "message":
			default:
				return false
			}
		}
		if !textContent(item["content"]) {
			return false
		}
	}
	return true
}

func textContent(raw json.RawMessage) bool {
	var text string
	if json.Unmarshal(raw, &text) == nil && string(raw) != "null" {
		return true
	}
	var blocks []struct {
		Type    string          `json:"type"`
		Text    *string         `json:"text"`
		Content json.RawMessage `json:"content"`
	}
	if json.Unmarshal(raw, &blocks) != nil || blocks == nil || len(blocks) > 512 {
		return false
	}
	for _, block := range blocks {
		switch block.Type {
		case "text", "input_text", "output_text":
			if block.Text == nil {
				return false
			}
		case "tool_result":
			if !textContent(block.Content) {
				return false
			}
		case "tool_use", "thinking", "redacted_thinking":
			// 工具参数允许业务 JSON；不递归扫描参数中的同名 type/content 字段。
		default:
			return false
		}
	}
	return true
}
