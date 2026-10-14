package kimi

import (
	"encoding/json"
	"omniproxy/internal/core/proxy"
)

func textInput(raw json.RawMessage, depth int) bool {
	if depth > 12 || len(raw) == 0 || string(raw) == "null" {
		return false
	}
	var value string
	if json.Unmarshal(raw, &value) == nil {
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
	case "", "message", "text", "input_text", "output_text", "thinking", "redacted_thinking", "reasoning", "reasoning_text", "summary_text", "tool_use", "tool_result", "function_call", "function_call_output", "custom_tool_call", "custom_tool_call_output":
	default:
		return false
	}
	if content, exists := block["content"]; exists && string(content) != "null" {
		return textInput(content, depth+1)
	}
	if kind == "function_call_output" || kind == "custom_tool_call_output" {
		return textInput(block["output"], depth+1)
	}
	return true
}

func validateTools(raw json.RawMessage, protocol string, depth int) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	if depth > 1 {
		return proxy.Failure("unsupported_content", 400)
	}
	var tools []map[string]json.RawMessage
	if json.Unmarshal(raw, &tools) != nil || tools == nil || len(tools) > 128 {
		return proxy.Failure("invalid_request", 400)
	}
	for _, tool := range tools {
		var kind string
		if field, exists := tool["type"]; exists && json.Unmarshal(field, &kind) != nil {
			return proxy.Failure("invalid_request", 400)
		}
		if protocol == "messages" {
			if kind != "" && kind != "custom" {
				return proxy.Failure("unsupported_content", 400)
			}
			continue
		}
		switch kind {
		case "function":
		case "namespace":
			if protocol != "responses" {
				return proxy.Failure("unsupported_content", 400)
			}
			if err := validateTools(tool["tools"], protocol, depth+1); err != nil {
				return err
			}
		case "custom":
			var name string
			var format struct {
				Type       string `json:"type"`
				Syntax     string `json:"syntax"`
				Definition string `json:"definition"`
			}
			if protocol != "responses" || json.Unmarshal(tool["name"], &name) != nil || name != "apply_patch" || json.Unmarshal(tool["format"], &format) != nil || format.Type != "grammar" || format.Syntax != "lark" || len(format.Definition) == 0 || len(format.Definition) > 65536 {
				return proxy.Failure("unsupported_content", 400)
			}
		case "web_search":
			if protocol != "responses" || depth > 0 {
				return proxy.Failure("unsupported_content", 400)
			}
		default:
			return proxy.Failure("unsupported_content", 400)
		}
	}
	return nil
}
