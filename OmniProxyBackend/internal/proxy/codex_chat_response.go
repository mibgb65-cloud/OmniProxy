package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func convertCodexResponsesToChat(resp *http.Response, requestedModel string, clientStream bool) (*http.Response, error) {
	header := resp.Header.Clone()
	header.Del("Content-Length")
	status := strconv.Itoa(resp.StatusCode) + " " + http.StatusText(resp.StatusCode)

	if clientStream {
		header.Set("Content-Type", "text/event-stream; charset=utf-8")
		return &http.Response{
			StatusCode: resp.StatusCode,
			Status:     status,
			Header:     header,
			Body:       codexChatSSEBody(resp.Body, requestedModel),
		}, nil
	}

	defer closeBody(resp.Body)
	body, err := readProxyResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}
	converted, err := codexResponsesBodyToChatJSON(resp.Header, body, requestedModel)
	if err != nil {
		return nil, err
	}
	header.Set("Content-Type", "application/json; charset=utf-8")
	return &http.Response{
		StatusCode: resp.StatusCode,
		Status:     status,
		Header:     header,
		Body:       io.NopCloser(bytes.NewReader(converted)),
	}, nil
}

// codexChatSSEBody rewrites the upstream Responses stream into chat chunks as
// they arrive. The upstream request always sets stream:true, so buffering the
// whole body first would hold every token back until the generation finished.
func codexChatSSEBody(src io.ReadCloser, requestedModel string) io.ReadCloser {
	reader, writer := io.Pipe()
	go func() {
		defer closeBody(src)
		_ = writer.CloseWithError(streamCodexResponsesToChatSSE(src, writer, requestedModel))
	}()
	return reader
}

func streamCodexResponsesToChatSSE(src io.Reader, dst io.Writer, requestedModel string) error {
	stream := &codexChatSSEStream{
		dst:     dst,
		created: time.Now().Unix(),
		id:      codexChatID(""),
		model:   requestedModel,
	}
	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, 0, 64*1024), maxProxyRequestBodyBytes)
	for scanner.Scan() {
		event, ok := codexParseResponsesSSELine(scanner.Text())
		if !ok {
			continue
		}
		if err := stream.write(event); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return stream.finish()
}

func codexResponsesBodyToChatJSON(header http.Header, body []byte, requestedModel string) ([]byte, error) {
	if strings.Contains(strings.ToLower(header.Get("Content-Type")), "text/event-stream") || bytes.Contains(body, []byte("data:")) {
		events := codexParseResponsesSSE(body)
		resp, deltaText := codexTerminalResponse(events)
		return json.Marshal(codexBuildChatCompletion(resp, requestedModel, deltaText))
	}
	var resp codexResponsesPayload
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&resp); err != nil {
		return nil, err
	}
	return json.Marshal(codexBuildChatCompletion(&resp, requestedModel, ""))
}

// codexChatSSEStream turns Responses events into chat chunks one at a time so
// the conversion can run while the upstream stream is still open.
type codexChatSSEStream struct {
	dst       io.Writer
	created   int64
	id        string
	model     string
	roleSent  bool
	finalized bool
}

func (s *codexChatSSEStream) chunk(delta codexChatDelta, finishReason *string) error {
	data, err := json.Marshal(codexChatChunk{
		ID:      s.id,
		Object:  "chat.completion.chunk",
		Created: s.created,
		Model:   s.model,
		Choices: []codexChunkChoice{{Index: 0, Delta: delta, FinishReason: finishReason}},
	})
	if err != nil {
		return err
	}
	_, err = s.dst.Write([]byte("data: " + string(data) + "\n\n"))
	return err
}

func (s *codexChatSSEStream) role() error {
	if s.roleSent {
		return nil
	}
	s.roleSent = true
	return s.chunk(codexChatDelta{Role: "assistant"}, nil)
}

func (s *codexChatSSEStream) write(event codexResponsesEvent) error {
	if event.Response != nil {
		if event.Response.ID != "" {
			s.id = codexChatID(event.Response.ID)
		}
		if event.Response.Model != "" {
			s.model = event.Response.Model
		}
	}

	switch event.Type {
	case "response.created":
		return s.role()
	case "response.output_text.delta":
		if event.Delta == "" {
			return nil
		}
		if err := s.role(); err != nil {
			return err
		}
		content := event.Delta
		return s.chunk(codexChatDelta{Content: &content}, nil)
	case "response.completed", "response.done", "response.incomplete", "response.failed":
		if err := s.role(); err != nil {
			return err
		}
		finishReason := codexFinishReason(event.Response)
		if err := s.chunk(codexChatDelta{}, &finishReason); err != nil {
			return err
		}
		s.finalized = true
	}
	return nil
}

func (s *codexChatSSEStream) finish() error {
	if !s.finalized {
		if err := s.role(); err != nil {
			return err
		}
		finishReason := "stop"
		if err := s.chunk(codexChatDelta{}, &finishReason); err != nil {
			return err
		}
	}
	_, err := s.dst.Write([]byte("data: [DONE]\n\n"))
	return err
}

func codexParseResponsesSSE(body []byte) []codexResponsesEvent {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), maxProxyRequestBodyBytes)
	events := []codexResponsesEvent{}
	for scanner.Scan() {
		if event, ok := codexParseResponsesSSELine(scanner.Text()); ok {
			events = append(events, event)
		}
	}
	return events
}

