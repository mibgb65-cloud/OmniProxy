package proxy

import (
	"encoding/json"
	"omniproxy/internal/core/routing"
)

// ParseProtocolRequest 统一保护请求体结构，具体工具／内容类型由适配器进一步验证。
func ParseProtocolRequest(body []byte, path string) (Request, error) {
	if path == "/v1/chat/completions" {
		r, e := ParseRequest(body)
		r.Protocol = "chat"
		return r, e
	}
	if err := validateJSON(body); err != nil {
		return Request{}, err
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil || fields == nil {
		return Request{}, Failure("invalid_request", 400)
	}
	request := Request{Body: body, Protocol: "responses"}
	if json.Unmarshal(fields["model"], &request.Model) != nil || !routing.ValidModel(request.Model) {
		return Request{}, Failure("invalid_model", 400)
	}
	if raw, ok := fields["stream"]; ok && json.Unmarshal(raw, &request.Stream) != nil {
		return Request{}, Failure("invalid_request", 400)
	}
	input := fields["input"]
	if path == "/v1/messages" {
		request.Protocol = "messages"
		input = fields["messages"]
		var max int
		if json.Unmarshal(fields["max_tokens"], &max) != nil || max < 1 || max > 393216 {
			return Request{}, Failure("invalid_request", 400)
		}
	}
	var items []json.RawMessage
	var text string
	if json.Unmarshal(input, &items) != nil {
		if request.Protocol != "responses" || json.Unmarshal(input, &text) != nil {
			return Request{}, Failure("invalid_messages", 400)
		}
	} else if len(items) > 512 {
		return Request{}, Failure("invalid_messages", 400)
	}
	return request, nil
}
