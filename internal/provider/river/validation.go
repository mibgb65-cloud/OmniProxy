package river

import (
	"encoding/json"

	"omniproxy/internal/core/proxy"
)

func validatedBody(request proxy.Request, path string) ([]byte, error) {
	chatLimit := proxy.MaxRequestBytes
	var parsed proxy.Request
	var err error
	if request.Protocol == "chat" && request.ConvertedFromMessages {
		// 普通 Chat 仍为 2 MiB；只有服务端可信转换沿用 Messages 的 16 MiB 容量。
		chatLimit = proxy.MaxMessagesRequestBytes
		parsed, err = proxy.ParseConvertedChatRequest(request.Body)
	} else {
		parsed, err = proxy.ParseProtocolRequest(request.Body, "/v1"+path)
	}
	if err != nil {
		return nil, err
	}
	if !SupportsModel(parsed.Model) || request.Model != parsed.Model || parsed.Stream != request.Stream {
		return nil, proxy.Failure("unsupported_model", 400)
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(request.Body, &fields) != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	for _, name := range []string{"max_tokens", "max_completion_tokens", "max_output_tokens"} {
		if raw, exists := fields[name]; exists {
			var limit int
			if json.Unmarshal(raw, &limit) != nil || limit < 1 || int64(limit) > *ModelMetadata(request.Model).OutputTokenLimit {
				return nil, proxy.Failure("invalid_request", 400)
			}
		}
	}
	if request.Protocol == "responses" && !textMessages(fields["input"]) {
		return nil, proxy.Failure("unsupported_content", 400)
	}
	if err := validateTools(fields["tools"], request.Protocol); err != nil {
		return nil, err
	}
	if err := validateParameters(fields, request.Model, request.Protocol); err != nil {
		return nil, err
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
		if err != nil || len(body) > chatLimit {
			return nil, proxy.Failure("request_too_large", 413)
		}
		return body, nil
	}
	return request.Body, nil
}

func textMessages(raw json.RawMessage) bool {
	var text string
	if json.Unmarshal(raw, &text) == nil && string(raw) != "null" {
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
		{
			var kind string
			if value, exists := item["type"]; exists && json.Unmarshal(value, &kind) != nil {
				return false
			}
			switch kind {
			case "function_call", "reasoning":
				continue
			case "function_call_output":
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
		Type string  `json:"type"`
		Text *string `json:"text"`
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
		default:
			return false
		}
	}
	return true
}
