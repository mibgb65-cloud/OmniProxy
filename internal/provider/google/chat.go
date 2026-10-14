package google

import (
	"encoding/json"
	"omniproxy/internal/core/proxy"
)

func chatBody(request proxy.Request) ([]byte, error) {
	parsed, err := proxy.ParseProtocolRequest(request.Body, "/v1/chat/completions")
	if err != nil {
		return nil, err
	}
	if parsed.Model != request.Model || parsed.Stream != request.Stream {
		return nil, proxy.Failure("invalid_request", 400)
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(request.Body, &fields) != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	for _, name := range []string{"max_tokens", "max_completion_tokens"} {
		if raw, exists := fields[name]; exists {
			var limit int64
			if json.Unmarshal(raw, &limit) != nil || limit < 1 || limit > 10000000 {
				return nil, proxy.Failure("invalid_request", 400)
			}
		}
	}
	if request.Stream {
		options := map[string]json.RawMessage{}
		if raw, exists := fields["stream_options"]; exists && string(raw) != "null" {
			if json.Unmarshal(raw, &options) != nil || options == nil {
				return nil, proxy.Failure("invalid_request", 400)
			}
		}
		options["include_usage"] = json.RawMessage("true")
		encoded, err := json.Marshal(options)
		if err != nil {
			return nil, proxy.Failure("invalid_request", 400)
		}
		fields["stream_options"] = encoded
		body, err := json.Marshal(fields)
		if err != nil || len(body) > proxy.MaxRequestBytes {
			return nil, proxy.Failure("request_too_large", 413)
		}
		return body, nil
	}
	return request.Body, nil
}
