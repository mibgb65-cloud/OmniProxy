package proxy

import (
	"bytes"
	"encoding/json"
	"strings"
)

// 编码客户端可将一条文本拆成多个 text 部分；上游文本 API 仍按字符串接收。
// 这里只归一化有界纯文本，不把图片或未知部分当作文本静默丢弃。
func normalizeChatText(raw json.RawMessage) (json.RawMessage, bool, error) {
	var messages []map[string]json.RawMessage
	if json.Unmarshal(raw, &messages) != nil || len(messages) == 0 || len(messages) > 512 {
		return nil, false, Failure("invalid_messages", 400)
	}
	changed := false
	for _, message := range messages {
		content := bytes.TrimSpace(message["content"])
		if len(content) == 0 || content[0] != '[' {
			continue
		}
		var parts []struct {
			Type string          `json:"type"`
			Text json.RawMessage `json:"text"`
		}
		if json.Unmarshal(content, &parts) != nil || len(parts) == 0 || len(parts) > 128 {
			return nil, false, Failure("invalid_messages", 400)
		}
		texts := make([]string, 0, len(parts))
		length := 0
		for _, part := range parts {
			var text string
			if part.Type != "text" || len(part.Text) == 0 || bytes.Equal(part.Text, []byte("null")) || json.Unmarshal(part.Text, &text) != nil {
				return nil, false, Failure("invalid_messages", 400)
			}
			length += len(text)
			if len(texts) > 0 {
				length++
			}
			if length > 1<<20 {
				return nil, false, Failure("invalid_messages", 400)
			}
			texts = append(texts, text)
		}
		message["content"], _ = json.Marshal(strings.Join(texts, "\n"))
		changed = true
	}
	if !changed {
		return raw, false, nil
	}
	encoded, err := json.Marshal(messages)
	return encoded, true, err
}
