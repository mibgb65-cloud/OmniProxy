package chatgpt

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"omniproxy/internal/core/proxy"
)

// 与 Responses 两种传输的事件上限一致；只查看首事件，不能等待整个响应结束。
const maxSSEPrefixBytes = proxy.MaxResponsesEventBytes

func inferResponseType(response proxy.Response) proxy.Response {
	infer := response.StatusCode == 200 && strings.TrimSpace(response.ContentType) == ""
	if !infer && !canInspectQuotaResponse(response.StatusCode, response.ContentType) {
		return response
	}
	if response.Body == nil {
		if infer {
			response.ContentTypeInferenceReason = "missing_body"
		}
		return response
	}
	var prefix bytes.Buffer
	reader := bufio.NewReader(io.TeeReader(io.LimitReader(response.Body, maxSSEPrefixBytes), &prefix))
	reason := firstEventReason(reader)
	if reason == "incomplete_event" {
		switch {
		case prefix.Len() == maxSSEPrefixBytes:
			reason = "prefix_limit"
		case prefix.Len() == 0:
			reason = "empty"
		}
	}
	if reason != "verified" {
		trimmed := bytes.TrimSpace(prefix.Bytes())
		if json.Valid(trimmed) {
			reason = "non_sse_json"
		} else if bytes.HasPrefix(trimmed, []byte("<")) {
			reason = "non_sse_html"
		}
	}
	// 执行前的明确额度拒绝与已执行后断流分开，保留真实 HTTP 状态供尝试统计。
	if denied, reset := quotaRejection(prefix.Bytes(), time.Now()); denied {
		response.PreExecutionQuotaRejected = true
		response.QuotaResetAt = reset
		reason = "quota_rejected"
	}
	if infer || response.PreExecutionQuotaRejected {
		response.ContentTypeInferenceReason = reason
		response.ContentTypeInferenceBytes = prefix.Len()
	}
	// 保留 bufio 已预读的全部原字节；不丢弃后续事件，也不额外读取整段流。
	response.Body = io.MultiReader(bytes.NewReader(prefix.Bytes()), response.Body)
	if infer && reason == "verified" {
		response.ContentType = "text/event-stream"
		response.ContentTypeInferred = true
	}
	return response
}

func firstEventReason(reader *bufio.Reader) string {
	var data []byte
	eventType := ""
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "incomplete_event"
			}
			return "read_error"
		}
		line = bytes.TrimSuffix(bytes.TrimSuffix(line, []byte("\n")), []byte("\r"))
		if len(line) == 0 {
			if data == nil {
				eventType = ""
				continue
			}
			return responseEventReason(data, eventType)
		}
		if line[0] == ':' {
			continue
		}
		field, value, ok := bytes.Cut(line, []byte(":"))
		if !ok {
			return "unexpected_field"
		}
		value = bytes.TrimPrefix(value, []byte(" "))
		switch string(field) {
		case "event":
			eventType = string(value)
		case "data":
			// 现有 Responses 转发按单行 JSON 处理；不在补头时引入另一套 SSE 语义。
			if data != nil {
				return "multiple_data"
			}
			data = bytes.Clone(value)
		case "id", "retry":
		default:
			return "unexpected_field"
		}
	}
}

func responseEventReason(data []byte, eventType string) string {
	var event struct {
		Type     string `json:"type"`
		Response struct {
			ID string `json:"id"`
		} `json:"response"`
		Error json.RawMessage `json:"error"`
	}
	if json.Unmarshal(data, &event) != nil {
		return "invalid_json"
	}
	if eventType != "" && eventType != event.Type {
		return "event_mismatch"
	}
	switch event.Type {
	case "response.created", "response.in_progress", "response.completed", "response.failed", "response.incomplete":
		if len(event.Response.ID) == 0 || len(event.Response.ID) > 512 {
			return "invalid_response_id"
		}
		return "verified"
	case "error":
		if len(event.Error) == 0 || string(event.Error) == "null" {
			return "invalid_error"
		}
		return "verified"
	}
	return "unexpected_event"
}
