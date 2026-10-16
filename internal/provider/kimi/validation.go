package kimi

import (
	"encoding/json"
	"omniproxy/internal/core/proxy"
)

func validatedBody(request proxy.Request) ([]byte, error) {
	return validatedAccountBody(Options{Region: "cn"}, request)
}

func validatedAccountBody(o Options, request proxy.Request) ([]byte, error) {
	path := "/v1/" + request.Protocol
	if request.Protocol == "chat" {
		path = "/v1/chat/completions"
	}
	parsed, err := proxy.ParseProtocolRequest(request.Body, path)
	if err != nil {
		return nil, err
	}
	if parsed.Model != request.Model || parsed.Stream != request.Stream {
		return nil, proxy.Failure("invalid_request", 400)
	}
	if !SupportsAccount(o, parsed.Model, request.Protocol) {
		return nil, proxy.Failure("unsupported_model", 400)
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(request.Body, &fields) != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	max := 1048576
	if request.Model != "kimi-k3" {
		max = 262144
	}
	for _, name := range []string{"max_tokens", "max_completion_tokens", "max_output_tokens"} {
		if raw, ok := fields[name]; ok {
			var limit int
			if json.Unmarshal(raw, &limit) != nil || limit < 1 || limit > max {
				return nil, proxy.Failure("invalid_request", 400)
			}
		}
	}
	if request.Protocol == "chat" {
		if raw, exists := fields["max_tokens"]; exists {
			if _, other := fields["max_completion_tokens"]; other {
				return nil, proxy.Failure("invalid_request", 400)
			}
			fields["max_completion_tokens"] = raw
			delete(fields, "max_tokens")
		}
		if err := accountChatParameters(o, fields, request.Model); err != nil {
			return nil, err
		}
		if request.Stream {
			options := map[string]json.RawMessage{}
			if raw, exists := fields["stream_options"]; exists {
				if json.Unmarshal(raw, &options) != nil || options == nil {
					return nil, proxy.Failure("invalid_request", 400)
				}
			}
			options["include_usage"] = json.RawMessage("true")
			fields["stream_options"], err = json.Marshal(options)
			if err != nil {
				return nil, err
			}
		}
	}
	input := fields["messages"]
	if request.Protocol == "responses" {
		input = fields["input"]
		// 官方接口固定 store=false、previous_response_id=null，需要客户端回传完整历史。
		for _, name := range []string{"store", "background"} {
			if raw, exists := fields[name]; exists {
				var value bool
				if json.Unmarshal(raw, &value) != nil || value {
					return nil, proxy.Failure("unsupported_content", 400)
				}
				delete(fields, name)
			}
		}
		if raw, exists := fields["previous_response_id"]; exists && string(raw) != "null" && string(raw) != `""` {
			return nil, proxy.Failure("unsupported_content", 400)
		}
		if raw, exists := fields["context_management"]; exists && string(raw) != "null" {
			return nil, proxy.Failure("unsupported_content", 400)
		}
		if raw, exists := fields["reasoning"]; exists {
			if err := validateEffort(raw, "effort"); err != nil {
				return nil, err
			}
		}
	}
	if request.Protocol == "messages" {
		if raw, exists := fields["output_config"]; exists && (request.Model == "kimi-k3" || o.Mode() == "coding") {
			if err := validateEffort(raw, "effort"); err != nil {
				return nil, err
			}
		}
		if raw, exists := fields["system"]; exists && !textInput(raw, 0) {
			return nil, proxy.Failure("unsupported_content", 400)
		}
	}
	if !textInput(input, 0) {
		return nil, proxy.Failure("unsupported_content", 400)
	}
	if err := validateTools(fields["tools"], request.Protocol, 0); err != nil {
		return nil, err
	}
	if o.Mode() == "coding" {
		if err := codingParameters(fields, request); err != nil {
			return nil, err
		}
		fields["model"], err = json.Marshal(UpstreamModel(o, request.Model))
		if err != nil {
			return nil, proxy.Failure("invalid_request", 400)
		}
	}
	body, err := json.Marshal(fields)
	limit := proxy.MaxRequestBytes
	if request.Protocol == "messages" {
		limit = proxy.MaxMessagesRequestBytes
	}
	if err != nil || len(body) > limit {
		clear(body)
		return nil, proxy.Failure("request_too_large", 413)
	}
	return body, nil
}

func validateEffort(raw json.RawMessage, key string) error {
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil || object == nil {
		return proxy.Failure("invalid_request", 400)
	}
	if effort, exists := object[key]; exists {
		var value string
		if json.Unmarshal(effort, &value) != nil || value != "low" && value != "high" && value != "max" {
			return proxy.Failure("invalid_request", 400)
		}
	}
	return nil
}
