package agentprotocol

import "encoding/json"

// ToChat 保留工具调用 ID 和结果关联；不能转换的多模态输入显式失败。
func ToChat(body []byte, protocol string) ([]byte, map[string]string, error) {
	var source Object
	if json.Unmarshal(body, &source) != nil {
		return nil, nil, ErrUnsupported
	}
	if protocol == "chat" {
		return body, map[string]string{}, nil
	}
	messages := []any{}
	instruction := source["instructions"]
	if protocol == "messages" {
		instruction = source["system"]
	}
	if instruction != nil {
		s, e := content(instruction)
		if e != nil {
			return nil, nil, e
		}
		messages = append(messages, Object{"role": "system", "content": s})
	}
	if protocol == "responses" {
		var e error
		messages, e = responseInput(source["input"], messages)
		if e != nil {
			return nil, nil, e
		}
	} else {
		for _, v := range array(source["messages"]) {
			m := object(v)
			role := text(m["role"])
			if role == "system" {
				// Claude Code 2.1.295 会附加系统提醒消息，保留其位置和文本，不接受系统工具块。
				s, e := content(m["content"])
				if e != nil {
					return nil, nil, e
				}
				messages = append(messages, Object{"role": role, "content": s})
				continue
			}
			if role != "assistant" && role != "user" {
				return nil, nil, ErrUnsupported
			}
			if _, ok := m["content"].(string); ok {
				messages = append(messages, Object{"role": role, "content": m["content"]})
				continue
			}
			bodyText := ""
			calls := []any{}
			for _, entry := range array(m["content"]) {
				part := object(entry)
				switch text(part["type"]) {
				case "text":
					bodyText += text(part["text"])
				case "tool_use":
					calls = append(calls, Object{"id": part["id"], "type": "function", "function": Object{"name": part["name"], "arguments": string(encode(part["input"]))}})
				case "tool_result":
					s, e := content(part["content"])
					if e != nil {
						return nil, nil, e
					}
					messages = append(messages, Object{"role": "tool", "tool_call_id": part["tool_use_id"], "content": s})
				case "thinking", "redacted_thinking": // 其他提供商不能验证 Anthropic 的推理签名，保留可见对话和工具链。
				case "image", "input_image", "image_url":
					return nil, nil, ErrUnsupportedImage
				default:
					return nil, nil, ErrUnsupported
				}
			}
			if bodyText != "" || len(calls) > 0 {
				m := Object{"role": role, "content": bodyText}
				if len(calls) > 0 {
					m["tool_calls"] = calls
				}
				messages = append(messages, m)
			}
		}
	}
	tools, kinds, err := chatTools(array(source["tools"]), protocol)
	if err != nil {
		return nil, nil, err
	}
	// 跨协议无法回传 DeepSeek 的原生 reasoning_content，关闭其思考以保证工具结果续接。
	result := Object{"model": source["model"], "messages": messages, "stream": false, "thinking": Object{"type": "disabled"}}
	if len(tools) > 0 {
		result["tools"] = tools
	}
	for _, key := range []string{"temperature", "top_p"} {
		if v, ok := source[key]; ok {
			result[key] = v
		}
	}
	if v, ok := source["max_tokens"]; ok {
		result["max_tokens"] = v
	}
	if v, ok := source["max_output_tokens"]; ok {
		result["max_tokens"] = v
	}
	if v, ok := source["tool_choice"]; ok {
		switch v {
		case "auto", "none", "required":
			result["tool_choice"] = v
		default:
			t := object(v)
			switch text(t["type"]) {
			case "auto":
				result["tool_choice"] = "auto"
			case "any":
				result["tool_choice"] = "required"
			case "tool", "function":
				result["tool_choice"] = Object{"type": "function", "function": Object{"name": t["name"]}}
			default:
				return nil, nil, ErrUnsupported
			}
		}
	}
	return encode(result), kinds, nil
}
func responseInput(v any, messages []any) ([]any, error) {
	if s, ok := v.(string); ok {
		return append(messages, Object{"role": "user", "content": s}), nil
	}
	for _, v := range array(v) {
		item := object(v)
		switch text(item["type"]) {
		case "message", "":
			role := text(item["role"])
			if role == "developer" {
				role = "system"
			}
			if role != "user" && role != "assistant" && role != "system" {
				return nil, ErrUnsupported
			}
			s, e := content(item["content"])
			if e != nil {
				return nil, e
			}
			messages = append(messages, Object{"role": role, "content": s})
		case "function_call":
			messages = append(messages, Object{"role": "assistant", "content": nil, "tool_calls": []any{Object{"id": item["call_id"], "type": "function", "function": Object{"name": toolName(item), "arguments": item["arguments"]}}}})
		case "custom_tool_call":
			messages = append(messages, Object{"role": "assistant", "content": nil, "tool_calls": []any{Object{"id": item["call_id"], "type": "function", "function": Object{"name": toolName(item), "arguments": string(encode(Object{"input": item["input"]}))}}}})
		case "function_call_output", "custom_tool_call_output":
			s, e := content(item["output"])
			if e != nil {
				return nil, e
			}
			messages = append(messages, Object{"role": "tool", "tool_call_id": item["call_id"], "content": s})
		case "reasoning":
		default:
			return nil, ErrUnsupported
		}
	}
	return messages, nil
}