func codexParseResponsesSSELine(line string) (codexResponsesEvent, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "data:") {
		return codexResponsesEvent{}, false
	}
	data := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
	if data == "" || data == "[DONE]" {
		return codexResponsesEvent{}, false
	}
	var event codexResponsesEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil || event.Type == "" {
		return codexResponsesEvent{}, false
	}
	return event, true
}

func codexTerminalResponse(events []codexResponsesEvent) (*codexResponsesPayload, string) {
	var text strings.Builder
	var terminal *codexResponsesPayload
	for _, event := range events {
		if event.Type == "response.output_text.delta" && event.Delta != "" {
			text.WriteString(event.Delta)
		}
		switch event.Type {
		case "response.completed", "response.done", "response.incomplete", "response.failed":
			if event.Response != nil {
				terminal = event.Response
			}
		}
	}
	return terminal, text.String()
}

func codexBuildChatCompletion(resp *codexResponsesPayload, requestedModel string, fallbackText string) codexChatCompletion {
	created := time.Now().Unix()
	model := strings.TrimSpace(requestedModel)
	id := ""
	text := fallbackText
	finishReason := "stop"
	var usage *codexChatUsage

	if resp != nil {
		id = resp.ID
		if resp.Model != "" {
			model = resp.Model
		}
		if outputText := codexResponsesOutputText(resp.Output); outputText != "" {
			text = outputText
		}
		finishReason = codexFinishReason(resp)
		if resp.Usage != nil {
			total := resp.Usage.TotalTokens
			if total == 0 && (resp.Usage.InputTokens > 0 || resp.Usage.OutputTokens > 0) {
				total = resp.Usage.InputTokens + resp.Usage.OutputTokens
			}
			usage = &codexChatUsage{
				PromptTokens:     resp.Usage.InputTokens,
				CompletionTokens: resp.Usage.OutputTokens,
				TotalTokens:      total,
			}
		}
	}
	if model == "" {
		model = "gpt-5.6-sol"
	}
	return codexChatCompletion{
		ID:      codexChatID(id),
		Object:  "chat.completion",
		Created: created,
		Model:   model,
		Choices: []codexChatChoice{{
			Index:        0,
			Message:      codexChatOutput{Role: "assistant", Content: text},
			FinishReason: finishReason,
		}},
		Usage: usage,
	}
}

func codexResponsesOutputText(outputs []codexResponsesOutput) string {
	var text strings.Builder
	for _, output := range outputs {
		switch output.Type {
		case "message":
			for _, part := range output.Content {
				if part.Type == "output_text" || part.Type == "text" {
					text.WriteString(part.Text)
				}
			}
		case "reasoning":
			continue
		}
	}
	return text.String()
}

func codexFinishReason(resp *codexResponsesPayload) string {
	if resp == nil {
		return "stop"
	}
	if resp.Status == "incomplete" && resp.IncompleteDetails != nil && resp.IncompleteDetails.Reason == "max_output_tokens" {
		return "length"
	}
	return "stop"
}

func codexJSONProxyResponse(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     strconv.Itoa(status) + " " + http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{"application/json; charset=utf-8"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func codexStringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}

func codexChatID(value string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return "chatcmpl-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func normalizeCodexChatModel(model string) string {
	key := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(model)), "-"))
	if key == "" {
		return "gpt-5.6-sol"
	}
	if strings.Contains(key, "/") {
		parts := strings.Split(key, "/")
		key = parts[len(parts)-1]
	}
	modelMap := map[string]string{
		"gpt-5.6":             "gpt-5.6-sol",
		"gpt-5.6-sol":         "gpt-5.6-sol",
		"gpt-5.6-terra":       "gpt-5.6-terra",
		"gpt-5.6-luna":        "gpt-5.6-luna",
		"gpt-5.5":             "gpt-5.5",
		"gpt-5.4":             "gpt-5.4",
		"gpt-5.4-mini":        "gpt-5.4-mini",
		"gpt-5.4-none":        "gpt-5.4",
		"gpt-5.4-low":         "gpt-5.4",
		"gpt-5.4-medium":      "gpt-5.4",
		"gpt-5.4-high":        "gpt-5.4",
		"gpt-5.4-xhigh":       "gpt-5.4",
		"gpt-5.3":             "gpt-5.3-codex",
		"gpt-5.3-codex":       "gpt-5.3-codex",
		"gpt-5.3-codex-spark": "gpt-5.3-codex-spark",
		"gpt-5.2":             "gpt-5.2",
		"gpt-5":               "gpt-5.4",
		"gpt-5-mini":          "gpt-5.4",
		"gpt-5-nano":          "gpt-5.4",
		"gpt-5.1":             "gpt-5.4",
		"gpt-5.1-codex":       "gpt-5.3-codex",
		"gpt-5.1-codex-max":   "gpt-5.3-codex",
		"gpt-5.1-codex-mini":  "gpt-5.3-codex",
		"gpt-5.2-codex":       "gpt-5.2",
		"codex-mini-latest":   "gpt-5.3-codex",
		"gpt-5-codex":         "gpt-5.3-codex",
	}
	if mapped, ok := modelMap[key]; ok {
		return mapped
	}
	for _, prefix := range []string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna", "gpt-5.6", "gpt-5.5", "gpt-5.4-mini", "gpt-5.4", "gpt-5.3-codex-spark", "gpt-5.3-codex", "gpt-5.2"} {
		if key == prefix || strings.HasPrefix(key, prefix+"-") {
			return modelMap[prefix]
		}
	}
	return strings.TrimSpace(model)
}
