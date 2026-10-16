package antigravity

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"omniproxy/internal/core/proxy"
)

type nativePart struct {
	Text      string `json:"text"`
	Thought   bool   `json:"thought"`
	Signature string `json:"thoughtSignature"`
	Call      *struct {
		ID   string          `json:"id"`
		Name string          `json:"name"`
		Args json.RawMessage `json:"args"`
	} `json:"functionCall"`
	Inline json.RawMessage `json:"inlineData"`
}
type nativeResponse struct {
	Candidates []struct {
		Content struct {
			Parts []nativePart `json:"parts"`
		} `json:"content"`
		Finish string `json:"finishReason"`
	} `json:"candidates"`
	Usage *struct {
		Input    int64 `json:"promptTokenCount"`
		Output   int64 `json:"candidatesTokenCount"`
		Thinking int64 `json:"thoughtsTokenCount"`
		Total    int64 `json:"totalTokenCount"`
	} `json:"usageMetadata"`
	ResponseID string `json:"responseId"`
}
type responseState struct {
	model, id, finish string
	text              strings.Builder
	calls             []object
	usage             object
	created           int64
}

func (s *responseState) accept(data []byte) (object, error) {
	var wrapped struct {
		Response json.RawMessage `json:"response"`
		Error    json.RawMessage `json:"error"`
	}
	if json.Unmarshal(data, &wrapped) != nil || len(wrapped.Error) > 0 {
		return nil, proxy.Failure("invalid_upstream_response", 502)
	}
	if len(wrapped.Response) > 0 {
		data = wrapped.Response
	}
	var value nativeResponse
	if json.Unmarshal(data, &value) != nil || len(value.Candidates) > 1 {
		return nil, proxy.Failure("invalid_upstream_response", 502)
	}
	if value.ResponseID != "" && s.text.Len() == 0 && len(s.calls) == 0 {
		s.id = value.ResponseID
	}
	delta := object{}
	if len(value.Candidates) > 0 {
		candidate := value.Candidates[0]
		for _, part := range candidate.Content.Parts {
			if len(part.Inline) > 0 {
				return nil, proxy.Failure("unsupported_content", 502)
			}
			// 首版不公开或伪造思考正文，只保留工具所需的不透明签名。
			if part.Text != "" && !part.Thought {
				s.text.WriteString(part.Text)
				prior, _ := delta["content"].(string)
				delta["content"] = prior + part.Text
			}
			if part.Call != nil {
				var arguments map[string]json.RawMessage
				if !safeString(part.Call.Name, 128) || json.Unmarshal(part.Call.Args, &arguments) != nil || arguments == nil || len(part.Signature) > 65536 {
					return nil, proxy.Failure("invalid_upstream_response", 502)
				}
				id := part.Call.ID
				if id == "" {
					id = fmt.Sprintf("call_%d", len(s.calls))
				}
				call := object{"index": len(s.calls), "id": id, "type": "function", "function": object{"name": part.Call.Name, "arguments": string(part.Call.Args)}}
				if part.Signature != "" {
					call["extra_content"] = object{"google": object{"thought_signature": part.Signature}}
				}
				s.calls = append(s.calls, call)
			} else if part.Signature != "" && len(s.calls) > 0 {
				s.calls[len(s.calls)-1]["extra_content"] = object{"google": object{"thought_signature": part.Signature}}
			}
		}
		if candidate.Finish != "" {
			switch candidate.Finish {
			case "STOP":
				s.finish = "stop"
			case "MAX_TOKENS":
				s.finish = "length"
			case "SAFETY", "RECITATION", "BLOCKLIST", "PROHIBITED_CONTENT", "SPII":
				s.finish = "content_filter"
			default:
				return nil, proxy.Failure("invalid_upstream_response", 502)
			}
		}
	}
	if value.Usage != nil {
		usage := value.Usage
		if usage.Input < 0 || usage.Output < 0 || usage.Thinking < 0 || usage.Total < 0 || usage.Input > 1e12 || usage.Output > 1e12 || usage.Thinking > 1e12 || usage.Total > 3e12 {
			return nil, proxy.Failure("invalid_upstream_response", 502)
		}
		s.usage = object{"prompt_tokens": usage.Input, "completion_tokens": usage.Output + usage.Thinking, "total_tokens": usage.Total}
	}
	if s.text.Len() > 16<<20 || len(s.calls) > 128 {
		return nil, proxy.Failure("invalid_upstream_response", 502)
	}
	return delta, nil
}

func (s *responseState) envelope(stream bool, delta object) object {
	if s.created == 0 {
		s.created = time.Now().Unix()
	}
	kind, key := "chat.completion", "message"
	if stream {
		kind, key = "chat.completion.chunk", "delta"
	}
	return object{"id": s.id, "object": kind, "created": s.created, "model": s.model, "choices": []any{object{"index": 0, key: delta, "finish_reason": nil}}}
}

func (s *responseState) completed(stream bool) (object, error) {
	if s.finish == "" {
		return nil, proxy.Failure("invalid_upstream_response", 502)
	}
	finish := s.finish
	if len(s.calls) > 0 {
		finish = "tool_calls"
	}
	message := object{"role": "assistant", "content": s.text.String()}
	if stream {
		message = object{}
	}
	if len(s.calls) > 0 {
		message["tool_calls"] = s.calls
	}
	result := s.envelope(stream, message)
	result["choices"].([]any)[0].(object)["finish_reason"] = finish
	if s.usage != nil {
		result["usage"] = s.usage
	}
	return result, nil
}
