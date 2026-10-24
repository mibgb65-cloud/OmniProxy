package proxy

import (
	"bytes"
	"encoding/json"
	"mime"
	"time"
)

// InspectableQuotaResponse 判定状态码与媒体类型是否可能包装执行前的额度拒绝。
// 只有 200 与 429 有这种包装；其它类型原样转发，不做正文判定。
func InspectableQuotaResponse(status int, contentType string) bool {
	if status != 200 && status != 429 {
		return false
	}
	if contentType == "" {
		return true
	}
	media, _, err := mime.ParseMediaType(contentType)
	return err == nil && (media == "text/event-stream" || media == "application/json")
}

// QuotaRejected 只识别首个完整事件中的额度拒绝；created 或其他生成事件之后的错误绝不重放。
// 返回的第二个值是可用的重置时间，缺失或越界时为零值（拒绝仍成立）。
func QuotaRejected(prefix []byte, now time.Time) (bool, time.Time) {
	raw := firstQuotaPayload(prefix)
	if raw == nil {
		return false, time.Time{}
	}
	var value struct {
		Type     string          `json:"type"`
		Code     string          `json:"code"`
		ResetsAt json.Number     `json:"resets_at"`
		Response json.RawMessage `json:"response"`
		Output   json.RawMessage `json:"output"`
		ID       json.RawMessage `json:"id"`
		Error    *struct {
			Type     string      `json:"type"`
			Code     string      `json:"code"`
			ResetsAt json.Number `json:"resets_at"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &value) != nil || value.Type != "" && value.Type != "error" || len(value.Response) > 0 || len(value.Output) > 0 || len(value.ID) > 0 {
		return false, time.Time{}
	}
	code := value.Code
	reset := value.ResetsAt
	if value.Error != nil {
		if value.Error.Code != "" {
			code = value.Error.Code
		} else {
			code = value.Error.Type
		}
		reset = value.Error.ResetsAt
	}
	// 固定协议错误码，禁止按自然语言提示、模糊 quota 字样或一般 200 错误猜测可重试。
	if code != "usage_limit_reached" {
		return false, time.Time{}
	}
	seconds, err := reset.Int64()
	if err != nil {
		return true, time.Time{}
	}
	at := time.Unix(seconds, 0)
	if !at.After(now) || at.After(now.Add(8*24*time.Hour)) {
		return true, time.Time{}
	}
	return true, at
}

// firstQuotaPayload 只返回首个完整事件的正文：整段 JSON，或一个以空行结束的 SSE 事件。
func firstQuotaPayload(prefix []byte) []byte {
	raw := bytes.TrimSpace(prefix)
	if !json.Valid(raw) {
		var data []byte
		event := ""
		complete := false
		remaining := prefix
		for len(remaining) > 0 {
			line, tail, terminated := bytes.Cut(remaining, []byte("\n"))
			if !terminated {
				return nil
			}
			remaining = tail
			line = bytes.TrimSuffix(line, []byte("\r"))
			if len(line) == 0 {
				if data != nil {
					complete = true
					break
				}
				event = ""
				continue
			}
			if line[0] == ':' {
				continue
			}
			key, value, ok := bytes.Cut(line, []byte(":"))
			if !ok {
				return nil
			}
			value = bytes.TrimPrefix(value, []byte(" "))
			switch string(key) {
			case "event":
				event = string(value)
			case "data":
				if data != nil {
					return nil
				}
				data = value
			case "id", "retry":
			default:
				return nil
			}
		}
		if len(data) == 0 || event != "" && event != "error" {
			return nil
		}
		if !complete {
			return nil
		}
		raw = data
	}

	return raw
}
