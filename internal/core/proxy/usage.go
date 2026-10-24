package proxy

import "encoding/json"

const maxReportedTokens = 1_000_000_000_000

func usageCount(value json.RawMessage) (int64, bool) {
	var n int64
	if len(value) == 0 || string(value) == "null" || json.Unmarshal(value, &n) != nil || n < 0 || n > maxReportedTokens {
		return 0, false
	}
	return n, true
}

// terminalStreamEvent 只判定协议终结事件，与是否携带用量无关：
// 终结事件已经交付之后再断开，不能再算成传输失败。
func terminalStreamEvent(body []byte, protocol string) bool {
	var envelope struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return false
	}
	switch protocol {
	case "responses":
		return envelope.Type == "response.completed" || envelope.Type == "response.incomplete"
	case "messages":
		return envelope.Type == "message_stop" || envelope.Type == "message_delta"
	default:
		return false
	}
}

// ParseUsage 只读取上游数字；缓存／推理计数不会被重复加到已包含它们的总数中。
func ParseUsage(body []byte, protocol string, stream bool) UsageSnapshot {
	var envelope struct {
		Type     string          `json:"type"`
		Usage    json.RawMessage `json:"usage"`
		Response struct {
			Usage json.RawMessage `json:"usage"`
		} `json:"response"`
		Message struct {
			Usage json.RawMessage `json:"usage"`
		} `json:"message"`
		Delta struct {
			StopReason json.RawMessage `json:"stop_reason"`
		} `json:"delta"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return UsageSnapshot{}
	}
	raw := envelope.Usage
	terminal := !stream
	if protocol == "responses" && stream {
		if envelope.Type != "response.completed" && envelope.Type != "response.incomplete" {
			return UsageSnapshot{}
		}
		raw = envelope.Response.Usage
		terminal = true
	}
	if protocol == "messages" && stream {
		if envelope.Type == "message_start" {
			raw = envelope.Message.Usage
		}
		terminal = envelope.Type == "message_delta" && len(envelope.Delta.StopReason) > 0 && string(envelope.Delta.StopReason) != "null"
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return UsageSnapshot{}
	}
	input, output := "input_tokens", "output_tokens"
	if protocol == "chat" {
		input, output = "prompt_tokens", "completion_tokens"
		terminal = true
	}
	result := UsageSnapshot{}
	result.Input, result.InputKnown = usageCount(fields[input])
	result.Output, result.OutputKnown = usageCount(fields[output])
	if protocol == "messages" && result.InputKnown {
		for _, key := range []string{"cache_creation_input_tokens", "cache_read_input_tokens"} {
			if value, ok := fields[key]; ok {
				n, valid := usageCount(value)
				if !valid || result.Input > maxReportedTokens-n {
					result.InputKnown = false
					break
				}
				result.Input += n
			}
		}
	}
	result.Complete = terminal && result.InputKnown && result.OutputKnown
	result.CachedInput = cachedInputUsage(fields, protocol, result)
	// Messages delta 只有输出；Trace 会把已报告输入与累计输出合并。
	if protocol == "messages" && stream && envelope.Type == "message_delta" {
		result.Complete = terminal && result.OutputKnown
	}
	return result
}
