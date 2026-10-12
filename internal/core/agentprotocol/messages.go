package agentprotocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func encodeMessages(r Result, stream bool) (io.Reader, string) {
	blocks := []any{}
	if r.Text != "" {
		blocks = append(blocks, Object{"type": "text", "text": r.Text})
	}
	for _, c := range r.Calls {
		var input Object
		if json.Unmarshal([]byte(c.Arguments), &input) != nil {
			input = Object{}
		}
		blocks = append(blocks, Object{"type": "tool_use", "id": c.ID, "name": c.Name, "input": input})
	}
	stop := "end_turn"
	if len(r.Calls) > 0 {
		stop = "tool_use"
	}
	if r.Finish == "length" {
		stop = "max_tokens"
	}
	message := Object{"id": "msg_" + r.ID, "type": "message", "role": "assistant", "model": r.Model, "content": blocks, "stop_reason": stop, "stop_sequence": nil, "usage": Object{"input_tokens": r.InputTokens, "output_tokens": r.OutputTokens}}
	if !stream {
		return bytes.NewReader(encode(message)), "application/json"
	}
	var b bytes.Buffer
	send := func(kind string, data Object) {
		data["type"] = kind
		fmt.Fprintf(&b, "event: %s\ndata: %s\n\n", kind, encode(data))
	}
	initial := Object{}
	for k, v := range message {
		initial[k] = v
	}
	initial["content"] = []any{}
	initial["stop_reason"] = nil
	initial["usage"] = Object{"input_tokens": r.InputTokens, "output_tokens": 0}
	send("message_start", Object{"message": initial})
	for i, v := range blocks {
		block := object(v)
		start := Object{}
		for k, v := range block {
			start[k] = v
		}
		delta := Object{"type": "text_delta", "text": block["text"]}
		if text(block["type"]) == "tool_use" {
			start["input"] = Object{}
			delta = Object{"type": "input_json_delta", "partial_json": string(encode(block["input"]))}
		} else {
			start["text"] = ""
		}
		send("content_block_start", Object{"index": i, "content_block": start})
		send("content_block_delta", Object{"index": i, "delta": delta})
		send("content_block_stop", Object{"index": i})
	}
	send("message_delta", Object{"delta": Object{"stop_reason": stop, "stop_sequence": nil}, "usage": Object{"output_tokens": r.OutputTokens}})
	send("message_stop", Object{})
	return &b, "text/event-stream"
}
func encodeChat(r Result, stream bool) (io.Reader, string) {
	calls := []any{}
	for _, c := range r.Calls {
		calls = append(calls, Object{"id": c.ID, "type": "function", "function": Object{"name": c.Name, "arguments": c.Arguments}})
	}
	message := Object{"role": "assistant", "content": r.Text}
	if len(calls) > 0 {
		message["tool_calls"] = calls
	}
	result := Object{"id": r.ID, "object": "chat.completion", "model": r.Model, "choices": []any{Object{"index": 0, "message": message, "finish_reason": r.Finish}}, "usage": Object{"prompt_tokens": r.InputTokens, "completion_tokens": r.OutputTokens, "total_tokens": r.InputTokens + r.OutputTokens}}
	if !stream {
		return bytes.NewReader(encode(result)), "application/json"
	}
	var b bytes.Buffer
	for i, v := range calls {
		object(v)["index"] = i
	}
	result["object"] = "chat.completion.chunk"
	result["choices"] = []any{Object{"index": 0, "delta": message, "finish_reason": nil}}
	fmt.Fprintf(&b, "data: %s\n\n", encode(result))
	result["choices"] = []any{Object{"index": 0, "delta": Object{}, "finish_reason": r.Finish}}
	fmt.Fprintf(&b, "data: %s\n\ndata: [DONE]\n\n", encode(result))
	return &b, "text/event-stream"
}
