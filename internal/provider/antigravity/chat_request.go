package antigravity

import (
	"crypto/rand"
	"encoding/json"
	"fmt"

	"omniproxy/internal/core/proxy"
)

type object = map[string]any
type chatTool struct {
	Type     string          `json:"type"`
	Function json.RawMessage `json:"function"`
}
type chatMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	ToolID  string          `json:"tool_call_id"`
	Calls   []struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
		Extra json.RawMessage `json:"extra_content"`
	} `json:"tool_calls"`
}

func textContent(raw json.RawMessage) (string, error) {
	if string(raw) == "null" || len(raw) == 0 {
		return "", nil
	}
	var value string
	if json.Unmarshal(raw, &value) == nil {
		return value, nil
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) != nil || len(parts) > 1000 {
		return "", proxy.Failure("unsupported_content", 400)
	}
	for _, part := range parts {
		if part.Type != "text" {
			return "", proxy.Failure("unsupported_content", 400)
		}
		value += part.Text
	}
	return value, nil
}

func chatPayload(request proxy.Request, cred Credentials) ([]byte, error) {
	if request.Protocol != "chat" || !supportedModel(request.Model) {
		return nil, proxy.Failure("unsupported_protocol", 400)
	}
	var input struct {
		Messages   []chatMessage   `json:"messages"`
		Tools      []chatTool      `json:"tools"`
		ToolChoice json.RawMessage `json:"tool_choice"`
	}
	if json.Unmarshal(request.Body, &input) != nil || len(input.Messages) == 0 || len(input.Messages) > 1000 || len(input.Tools) > 128 {
		return nil, proxy.Failure("invalid_request", 400)
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(request.Body, &fields) != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	for _, key := range []string{"response_format", "audio", "modalities", "prediction", "reasoning_effort", "extra_body"} {
		if value, ok := fields[key]; ok && string(value) != "null" {
			return nil, proxy.Failure("unsupported_content", 400)
		}
	}
	native := object{}
	contents, systems := []any{}, []any{}
	calls := map[string]string{}
	for _, message := range input.Messages {
		text, err := textContent(message.Content)
		if err != nil {
			return nil, err
		}
		parts := []any{}
		if text != "" {
			parts = append(parts, object{"text": text})
		}
		switch message.Role {
		case "system", "developer":
			if len(message.Calls) > 0 {
				return nil, proxy.Failure("unsupported_content", 400)
			}
			systems = append(systems, parts...)
			continue
		case "assistant":
			for _, call := range message.Calls {
				if !safeString(call.ID, 256) || !safeString(call.Function.Name, 128) || call.Type != "function" || calls[call.ID] != "" {
					return nil, proxy.Failure("invalid_request", 400)
				}
				var args object
				if json.Unmarshal([]byte(call.Function.Arguments), &args) != nil || args == nil {
					return nil, proxy.Failure("invalid_request", 400)
				}
				part := object{"functionCall": object{"id": call.ID, "name": call.Function.Name, "args": args}}
				if len(call.Extra) > 0 {
					var extra struct {
						Google struct {
							Signature string `json:"thought_signature"`
						} `json:"google"`
					}
					if json.Unmarshal(call.Extra, &extra) != nil || len(extra.Google.Signature) > 65536 {
						return nil, proxy.Failure("invalid_request", 400)
					}
					if extra.Google.Signature != "" {
						part["thoughtSignature"] = extra.Google.Signature
					}
				}
				parts = append(parts, part)
				calls[call.ID] = call.Function.Name
			}
		case "tool":
			name := calls[message.ToolID]
			if name == "" {
				return nil, proxy.Failure("invalid_request", 400)
			}
			parts = []any{object{"functionResponse": object{"id": message.ToolID, "name": name, "response": object{"result": text}}}}
			delete(calls, message.ToolID)
		case "user":
			if len(message.Calls) > 0 {
				return nil, proxy.Failure("invalid_request", 400)
			}
		default:
			return nil, proxy.Failure("unsupported_content", 400)
		}
		role := "user"
		if message.Role == "assistant" {
			role = "model"
		}
		if len(parts) == 0 {
			return nil, proxy.Failure("unsupported_content", 400)
		}
		contents = append(contents, object{"role": role, "parts": parts})
	}
	if len(contents) == 0 || len(calls) > 0 {
		return nil, proxy.Failure("invalid_request", 400)
	}
	native["contents"] = contents
	if len(systems) > 0 {
		native["systemInstruction"] = object{"parts": systems}
	}
	if len(input.Tools) > 0 {
		declarations, err := toolDeclarations(input.Tools)
		if err != nil {
			return nil, err
		}
		native["tools"] = []any{object{"functionDeclarations": declarations}}
	}
	choice, err := toolChoice(input.ToolChoice, input.Tools)
	if err != nil {
		return nil, err
	}
	if choice != nil {
		native["toolConfig"] = choice
	}
	config, err := generationOptions(fields)
	if err != nil {
		return nil, err
	}
	if len(config) > 0 {
		native["generationConfig"] = config
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, Failure("operation_failed")
	}
	id[6] = (id[6] & 15) | 64
	id[8] = (id[8] & 63) | 128
	native["sessionId"] = fmt.Sprintf("-%d", uint64(id[0])<<48|uint64(id[1])<<40|uint64(id[2])<<32|uint64(id[3])<<24|uint64(id[4])<<16|uint64(id[5])<<8|uint64(id[7]))
	return json.Marshal(object{"project": cred.Project, "model": request.Model, "userAgent": "antigravity", "requestType": "agent", "requestId": fmt.Sprintf("agent-%x-%x-%x-%x-%x", id[0:4], id[4:6], id[6:8], id[8:10], id[10:16]), "request": native})
}
