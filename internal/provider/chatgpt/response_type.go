package chatgpt

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"omniproxy/internal/core/proxy"
)

// 与 HTTP 转发的单行上限一致；只查看首事件，不能等待整个响应结束。
const maxSSEPrefixBytes = 512 << 10

func inferResponseType(response proxy.Response) proxy.Response {
	if response.StatusCode != 200 || strings.TrimSpace(response.ContentType) != "" || response.Body == nil {
		return response
	}
	// 固定订阅端点确有省略 Content-Type 的 SSE。只补已验证的缺失头，不覆盖明确声明的其他类型。
	var prefix bytes.Buffer
	reader := bufio.NewReader(io.TeeReader(io.LimitReader(response.Body, maxSSEPrefixBytes), &prefix))
	valid := validFirstEvent(reader)
	// Tee 同时保存 bufio 的预读字节，消费端必须看到逐字节相同的完整流。
	response.Body = io.MultiReader(bytes.NewReader(prefix.Bytes()), response.Body)
	if valid {
		response.ContentType = "text/event-stream"
		response.ContentTypeInferred = true
	}
	return response
}

func validFirstEvent(reader *bufio.Reader) bool {
	var data []byte
	eventType := ""
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return false
		}
		line = bytes.TrimSuffix(bytes.TrimSuffix(line, []byte("\n")), []byte("\r"))
		if len(line) == 0 {
			if data == nil {
				eventType = ""
				continue
			}
			return validResponseEvent(data, eventType)
		}
		if line[0] == ':' {
			continue
		}
		field, value, ok := bytes.Cut(line, []byte(":"))
		if !ok {
			return false
		}
		value = bytes.TrimPrefix(value, []byte(" "))
		switch string(field) {
		case "event":
			eventType = string(value)
		case "data":
			// 现有 Responses 转发按单行 JSON 处理；不在补头时引入另一套 SSE 语义。
			if data != nil {
				return false
			}
			data = bytes.Clone(value)
		case "id", "retry":
		default:
			return false
		}
	}
}

func validResponseEvent(data []byte, eventType string) bool {
	var event struct {
		Type     string `json:"type"`
		Response struct {
			ID string `json:"id"`
		} `json:"response"`
		Error json.RawMessage `json:"error"`
	}
	if json.Unmarshal(data, &event) != nil || eventType != "" && eventType != event.Type {
		return false
	}
	switch event.Type {
	case "response.created", "response.in_progress", "response.completed", "response.failed", "response.incomplete":
		return len(event.Response.ID) > 0 && len(event.Response.ID) <= 512
	case "error":
		return len(event.Error) > 0 && string(event.Error) != "null"
	}
	return false
}
