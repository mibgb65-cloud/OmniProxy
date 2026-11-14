package proxy

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
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

func hasUpstreamError(data []byte) bool {
	var envelope struct {
		Error    json.RawMessage `json:"error"`
		Response json.RawMessage `json:"response"`
	}
	if json.Unmarshal(data, &envelope) != nil {
		return true
	}
	if len(envelope.Error) > 0 && string(envelope.Error) != "null" {
		return true
	}
	// 只识别 Responses 的嵌套错误，不把其他协议的同名扩展字段误判为失败。
	var response struct {
		Error json.RawMessage `json:"error"`
	}
	return json.Unmarshal(envelope.Response, &response) == nil &&
		len(response.Error) > 0 && string(response.Error) != "null"
}
