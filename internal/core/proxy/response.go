package proxy

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
)

func copyResponse(w http.ResponseWriter, response Response, stream bool, committed *bool, protocol ...string) error {
	if response.PreExecutionQuotaRejected || response.StatusCode != http.StatusOK {
		failure := upstreamFailure(response)
		if failure.RetryAfter != "" {
			w.Header().Set("Retry-After", failure.RetryAfter)
		}
		return failure
	}
	contentType, _, err := mime.ParseMediaType(response.ContentType)
	if err != nil || response.Body == nil || (!stream && contentType != "application/json") || (stream && contentType != "text/event-stream") {
		return Failure("invalid_upstream_response", 502)
	}
	if stream {
		return copyStream(w, response.Body, committed, protocol...)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil || len(body) > maxResponseBytes || !json.Valid(body) {
		return Failure("invalid_upstream_response", 502)
	}
	var envelope struct {
		Error   json.RawMessage   `json:"error"`
		Choices []json.RawMessage `json:"choices"`
		Output  []json.RawMessage `json:"output"`
		Content []json.RawMessage `json:"content"`
	}
	valid := json.Unmarshal(body, &envelope) == nil && len(envelope.Choices) > 0
	if len(protocol) > 0 && protocol[0] == "responses" {
		valid = json.Unmarshal(body, &envelope) == nil && envelope.Output != nil
	}
	if len(protocol) > 0 && protocol[0] == "messages" {
		valid = json.Unmarshal(body, &envelope) == nil && envelope.Content != nil
	}
	if len(protocol) > 0 && protocol[0] == "responses/compact" {
		valid = validCompactionResult(body)
	}
	if !valid || hasUpstreamError(body) {
		return Failure("invalid_upstream_response", 502)
	}
	controller := http.NewResponseController(w)
	if err := controller.SetWriteDeadline(time.Now().Add(15 * time.Second)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return Failure("client_write_failed", 502)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	*committed = true
	_, err = w.Write(body)
	if err != nil {
		return Failure("client_write_failed", 502)
	}
	return nil
}

// 不把读流的底层错误传出 Core，否则凭据回调外层会将其误分类；仅返回固定安全码。
func streamReadFailure(err error) error {
	if errors.Is(err, bufio.ErrTooLong) {
		return Failure("response_too_large", 502)
	}
	return Failure("upstream_stream_failed", 502)
}

// streamDataKind 是一条 SSE 数据行的可核实分类，只用于错误码选择与日志定位。
type streamDataKind string

const (
	streamDataInvalidJSON     streamDataKind = "invalid_json"
	streamDataUnexpectedShape streamDataKind = "unexpected_shape"
	streamDataUpstreamError   streamDataKind = "upstream_error"
)

// streamDataFailure 说明一条数据行为什么不能继续转发：
// code 是返回客户端的稳定错误码，kind／eventType／upstreamCode 只进日志。
type streamDataFailure struct {
	code         string
	kind         streamDataKind
	eventType    string
	upstreamCode string
}

// inspectStreamData 按可证实的信号分类一条 SSE 数据行：上游明确报错记为
// upstream_error_event，非法 JSON 或合法但非对象保持 upstream_stream_failed。
// 上游说明文字不参与分类，也不写日志。
func inspectStreamData(data []byte) (streamDataFailure, bool) {
	if !json.Valid(data) {
		return streamDataFailure{code: "upstream_stream_failed", kind: streamDataInvalidJSON}, true
	}
	var envelope struct {
		Type     string          `json:"type"`
		Error    json.RawMessage `json:"error"`
		Response json.RawMessage `json:"response"`
	}
	if json.Unmarshal(data, &envelope) != nil {
		return streamDataFailure{code: "upstream_stream_failed", kind: streamDataUnexpectedShape}, true
	}
	eventType := sanitizeStreamSignature(envelope.Type)
	if code, reported := upstreamErrorCode(envelope.Error); reported {
		return streamDataFailure{code: "upstream_error_event", kind: streamDataUpstreamError, eventType: eventType, upstreamCode: code}, true
	}
	// 只识别 Responses 的嵌套错误，不把其他协议的同名扩展字段误判为失败。
	var response struct {
		Error json.RawMessage `json:"error"`
	}
	if json.Unmarshal(envelope.Response, &response) == nil {
		if code, reported := upstreamErrorCode(response.Error); reported {
			return streamDataFailure{code: "upstream_error_event", kind: streamDataUpstreamError, eventType: eventType, upstreamCode: code}, true
		}
	}
	return streamDataFailure{}, false
}

// upstreamErrorCode 报告 error 字段是否表达了失败，并取出可核实的短错误码。
// 只有对象形式的 code／type 会作为错误码保留；字符串与其他形状算失败但不留文本。
func upstreamErrorCode(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", false
	}
	if raw[0] != '{' {
		return "", true
	}
	var detail struct {
		Code string `json:"code"`
		Type string `json:"type"`
	}
	if json.Unmarshal(raw, &detail) != nil {
		return "", true
	}
	for _, candidate := range []string{detail.Code, detail.Type} {
		if value := sanitizeStreamSignature(candidate); value != "" {
			return value, true
		}
	}
	return "", true
}

// sanitizeStreamSignature 把上游标识压成可打印短码：去掉换行与不可打印字符，
// 最长 40 个字符，避免把上游文本原样写进日志。
func sanitizeStreamSignature(value string) string {
	const limit = 40
	var builder strings.Builder
	for _, r := range value {
		if builder.Len() >= limit {
			break
		}
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			builder.WriteRune(r)
		case strings.ContainsRune("._-", r):
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func hasUpstreamError(data []byte) bool {
	_, failed := inspectStreamData(data)
	return failed
}
