package proxy

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func copyResponse(w http.ResponseWriter, response Response, stream bool, committed *bool, protocol ...string) error {
	if response.StatusCode != http.StatusOK {
		status, code := response.StatusCode, "upstream_rejected_request"
		switch status {
		case 401, 403:
			code = "upstream_authentication_failed"
		case 402:
			code = "upstream_insufficient_balance"
		case 429:
			code = "upstream_rate_limited"
		default:
			if status < 400 || status >= 500 {
				status, code = 502, "upstream_unavailable"
			}
		}
		if value, err := strconv.Atoi(response.RetryAfter); err == nil && value >= 0 && value <= 86400 {
			w.Header().Set("Retry-After", strconv.Itoa(value))
		}
		return Failure(code, status)
	}
	contentType, _, err := mime.ParseMediaType(response.ContentType)
	if err != nil || response.Body == nil || (!stream && contentType != "application/json") || (stream && contentType != "text/event-stream") {
		return Failure("invalid_upstream_response", 502)
	}
	if stream {
		return copyStream(w, response.Body, committed)
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
	return err
}

func copyStream(w http.ResponseWriter, body io.Reader, committed *bool) error {
	controller := http.NewResponseController(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	scanner := bufio.NewScanner(io.LimitReader(body, maxStreamBytes+1))
	scanner.Buffer(make([]byte, 4096), 512<<10)
	var total int
	for scanner.Scan() {
		line := scanner.Text()
		total += len(line) + 1
		if total > maxStreamBytes {
			return Failure("response_too_large", 502)
		}
		done := false
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			done = data == "[DONE]"
			if !done {
				var event struct {
					Type string `json:"type"`
				}
				if json.Unmarshal([]byte(data), &event) == nil {
					done = event.Type == "response.completed" || event.Type == "message_stop"
				}
			}
			if data != "" && !done && (!json.Valid([]byte(data)) || hasUpstreamError([]byte(data))) {
				return Failure("upstream_stream_failed", 502)
			}
		}
		if err := controller.SetWriteDeadline(time.Now().Add(15 * time.Second)); err != nil && !errors.Is(err, http.ErrNotSupported) {
			return Failure("client_write_failed", 502)
		}
		*committed = true
		frame := line + "\n"
		if done {
			frame += "\n"
		}
		if _, err := io.WriteString(w, frame); err != nil {
			return err
		}
		if err := controller.Flush(); err != nil {
			return err
		}
		if done {
			return nil
		}
	}
	return Failure("upstream_stream_failed", 502)
}

func hasUpstreamError(data []byte) bool {
	var envelope struct {
		Error json.RawMessage `json:"error"`
	}
	if json.Unmarshal(data, &envelope) != nil {
		return true
	}
	return len(envelope.Error) > 0 && string(envelope.Error) != "null"
}
