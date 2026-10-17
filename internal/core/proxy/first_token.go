package proxy

import (
	"encoding/json"
	"time"
)

// 首字只识别客户端实际收到的模型内容，不把握手、心跳、用量或空 delta 当作输出。
func hasModelOutput(protocol string, data []byte) bool {
	var event struct {
		Type    string          `json:"type"`
		Delta   json.RawMessage `json:"delta"`
		Choices []struct {
			Delta struct {
				Content      json.RawMessage                  `json:"content"`
				Reasoning    string                           `json:"reasoning_content"`
				Refusal      string                           `json:"refusal"`
				FunctionCall struct{ Name, Arguments string } `json:"function_call"`
				ToolCalls    []struct {
					Function struct{ Name, Arguments string } `json:"function"`
				} `json:"tool_calls"`
			} `json:"delta"`
		} `json:"choices"`
		ContentBlock struct{ Type, Text, Thinking, Name string }   `json:"content_block"`
		Item         struct{ Type, Name, Arguments, Input string } `json:"item"`
		Part         struct{ Type, Text, Refusal string }          `json:"part"`
	}
	if json.Unmarshal(data, &event) != nil {
		return false
	}
	switch protocol {
	case "chat":
		for _, choice := range event.Choices {
			delta := choice.Delta
			if nonemptyText(delta.Content) || delta.Reasoning != "" || delta.Refusal != "" || delta.FunctionCall.Name != "" || delta.FunctionCall.Arguments != "" {
				return true
			}
			for _, tool := range delta.ToolCalls {
				if tool.Function.Name != "" || tool.Function.Arguments != "" {
					return true
				}
			}
		}
	case "responses":
		switch event.Type {
		case "response.output_text.delta", "response.reasoning_text.delta", "response.reasoning_summary_text.delta", "response.refusal.delta", "response.function_call_arguments.delta", "response.custom_tool_call_input.delta":
			var delta string
			return json.Unmarshal(event.Delta, &delta) == nil && delta != ""
		case "response.output_item.added":
			return (event.Item.Type == "function_call" || event.Item.Type == "custom_tool_call") && (event.Item.Name != "" || event.Item.Arguments != "" || event.Item.Input != "")
		case "response.content_part.added":
			return event.Part.Type == "output_text" && event.Part.Text != "" || event.Part.Type == "refusal" && event.Part.Refusal != ""
		}
	case "messages":
		if event.Type == "content_block_start" {
			block := event.ContentBlock
			return block.Type == "text" && block.Text != "" || block.Type == "thinking" && block.Thinking != "" || block.Type == "tool_use" && block.Name != ""
		}
		if event.Type == "content_block_delta" {
			var delta struct {
				Type, Text, Thinking string
				PartialJSON          string `json:"partial_json"`
			}
			if json.Unmarshal(event.Delta, &delta) == nil {
				return delta.Type == "text_delta" && delta.Text != "" || delta.Type == "thinking_delta" && delta.Thinking != "" || delta.Type == "input_json_delta" && delta.PartialJSON != ""
			}
		}
	}
	return false
}

func nonemptyText(data json.RawMessage) bool {
	var value string
	if json.Unmarshal(data, &value) == nil {
		return value != ""
	}
	var parts []struct{ Type, Text string }
	if json.Unmarshal(data, &parts) == nil {
		for _, part := range parts {
			if (part.Type == "text" || part.Type == "output_text") && part.Text != "" {
				return true
			}
		}
	}
	return false
}

func (t *Trace) hasFirstContent(data []byte) bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.event.Stream && t.event.FirstTokenMS == nil && hasModelOutput(t.event.Protocol, data)
}

func (t *Trace) markFirstToken() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.event.Stream && t.event.FirstTokenMS == nil {
		elapsed := t.elapsedMS()
		t.event.FirstTokenMS = &elapsed
	}
}

func (t *Trace) elapsedMS() int64 {
	start := t.startedAt
	if start.IsZero() {
		start = t.event.StartedAt
	}
	return max(0, time.Since(start).Milliseconds())
}
