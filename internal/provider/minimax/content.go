package minimax

import "encoding/json"

func validToolName(name string) bool {
	if len(name) == 0 || len(name) > 128 {
		return false
	}
	for _, r := range name {
		if r != '_' && r != '-' && !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func textInput(raw json.RawMessage, depth int) bool {
	if depth > 12 || len(raw) == 0 || string(raw) == "null" {
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
	case "", "message", "text", "input_text", "output_text", "thinking", "redacted_thinking", "reasoning", "reasoning_text", "summary_text", "tool_use", "tool_result", "function_call", "function_call_output":
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
	if kind == "function_call_output" {
		return textInput(block["output"], depth+1)
	}
	return true
}
