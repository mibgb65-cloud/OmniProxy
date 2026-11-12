package proxy

import "encoding/json"

// antigravityDroppedChatFields 是 Antigravity 桌面端会带、但 Google 订阅的
// Chat 适配器明确不接受的字段。只在这个客户端上剥离，其他客户端的请求语义
// 保持不变。
var antigravityDroppedChatFields = []string{"reasoning_effort", "service_tier", "metadata", "store"}

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
