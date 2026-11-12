package proxy

import "encoding/json"

// antigravityDroppedChatFields 是 Antigravity 桌面端会带、但 Google 订阅的
// Chat 适配器明确不接受的字段。只在这个客户端上剥离，其他客户端的请求语义
// 保持不变。
//
// reasoning_effort 不在这里剥离：它是选择上游档位变体（…-high／…-medium）的唯一依据，
// 必须保留到路由完成；转发前由 StripClientEffort 移除。
var antigravityDroppedChatFields = []string{"service_tier", "metadata", "store"}

// NormalizeClientBody 按客户端规整已读取的正文；只处理有明确依据的字段。
func NormalizeClientBody(client string, body []byte) ([]byte, error) {
	if client != "antigravity" {
		return body, nil
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil || fields == nil {
		return body, nil
	}
	changed := false
	for _, key := range antigravityDroppedChatFields {
		if _, exists := fields[key]; exists {
			delete(fields, key)
			changed = true
		}
	}
	if !changed {
		return body, nil
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return nil, Failure("invalid_request", 400)
	}
	return encoded, nil
}

// StripClientEffort 在路由完成后移除客户端自带的推理档位字段。
//
// 档位已经在选型时用掉：上游只接受具体型号（…-high），Google 订阅适配器的生成参数
// 又是白名单校验，残留的 reasoning_effort／output_config.effort 会被直接拒绝。
func StripClientEffort(client, protocol string, body []byte) ([]byte, error) {
	if client != "antigravity" {
		return body, nil
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil || fields == nil {
		return body, nil
	}
	changed := false
	switch protocol {
	case "messages", "responses":
		key := "output_config"
		if protocol == "responses" {
			key = "reasoning"
		}
		raw, exists := fields[key]
		if exists && string(raw) != "null" {
			var container map[string]json.RawMessage
			if json.Unmarshal(raw, &container) != nil {
				return nil, Failure("invalid_request", 400)
			}
			if _, ok := container["effort"]; ok {
				delete(container, "effort")
				changed = true
				if len(container) == 0 {
					delete(fields, key)
				} else {
					encoded, err := json.Marshal(container)
					if err != nil {
						return nil, Failure("invalid_request", 400)
					}
					fields[key] = encoded
				}
			}
		}
	default:
		if _, exists := fields["reasoning_effort"]; exists {
			delete(fields, "reasoning_effort")
			changed = true
		}
	}
	if !changed {
		return body, nil
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return nil, Failure("invalid_request", 400)
	}
	return encoded, nil
}
