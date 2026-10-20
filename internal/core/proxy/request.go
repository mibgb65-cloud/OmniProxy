package proxy

import (
	"encoding/json"
	"math"
	"omniproxy/internal/core/routing"
)

// ParseRequest 验证路由字段、文本消息和常用参数；其他 JSON 字段受整体大小与嵌套限制并由上游校验。
func ParseRequest(body []byte) (Request, error) {
	return parseChatRequest(body, MaxRequestBytes, 1<<20)
}

// ParseConvertedChatRequest 保留可信 Messages 转换的原始容量，不放宽外部 Chat 接口。
func ParseConvertedChatRequest(body []byte) (Request, error) {
	return parseChatRequest(body, MaxMessagesRequestBytes, MaxMessagesRequestBytes)
}

func parseChatRequest(body []byte, maxBodyBytes, maxMessageBytes int) (Request, error) {
	if err := validateJSONSize(body, maxBodyBytes); err != nil {
		return Request{}, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil || fields == nil {
		return Request{}, Failure("invalid_request", 400)
	}
	var request Request
	if err := json.Unmarshal(fields["model"], &request.Model); err != nil || !routing.ValidModel(request.Model) {
		return Request{}, Failure("invalid_model", 400)
	}
	if raw, exists := fields["stream"]; exists && string(raw) != "null" {
		if err := json.Unmarshal(raw, &request.Stream); err != nil {
			return Request{}, Failure("invalid_request", 400)
		}
	}
	if normalized, changed, err := normalizeChatText(fields["messages"], maxMessageBytes); err != nil {
		return Request{}, err
	} else if changed {
		fields["messages"] = normalized
		normalizedBody, err := json.Marshal(fields)
		if err != nil || len(normalizedBody) > maxBodyBytes {
			return Request{}, Failure("invalid_messages", 400)
		}
		body = normalizedBody
	}
	if err := validateMessages(fields["messages"], maxMessageBytes); err != nil {
		return Request{}, err
	}
	if err := validateParameters(fields); err != nil {
		return Request{}, err
	}
	request.Body = body
	return request, nil
}

func validateMessages(raw json.RawMessage, maxMessageBytes int) error {
	var messages []struct {
		Role       string            `json:"role"`
		Content    json.RawMessage   `json:"content"`
		ToolCalls  []json.RawMessage `json:"tool_calls"`
		ToolCallID string            `json:"tool_call_id"`
		Name       string            `json:"name"`
	}
	if err := json.Unmarshal(raw, &messages); err != nil || len(messages) == 0 || len(messages) > MaxConversationMessages {
		return Failure("invalid_messages", 400)
	}
	for _, message := range messages {
		if message.Role != "system" && message.Role != "user" && message.Role != "assistant" && message.Role != "tool" {
			return Failure("invalid_messages", 400)
		}
		if len(message.Name) > 128 || len(message.ToolCallID) > 256 || len(message.ToolCalls) > 128 {
			return Failure("invalid_messages", 400)
		}
		if message.Role == "tool" && message.ToolCallID == "" {
			return Failure("invalid_messages", 400)
		}
		if message.Role == "assistant" && (len(message.Content) == 0 || string(message.Content) == "null") && len(message.ToolCalls) > 0 {
			continue
		}
		var content string
		if len(message.Content) == 0 || string(message.Content) == "null" || json.Unmarshal(message.Content, &content) != nil || len(content) > maxMessageBytes {
			return Failure("invalid_messages", 400)
		}
	}
	return nil
}

func validateParameters(fields map[string]json.RawMessage) error {
	for name, bounds := range map[string][2]float64{"temperature": {0, 2}, "top_p": {0, 1}, "max_tokens": {1, 393216}} {
		raw, exists := fields[name]
		if !exists || string(raw) == "null" {
			continue
		}
		var value float64
		if json.Unmarshal(raw, &value) != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < bounds[0] || value > bounds[1] || name == "max_tokens" && math.Trunc(value) != value || name == "top_p" && value == 0 {
			return Failure("invalid_request", 400)
		}
	}
	return nil
}
