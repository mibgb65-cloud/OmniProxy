package agentprotocol

import "encoding/json"

func ChatToResponses(body []byte) ([]byte, error) {
	var chat Object
	if json.Unmarshal(body, &chat) != nil {
		return nil, ErrUnsupported
	}
	input := []any{}
	for _, v := range array(chat["messages"]) {
		m := object(v)
		role := text(m["role"])
		if role == "tool" {
			input = append(input, Object{"type": "function_call_output", "call_id": m["tool_call_id"], "output": m["content"]})
			continue
		}
		if text(m["content"]) != "" {
			input = append(input, Object{"type": "message", "role": role, "content": m["content"]})
		}
		for _, v := range array(m["tool_calls"]) {
			c := object(v)
			f := object(c["function"])
			input = append(input, Object{"type": "function_call", "call_id": c["id"], "name": f["name"], "arguments": f["arguments"]})
		}
	}
	tools := []any{}
	for _, v := range array(chat["tools"]) {
		f := object(object(v)["function"])
		tools = append(tools, Object{"type": "function", "name": f["name"], "description": f["description"], "parameters": f["parameters"]})
	}
	result := Object{"model": chat["model"], "input": input, "tools": tools, "stream": true, "store": false, "instructions": ""}
	if v, ok := chat["tool_choice"]; ok {
		if m := object(v); m != nil {
			v = Object{"type": "function", "name": object(m["function"])["name"]}
		}
		result["tool_choice"] = v
	}
	return encode(result), nil
}
