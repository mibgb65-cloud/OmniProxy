package antigravity

import (
	"encoding/json"

	"omniproxy/internal/core/proxy"
)

const (
	// interruptedToolNote 是客户端没有回答的调用补上的结果：中断的一轮会留下
	// 未应答的调用，Google 对「functionCall 数不等于 functionResponse 数」的
	// 历史拒绝整个请求。
	interruptedToolNote = "[The result of this tool call is unavailable: the turn was interrupted.]"
	// continuedPrompt 是历史以模型回合开头时前置的用户回合。
	continuedPrompt = "[Continuing from previous AI thoughts...]"
	// modelLastPrompt 是历史以模型回合结尾时追加的用户回合：Code Assist 不接受
	// 「Requests ending with a model turn」的请求。
	modelLastPrompt = "Continue."
	// maxRepairedMessages 是修复后消息数的上限，防止合成结果无限放大请求。
	maxRepairedMessages = 2000
)

// repairedMessages 让历史满足 Google 的回合约束，替代此前直接拒绝请求的做法：
// 同一调用的重复结果只保留一条；未回答的调用补一条合成结果；历史不以模型回合
// 开头或结尾。未知的 tool_call_id 仍然拒绝（无法猜测工具名）。
func repairedMessages(messages []chatMessage) ([]chatMessage, error) {
	answered := make(map[string]bool, len(messages))
	declared := make(map[string]bool, len(messages))
	var out []chatMessage
	for _, message := range messages {
		switch message.Role {
		case "assistant":
			for _, call := range message.Calls {
				declared[call.ID] = true
				delete(answered, call.ID)
			}
		case "tool":
			if !declared[message.ToolID] {
				// 先于任何调用出现的工具结果无法关联到工具名。
				return nil, proxy.Failure("invalid_request", 400)
			}
			if answered[message.ToolID] {
				// 同一调用被回答两次：第一次为准，后一条丢弃。
				continue
			}
			answered[message.ToolID] = true
		}
		out = append(out, message)
	}
	for _, message := range messages {
		for _, call := range message.Calls {
			if answered[call.ID] {
				continue
			}
			note, _ := json.Marshal(interruptedToolNote)
			out = append(out, chatMessage{Role: "tool", ToolID: call.ID, Content: note})
			answered[call.ID] = true
		}
	}
	first, last := -1, -1
	for index, message := range out {
		if message.Role == "system" || message.Role == "developer" {
			continue
		}
		if first < 0 {
			first = index
		}
		last = index
	}
	if first < 0 {
		return nil, proxy.Failure("invalid_request", 400)
	}
	if out[first].Role == "assistant" {
		text, _ := json.Marshal(continuedPrompt)
		out = append([]chatMessage{{Role: "user", Content: text}}, out...)
		last++
	}
	if out[last].Role == "assistant" && len(out[last].Calls) == 0 {
		text, _ := json.Marshal(modelLastPrompt)
		out = append(out, chatMessage{Role: "user", Content: text})
	}
	if len(out) > maxRepairedMessages {
		return nil, proxy.Failure("invalid_request", 400)
	}
	return out, nil
}

// mergeTurns 合并相邻同角色回合：Google 把连续的 user 或 model 回合视作同一段
// 历史，官方客户端也是合并后发送。带 functionResponse 的回合保持独立。
func mergeTurns(contents []object) []object {
	out := make([]object, 0, len(contents))
	for _, content := range contents {
		if len(out) == 0 {
			out = append(out, content)
			continue
		}
		previous := out[len(out)-1]
		if previous["role"] != content["role"] || hasResponse(previous) || hasResponse(content) {
			out = append(out, content)
			continue
		}
		left, _ := previous["parts"].([]object)
		right, _ := content["parts"].([]object)
		previous["parts"] = append(left, right...)
	}
	return out
}

// hasResponse 判断一个回合是否包含工具结果。
func hasResponse(content object) bool {
	parts, _ := content["parts"].([]object)
	for _, part := range parts {
		if part["functionResponse"] != nil {
			return true
		}
	}
	return false
}
