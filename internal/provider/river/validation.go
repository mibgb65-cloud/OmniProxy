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
				if !supportedContent(item["output"]) {
					return false
				}
				continue
			case "", "message":
			default:
				return false
			}
		}
		if !supportedContent(item["content"]) {
			return false
		}
	}
	return true
}

// supportedContent 允许官方已公布的 input_image 块，同时继续拒绝 file／audio 等
// 适配器尚未接入的内容类型，避免把无法验证的输入当作已支持透传。
func supportedContent(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var text string
	if json.Unmarshal(raw, &text) == nil && string(raw) != "null" {
		return true
	}
	var blocks []json.RawMessage
	if json.Unmarshal(raw, &blocks) != nil || blocks == nil || len(blocks) > 512 {
		return false
	}
	for _, entry := range blocks {
		var block struct {
			Type string  `json:"type"`
			Text *string `json:"text"`
		}
		if json.Unmarshal(entry, &block) != nil {
			return false
		}
		switch block.Type {
		case "input_image":
			if !imageBlockValid(entry) {
				return false
			}
		case "text", "input_text", "output_text":
			// 上游按字符串接收文本；缺少 text 字段说明块已损坏。
			if block.Text == nil {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// imageBlockValid 只接受官方公布的 image_url 字符串或对象形式，并保留 detail。
// 未知字段（例如伪造的额外键）不通过，避免把未核对的结构当图片转发。
func imageBlockValid(entry json.RawMessage) bool {
	var block map[string]json.RawMessage
	if json.Unmarshal(entry, &block) != nil || block == nil {
		return false
	}
	for name := range block {
		switch name {
		case "type", "image_url", "detail":
		default:
			return false
		}
	}
	raw, exists := block["image_url"]
	if !exists {
		return false
	}
	detail := ""
	var url string
	if json.Unmarshal(raw, &url) == nil {
		if url == "" || len(url) > 8192 || !validURLChars(url) {
			return false
		}
	} else {
		var object struct {
			URL    string `json:"url"`
			Detail string `json:"detail"`
		}
		if json.Unmarshal(raw, &object) != nil || object.URL == "" || len(object.URL) > 8192 || !validURLChars(object.URL) {
			return false
		}
		detail = object.Detail
	}
	if rawDetail, ok := block["detail"]; ok {
		// 同时出现的对象内 detail 与块级 detail 必须一致，避免歧义。
		var level string
		if json.Unmarshal(rawDetail, &level) != nil || (detail != "" && level != detail) {
			return false
		}
		detail = level
	}
	switch detail {
	case "", "low", "high", "auto", "original":
		return true
	}
	return false
}

// validURLChars 排除控制字符，避免把任意正文伪装成图片地址。
func validURLChars(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < 0x20 {
			return false
		}
	}
	return true
}
