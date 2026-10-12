package agentprotocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

func responseObject(r Result) Object {
	output := []any{}
	if r.Text != "" {
		output = append(output, Object{"type": "message", "id": "msg_" + r.ID, "role": "assistant", "status": "completed", "content": []any{Object{"type": "output_text", "text": r.Text, "annotations": []any{}}}})
	}
	for _, c := range r.Calls {
		item := Object{"type": "function_call", "id": "fc_" + c.ID, "call_id": c.ID, "name": c.Name, "arguments": c.Arguments, "status": "completed"}
		if namespace, name, ok := strings.Cut(c.Name, "."); ok {
			item["namespace"] = namespace
			item["name"] = name
		}
		if c.Kind == "custom" {
			var arg Object
			if json.Unmarshal([]byte(c.Arguments), &arg) == nil {
				item["type"] = "custom_tool_call"
				item["input"] = text(arg["input"])
				delete(item, "arguments")
			}
		}
		output = append(output, item)
	}
	return Object{"id": "resp_" + r.ID, "object": "response", "created_at": time.Now().Unix(), "status": "completed", "model": r.Model, "output": output, "usage": Object{"input_tokens": r.InputTokens, "output_tokens": r.OutputTokens, "total_tokens": r.InputTokens + r.OutputTokens}, "error": nil}
}

// Encode 输出符合客户端事件顺序的有界 SSE，工具调用只在参数完整后发送。
func Encode(r Result, protocol string, stream bool) (io.Reader, string) {
	if protocol == "messages" {
		return encodeMessages(r, stream)
	}
	if protocol == "chat" {
		return encodeChat(r, stream)
	}
	response := responseObject(r)
	if !stream {
		return bytes.NewReader(encode(response)), "application/json"
	}
	var buffer bytes.Buffer
	sequence := 0
	send := func(kind string, event Object) {
		event["type"] = kind
		event["sequence_number"] = sequence
		sequence++
		fmt.Fprintf(&buffer, "event: %s\ndata: %s\n\n", kind, encode(event))
	}
	initial := Object{}
	for k, v := range response {
		initial[k] = v
	}
	initial["status"] = "in_progress"
	initial["output"] = []any{}
	send("response.created", Object{"response": initial})
	send("response.in_progress", Object{"response": initial})
	for i, v := range array(response["output"]) {
		item := object(v)
		start := Object{}
		for k, v := range item {
			start[k] = v
		}
		start["status"] = "in_progress"
		if text(item["type"]) == "message" {
			start["content"] = []any{}
		} else {
			start["arguments"] = ""
			if text(item["type"]) == "custom_tool_call" {
				start["input"] = ""
				delete(start, "arguments")
			}
		}
		send("response.output_item.added", Object{"output_index": i, "item": start})
		meta := func() Object { return Object{"item_id": item["id"], "output_index": i, "content_index": 0} }
		switch text(item["type"]) {
		case "message":
			part := Object{"type": "output_text", "text": "", "annotations": []any{}}
			e := meta()
			e["part"] = part
			send("response.content_part.added", e)
			e = meta()
			e["delta"] = r.Text
			send("response.output_text.delta", e)
			e = meta()
			e["text"] = r.Text
			send("response.output_text.done", e)
			e = meta()
			e["part"] = array(item["content"])[0]
			send("response.content_part.done", e)
		case "custom_tool_call":
			e := meta()
			e["delta"] = item["input"]
			send("response.custom_tool_call_input.delta", e)
			e = meta()
			e["input"] = item["input"]
			send("response.custom_tool_call_input.done", e)
		default:
			e := meta()
			e["delta"] = item["arguments"]
			send("response.function_call_arguments.delta", e)
			e = meta()
			e["arguments"] = item["arguments"]
			send("response.function_call_arguments.done", e)
		}
		send("response.output_item.done", Object{"output_index": i, "item": item})
	}
	send("response.completed", Object{"response": response})
	return &buffer, "text/event-stream"
}
