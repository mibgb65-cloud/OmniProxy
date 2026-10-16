package antigravity

import (
	"encoding/json"
	"unicode/utf8"

	"omniproxy/internal/core/proxy"
)

func toolDeclarations(tools []chatTool) ([]object, error) {
	output := []object{}
	seen := map[string]bool{}
	for _, tool := range tools {
		var function struct {
			Name        string                     `json:"name"`
			Description string                     `json:"description"`
			Parameters  map[string]json.RawMessage `json:"parameters"`
			Strict      bool                       `json:"strict"`
		}
		if tool.Type != "function" || json.Unmarshal(tool.Function, &function) != nil || !safeString(function.Name, 128) || seen[function.Name] || len(function.Description) > 65536 || !utf8.ValidString(function.Description) || function.Strict {
			return nil, proxy.Failure("unsupported_content", 400)
		}
		seen[function.Name] = true
		declaration := object{"name": function.Name}
		if function.Description != "" {
			declaration["description"] = function.Description
		}
		if function.Parameters != nil {
			declaration["parameters"] = function.Parameters
		}
		output = append(output, declaration)
	}
	return output, nil
}

func generationOptions(fields map[string]json.RawMessage) (object, error) {
	allowed := map[string]bool{}
	for _, key := range []string{"model", "messages", "stream", "stream_options", "tools", "tool_choice", "temperature", "top_p", "max_tokens", "max_completion_tokens", "stop", "n", "user", "parallel_tool_calls"} {
		allowed[key] = true
	}
	for key, value := range fields {
		if !allowed[key] && string(value) != "null" {
			return nil, proxy.Failure("unsupported_content", 400)
		}
	}
	config := object{}
	for _, field := range []struct {
		key, native      string
		minimum, maximum float64
	}{{"temperature", "temperature", 0, 2}, {"top_p", "topP", 0, 1}} {
		if raw, ok := fields[field.key]; ok && string(raw) != "null" {
			var value float64
			if json.Unmarshal(raw, &value) != nil || value < field.minimum || value > field.maximum {
				return nil, proxy.Failure("invalid_request", 400)
			}
			config[field.native] = value
		}
	}
	var tokenLimit int64
	for _, key := range []string{"max_tokens", "max_completion_tokens"} {
		if raw, ok := fields[key]; ok && string(raw) != "null" {
			var value int64
			if json.Unmarshal(raw, &value) != nil || value < 1 || value > 1<<20 || tokenLimit != 0 && value != tokenLimit {
				return nil, proxy.Failure("invalid_request", 400)
			}
			tokenLimit = value
		}
	}
	if tokenLimit != 0 {
		config["maxOutputTokens"] = tokenLimit
	}
	if raw, ok := fields["n"]; ok && string(raw) != "null" {
		var count int
		if json.Unmarshal(raw, &count) != nil || count != 1 {
			return nil, proxy.Failure("unsupported_content", 400)
		}
	}
	if raw, ok := fields["parallel_tool_calls"]; ok && string(raw) != "null" {
		var enabled bool
		if json.Unmarshal(raw, &enabled) != nil || !enabled {
			return nil, proxy.Failure("unsupported_content", 400)
		}
	}
	if raw, ok := fields["stop"]; ok && string(raw) != "null" {
		var sequence string
		var sequences []string
		if json.Unmarshal(raw, &sequence) == nil {
			sequences = []string{sequence}
		} else if json.Unmarshal(raw, &sequences) != nil {
			return nil, proxy.Failure("invalid_request", 400)
		}
		if len(sequences) > 5 {
			return nil, proxy.Failure("invalid_request", 400)
		}
		for _, value := range sequences {
			if len(value) == 0 || len(value) > 256 || !utf8.ValidString(value) {
				return nil, proxy.Failure("invalid_request", 400)
			}
		}
		config["stopSequences"] = sequences
	}
	return config, nil
}
