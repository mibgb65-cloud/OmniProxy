package workbuddyai

import (
	"encoding/json"
	"strings"
)

const (
	// defaultSystemPrompt 是官方客户端的默认系统提示词；上游要求首条消息必须是 system，
	// 实际观测：只有 user 消息时返回 400 code 11128 "first message is not system prompt"。
	defaultSystemPrompt = "You are a helpful assistant."
	// chatChannelRejected 是上游按渠道规则拒绝系统提示词的业务码（首条非 system 等）。
	chatChannelRejected = 11128
	// instructionsOpen/Close 是 11128 重试形态里包裹系统提示词的标签（照社区插件）。
	instructionsOpen  = "<instructions>\n"
	instructionsClose = "\n</instructions>"
)

// prepareChatBody 让出站正文满足上游的会话形态：
//
//   - `developer` 角色归一成 `system`（Chat 的新角色名上游不认）；
//   - `max_completion_tokens` 在缺 `max_tokens` 时改名（官方客户端只发后者）；
//   - 首条消息不是 `system` 时，在最前面补一条默认 system。
//
// 解析失败、没有 messages 或本来就合规时原样返回：不猜测未知结构，也不改写无关字段。
func prepareChatBody(body []byte) []byte {
	top, messages, ok := splitChat(body)
	if !ok {
		return body
	}
	prepared := make([]json.RawMessage, 0, len(messages)+1)
	changed := false
	for _, raw := range messages {
		updated, normalized := withRole(raw, "system")
		if normalized {
			raw, changed = updated, true
		}
		prepared = append(prepared, raw)
	}
	if messageRole(prepared[0]) != "system" {
		prepared = append([]json.RawMessage{jsonMessage("system", defaultSystemPrompt)}, prepared...)
		changed = true
	}
	if raw, found := top["max_completion_tokens"]; found {
		if _, exists := top["max_tokens"]; !exists {
			top["max_tokens"], changed = raw, true
			delete(top, "max_completion_tokens")
		}
	}
	if !changed {
		return body
	}
	return joinChat(top, prepared)
}

// systemAsUserBody 是 11128 重试形态：把开头的系统提示词读成用户消息（`<instructions>` 包裹），
// 开头换成默认 system；聊天中其它 system 消息就地改成 user。
//
// 上游的渠道校验只读 system 与 assistant 消息，因此这个形态能让被拒的系统提示词通过。
// 没有可移动内容时返回 nil（调用方不重试）。
func systemAsUserBody(body []byte) []byte {
	top, messages, ok := splitChat(body)
	if !ok {
		return nil
	}
	lead := 0
	var system []string
	for ; lead < len(messages) && messageRole(messages[lead]) == "system"; lead++ {
		system = append(system, messageText(messages[lead]))
	}
	said := strings.TrimSpace(strings.Join(system, "\n\n"))
	moved := said != "" && said != defaultSystemPrompt
	rest := make([]json.RawMessage, 0, len(messages)-lead)
	changed := lead > 0
	for _, raw := range messages[lead:] {
		if updated, normalized := withRole(raw, "user"); normalized {
			raw = updated
		}
		rest = append(rest, raw)
	}
	if !moved && !changed {
		return nil
	}
	if moved {
		text := instructionsOpen + said + instructionsClose
		index := -1
		for position, raw := range rest {
			if messageRole(raw) == "user" {
				index = position
				break
			}
		}
		switch {
		case index < 0:
			rest = append([]json.RawMessage{jsonMessage("user", text)}, rest...)
		default:
			rest[index] = prependUserText(rest[index], text)
		}
	}
	prepared := append([]json.RawMessage{jsonMessage("system", defaultSystemPrompt)}, rest...)
	return joinChat(top, prepared)
}

// splitChat 取出顶层字段与 messages；结构不符时 ok 为 false。
func splitChat(body []byte) (map[string]json.RawMessage, []json.RawMessage, bool) {
	var top map[string]json.RawMessage
	if json.Unmarshal(body, &top) != nil {
		return nil, nil, false
	}
	raw, found := top["messages"]
	if !found {
		return nil, nil, false
	}
	var messages []json.RawMessage
	if json.Unmarshal(raw, &messages) != nil || len(messages) == 0 {
		return nil, nil, false
	}
	return top, messages, true
}

// joinChat 把新的 messages 写回顶层字段并编码。
func joinChat(top map[string]json.RawMessage, messages []json.RawMessage) []byte {
	encoded, err := json.Marshal(messages)
	if err != nil {
		return nil
	}
	top["messages"] = encoded
	body, err := json.Marshal(top)
	if err != nil {
		return nil
	}
	return body
}

// withRole 把消息的角色改成 role；角色本来就不同且需要改时 normalized 为 true。
func withRole(raw json.RawMessage, role string) (json.RawMessage, bool) {
	switch role {
	case "system":
		if messageRole(raw) != "developer" {
			return raw, false
		}
	case "user":
		if current := messageRole(raw); current != "system" && current != "developer" {
			return raw, false
		}
	}
	var message map[string]json.RawMessage
	if json.Unmarshal(raw, &message) != nil {
		return raw, false
	}
	encoded, err := json.Marshal(role)
	if err != nil {
		return raw, false
	}
	message["role"] = encoded
	updated, err := json.Marshal(message)
	if err != nil {
		return raw, false
	}
	return updated, true
}

// prependUserText 把 text 放到用户消息内容前：字符串直接连接，数组则插入一个 text 片段。
func prependUserText(raw json.RawMessage, text string) json.RawMessage {
	var message map[string]json.RawMessage
	if json.Unmarshal(raw, &message) != nil {
		return raw
	}
	var plain string
	if json.Unmarshal(message["content"], &plain) == nil {
		joined, err := json.Marshal(text + "\n\n" + plain)
		if err != nil {
			return raw
		}
		message["content"] = joined
	} else {
		var parts []json.RawMessage
		if json.Unmarshal(message["content"], &parts) == nil {
			prefix, err := json.Marshal(map[string]string{"type": "text", "text": text})
			if err != nil {
				return raw
			}
			message["content"] = mustMarshal(append([]json.RawMessage{prefix}, parts...))
		} else {
			message["content"] = mustMarshal(text)
		}
	}
	updated, err := json.Marshal(message)
	if err != nil {
		return raw
	}
	return updated
}

// messageRole 读取消息角色；结构不符时返回空串。
func messageRole(raw json.RawMessage) string {
	var message struct {
		Role string `json:"role"`
	}
	if json.Unmarshal(raw, &message) != nil {
		return ""
	}
	return message.Role
}

// messageText 取消息内容的纯文本：字符串原样，数组取各 text 片段，其它为空。
func messageText(raw json.RawMessage) string {
	var message struct {
		Content json.RawMessage `json:"content"`
	}
	if json.Unmarshal(raw, &message) != nil {
		return ""
	}
	var plain string
	if json.Unmarshal(message.Content, &plain) == nil {
		return plain
	}
	var parts []struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(message.Content, &parts) != nil {
		return ""
	}
	texts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part.Text != "" {
			texts = append(texts, part.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// jsonMessage 构造一条最小消息。
func jsonMessage(role, content string) json.RawMessage {
	encoded, err := json.Marshal(map[string]string{"role": role, "content": content})
	if err != nil {
		return json.RawMessage(`{"role":"` + role + `","content":""}`)
	}
	return encoded
}

// mustMarshal 编码为 RawMessage；失败时退回 null，调用方会按原样转发。
func mustMarshal(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage("null")
	}
	return encoded
}
