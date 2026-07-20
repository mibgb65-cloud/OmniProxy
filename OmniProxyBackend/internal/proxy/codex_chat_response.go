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
	defer closeBody(resp.Body)
	body, err := readProxyResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}
	header := resp.Header.Clone()
	var converted []byte
	if clientStream {
		converted, err = codexResponsesSSEToChatSSE(body, requestedModel)
		header.Set("Content-Type", "text/event-stream; charset=utf-8")
	} else {
		converted, err = codexResponsesBodyToChatJSON(resp.Header, body, requestedModel)
		header.Set("Content-Type", "application/json; charset=utf-8")
	}
	if err != nil {
		return nil, err
	}
	header.Del("Content-Length")
	return &http.Response{
		StatusCode: resp.StatusCode,
		Status:     strconv.Itoa(resp.StatusCode) + " " + http.StatusText(resp.StatusCode),
		Header:     header,
		Body:       io.NopCloser(bytes.NewReader(converted)),
	}, nil
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

func codexResponsesSSEToChatSSE(body []byte, requestedModel string) ([]byte, error) {
	events := codexParseResponsesSSE(body)
	created := time.Now().Unix()
	id := codexChatID("")
	model := requestedModel
	roleSent := false
	finalized := false
	var out bytes.Buffer

	writeChunk := func(chunk codexChatChunk) error {
		data, err := json.Marshal(chunk)
		if err != nil {
			return err
		}
		_, err = out.WriteString("data: " + string(data) + "\n\n")
		return err
	}
	writeRole := func() error {
		if roleSent {
			return nil
		}
		roleSent = true
		return writeChunk(codexChatChunk{
			ID:      id,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   model,
			Choices: []codexChunkChoice{{Index: 0, Delta: codexChatDelta{Role: "assistant"}}},
		})
	}

	for _, event := range events {
		if event.Response != nil {
			if event.Response.ID != "" {
				id = codexChatID(event.Response.ID)
			}
			if event.Response.Model != "" {
				model = event.Response.Model
			}
		}
		switch event.Type {
		case "response.created":
			if err := writeRole(); err != nil {
				return nil, err
			}
		case "response.output_text.delta":
			if event.Delta == "" {
				continue
			}
			if err := writeRole(); err != nil {
				return nil, err
			}
			content := event.Delta
			if err := writeChunk(codexChatChunk{
				ID:      id,
				Object:  "chat.completion.chunk",
				Created: created,
				Model:   model,
				Choices: []codexChunkChoice{{Index: 0, Delta: codexChatDelta{Content: &content}}},
			}); err != nil {
				return nil, err
			}
		case "response.completed", "response.done", "response.incomplete", "response.failed":
			if err := writeRole(); err != nil {
				return nil, err
			}
			finishReason := codexFinishReason(event.Response)
			if err := writeChunk(codexChatChunk{
				ID:      id,
				Object:  "chat.completion.chunk",
				Created: created,
				Model:   model,
				Choices: []codexChunkChoice{{Index: 0, Delta: codexChatDelta{}, FinishReason: &finishReason}},
			}); err != nil {
				return nil, err
			}
			finalized = true
		}
	}
	if !finalized {
		if err := writeRole(); err != nil {
			return nil, err
		}
		finishReason := "stop"
		if err := writeChunk(codexChatChunk{
			ID:      id,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   model,
			Choices: []codexChunkChoice{{Index: 0, Delta: codexChatDelta{}, FinishReason: &finishReason}},
		}); err != nil {
			return nil, err
		}
	}
	_, _ = out.WriteString("data: [DONE]\n\n")
	return out.Bytes(), nil
}

func codexParseResponsesSSE(body []byte) []codexResponsesEvent {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), maxProxyRequestBodyBytes)
	events := []codexResponsesEvent{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var event codexResponsesEvent
		if err := json.Unmarshal([]byte(data), &event); err == nil && event.Type != "" {
			events = append(events, event)
		}
	}
	return events
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
