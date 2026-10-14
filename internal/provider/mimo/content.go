package mimo

import (
	"encoding/json"

	"omniproxy/internal/core/proxy"
)

func textInputs(raw json.RawMessage, responses bool) bool {
	var text string
	if responses && string(raw) != "null" && json.Unmarshal(raw, &text) == nil {
		return true
	}
	var items []map[string]json.RawMessage
	if json.Unmarshal(raw, &items) != nil || len(items) == 0 || len(items) > 512 {
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
			case "reasoning", "function_call":
				continue
			case "function_call_output":
				if !textContent(item["output"], 0) {
					return false
				}
				continue
			case "", "message":
			default:
				return false
			}
		}
		var role string
		if json.Unmarshal(item["role"], &role) != nil || role != "user" && role != "assistant" && role != "system" && !(responses && role == "developer") {
			return false
		}
		if !textContent(item["content"], 0) {
			return false
		}
	}
	return true
}

func textContent(raw json.RawMessage, depth int) bool {
	if depth > 16 {
		return false
	}
	var text string
	if string(raw) != "null" && json.Unmarshal(raw, &text) == nil {
		return true
	}
	var blocks []map[string]json.RawMessage
	if json.Unmarshal(raw, &blocks) != nil || blocks == nil || len(blocks) > 512 {
		return false
	}
	for _, block := range blocks {
		var kind string
		if json.Unmarshal(block["type"], &kind) != nil {
			return false
		}
		switch kind {
		case "text", "input_text", "output_text":
			if string(block["text"]) == "null" || json.Unmarshal(block["text"], &text) != nil {
				return false
			}
		case "thinking":
			if string(block["thinking"]) == "null" || json.Unmarshal(block["thinking"], &text) != nil {
				return false
			}
		case "redacted_thinking":
			if string(block["data"]) == "null" || json.Unmarshal(block["data"], &text) != nil {
				return false
			}
		case "tool_result":
			if !textContent(block["content"], depth+1) {
				return false
			}
		case "tool_use":
			var input map[string]json.RawMessage
			if json.Unmarshal(block["input"], &input) != nil || input == nil {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func validateTools(raw json.RawMessage, protocol string) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var tools []map[string]json.RawMessage
	if json.Unmarshal(raw, &tools) != nil || tools == nil || len(tools) > 128 {
		return proxy.Failure("invalid_request", 400)
	}
	for _, tool := range tools {
		var kind string
		if value, exists := tool["type"]; exists && json.Unmarshal(value, &kind) != nil {
			return proxy.Failure("invalid_request", 400)
		}
		if protocol == "messages" {
			if kind != "" && kind != "custom" {
				return proxy.Failure("unsupported_content", 400)
			}
		} else {
			if kind != "function" {
				return proxy.Failure("unsupported_content", 400)
			}
			if protocol == "chat" {
				var function map[string]json.RawMessage
				if json.Unmarshal(tool["function"], &function) != nil || function == nil {
					return proxy.Failure("invalid_request", 400)
				}
			}
		}
	}
	return nil
}
