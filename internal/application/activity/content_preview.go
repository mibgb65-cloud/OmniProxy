package activity

import (
	"encoding/json"
	"strings"
)

const previewLimit = 320
const previewTexts = 4

// contentPreview 从请求正文中提取最近的非 system 文本，供记录列表快速浏览。
// 解析失败时回退为折叠空白后的原文前缀；不返回完整正文。
func contentPreview(content string) string {
	texts := extractTexts(content)
	preview := ""
	if len(texts) == 0 {
		preview = content
	} else {
		if len(texts) > previewTexts {
			texts = texts[len(texts)-previewTexts:]
		}
		preview = strings.Join(texts, " · ")
	}
	preview = strings.Join(strings.Fields(preview), " ")
	return truncateRunes(preview, previewLimit)
}

func extractTexts(content string) []string {
	var root map[string]json.RawMessage
	if json.Unmarshal([]byte(content), &root) != nil {
		return nil
	}
	if raw, ok := root["messages"]; ok {
		return messageTexts(raw)
	}
	if raw, ok := root["input"]; ok {
		return inputTexts(raw)
	}
	return nil
}

func messageTexts(raw json.RawMessage) []string {
	var messages []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if json.Unmarshal(raw, &messages) != nil {
		return nil
	}
	texts := make([]string, 0, len(messages))
	for _, message := range messages {
		if isSystemRole(message.Role) {
			continue
		}
		if text := partText(message.Content); text != "" {
			texts = append(texts, text)
		}
	}
	return texts
}

func inputTexts(raw json.RawMessage) []string {
	var direct string
	if json.Unmarshal(raw, &direct) == nil {
		if direct == "" {
			return nil
		}
		return []string{direct}
	}
	var items []json.RawMessage
	if json.Unmarshal(raw, &items) != nil {
		return nil
	}
	texts := make([]string, 0, len(items))
	for _, item := range items {
		var fields struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
			Text    string          `json:"text"`
		}
		if json.Unmarshal(item, &fields) != nil {
			continue
		}
		switch {
		case fields.Text != "":
			texts = append(texts, fields.Text)
		case len(fields.Content) > 0 && !isSystemRole(fields.Role):
			if text := partText(fields.Content); text != "" {
				texts = append(texts, text)
			}
		}
	}
	return texts
}

// partText 支持字符串内容与 [{type,text}] 分段内容。
func partText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	var parts []struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) != nil {
		return ""
	}
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part.Text) != "" {
			values = append(values, part.Text)
		}
	}
	return strings.Join(values, "\n")
}

func isSystemRole(role string) bool {
	switch strings.ToLower(role) {
	case "system", "developer":
		return true
	default:
		return false
	}
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "…"
}
