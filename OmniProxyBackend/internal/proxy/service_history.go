package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"omniproxy/internal/history"
	"omniproxy/internal/logs"
	"omniproxy/internal/token"
	"strconv"
	"strings"
	"sync"
	"time"
)

func (s *Service) writeResponse(w http.ResponseWriter, resp *http.Response) (token.TokenConsumption, []byte) {
	defer closeBody(resp.Body)

	capture := &usageCapture{}
	removeHopHeaders(resp.Header)
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	target := io.Writer(w)
	if flusher, ok := w.(http.Flusher); ok {
		target = flushWriter{writer: w, flusher: flusher}
	}
	_, _ = io.Copy(io.MultiWriter(target, capture), resp.Body)
	body := capture.Bytes()
	return parseTokenConsumption(resp.Header, body), body
}

func parseRemaining(header http.Header) (int, bool) {
	keys := []string{
		"x-ratelimit-remaining-tokens",
		"x-ratelimit-remaining",
		"x-ratelimit-remaining-requests",
	}
	for _, key := range keys {
		value := strings.TrimSpace(header.Get(key))
		if value == "" {
			continue
		}
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func levelForStatus(status int) logs.Level {
	if status >= 500 {
		return logs.LevelError
	}
	if status >= 400 {
		return logs.LevelWarn
	}
	return logs.LevelInfo
}

func (s *Service) recordHistory(r *http.Request, route routeInfo, selected *token.Token, status int, duration int64, consumption token.TokenConsumption, level logs.Level, message string, retryChain ...history.RetryAttempt) {
	if s.history == nil {
		return
	}
	entry := history.Entry{
		Level:             string(level),
		Method:            r.Method,
		Path:              r.URL.RequestURI(),
		Provider:          token.NormalizeProvider(route.Provider),
		Protocol:          route.Protocol,
		ClientKey:         route.ClientKey,
		ClientName:        route.ClientName,
		Model:             route.Model,
		Status:            status,
		Duration:          duration,
		InputTokens:       consumption.InputTokens,
		OutputTokens:      consumption.OutputTokens,
		TotalTokens:       consumption.TotalTokens,
		CooldownTriggered: retryChainCooldownTriggered(retryChain),
		Message:           message,
	}
	if entry.Protocol == "" {
		entry.Protocol = "openai"
	}
	if selected != nil {
		entry.TokenID = selected.ID
		entry.TokenName = token.DisplayName(*selected)
	}
	if len(retryChain) > 0 {
		entry.RetryChain = append([]history.RetryAttempt(nil), retryChain...)
	}
	s.history.Add(entry)
}

func (s *Service) beginActiveRequest(r *http.Request, route routeInfo, selected token.Token) (func(), func(string)) {
	s.activeMu.Lock()

	if s.activeRequests == nil {
		s.activeRequests = map[int64]ActiveRequest{}
	}
	s.activeSeq++
	id := s.activeSeq
	path := ""
	method := ""
	if r != nil {
		method = r.Method
		if r.URL != nil {
			path = r.URL.RequestURI()
		}
	}
	entry := ActiveRequest{
		ID:         id,
		StartedAt:  time.Now(),
		ClientKey:  route.ClientKey,
		ClientName: route.ClientName,
		Method:     method,
		Path:       path,
		Provider:   token.NormalizeProvider(route.Provider),
		Protocol:   route.Protocol,
		Model:      route.Model,
		TokenID:    selected.ID,
		TokenName:  token.DisplayName(selected),
	}
	s.activeRequests[id] = entry
	s.activeMu.Unlock()
	if s.activity != nil {
		s.activity.ActiveRequestStarted(entry)
	}
	var once sync.Once
	finish := func() {
		once.Do(func() {
			s.activeMu.Lock()
			finished := entry
			if current, ok := s.activeRequests[id]; ok {
				finished = current
			}
			delete(s.activeRequests, id)
			s.activeMu.Unlock()
			if s.activity != nil {
				s.activity.ActiveRequestFinished(finished)
			}
		})
	}
	updateModel := func(model string) {
		model = strings.TrimSpace(model)
		if model == "" {
			return
		}
		s.activeMu.Lock()
		if current, ok := s.activeRequests[id]; ok {
			current.Model = model
			s.activeRequests[id] = current
		}
		s.activeMu.Unlock()
	}
	return finish, updateModel
}

func appendRetryAttempt(chain []history.RetryAttempt, attempt int, route routeInfo, selected *token.Token, status int, duration int64, cooldownTriggered bool, message string) []history.RetryAttempt {
	item := history.RetryAttempt{
		Attempt:           attempt,
		Provider:          token.NormalizeProvider(route.Provider),
		Protocol:          route.Protocol,
		Model:             route.Model,
		Status:            status,
		Duration:          duration,
		CooldownTriggered: cooldownTriggered,
		Message:           strings.TrimSpace(message),
	}
	if item.Protocol == "" {
		item.Protocol = "openai"
	}
	if selected != nil {
		item.TokenID = selected.ID
		item.TokenName = token.DisplayName(*selected)
	}
	return append(chain, item)
}

func retryChainCooldownTriggered(chain []history.RetryAttempt) bool {
	for _, attempt := range chain {
		if attempt.CooldownTriggered {
			return true
		}
	}
	return false
}

func proxyHistoryMessage(status int, model string, consumption token.TokenConsumption, fallback string, body []byte) string {
	if status >= 400 {
		text := fmt.Sprintf("upstream returned %d", status)
		if statusText := http.StatusText(status); statusText != "" {
			text = fmt.Sprintf("%s %s", text, statusText)
		}
		if summary := upstreamErrorSummary(body); summary != "" {
			text = summary
		}
		return proxyLogMessage(model, consumption, text)
	}
	return proxyLogMessage(model, consumption, fallback)
}

func upstreamErrorSummary(body []byte) string {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return ""
	}

	var payload any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err == nil {
		if message := findErrorSummary(payload); message != "" {
			return message
		}
	}
	return limitSummary(strings.Join(strings.Fields(string(body)), " "))
}

func findErrorSummary(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range []string{"message", "detail", "error_description"} {
			if message, ok := typed[key].(string); ok && strings.TrimSpace(message) != "" {
				return limitSummary(message)
			}
		}
		if errorValue, ok := typed["error"]; ok {
			if message, ok := errorValue.(string); ok && strings.TrimSpace(message) != "" {
				return limitSummary(message)
			}
			if message := findErrorSummary(errorValue); message != "" {
				return message
			}
		}
		for _, child := range typed {
			if message := findErrorSummary(child); message != "" {
				return message
			}
		}
	case []any:
		for _, child := range typed {
			if message := findErrorSummary(child); message != "" {
				return message
			}
		}
	}
	return ""
}

func limitSummary(value string) string {
	value = strings.TrimSpace(value)
	const max = 320
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}

func proxyLogMessage(_ string, consumption token.TokenConsumption, fallback string) string {
	message := fallback
	if consumption.TotalTokens <= 0 {
		return message
	}
	return fmt.Sprintf("%s, used %d tokens", message, consumption.TotalTokens)
}
