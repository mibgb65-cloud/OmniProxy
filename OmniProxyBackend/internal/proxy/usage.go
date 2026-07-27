package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"omniproxy/internal/token"
)

const maxUsageCaptureBytes = 2 * 1024 * 1024

type usageCapture struct {
	buf bytes.Buffer
}

func (c *usageCapture) Write(p []byte) (int, error) {
	if c.buf.Len() < maxUsageCaptureBytes {
		remaining := maxUsageCaptureBytes - c.buf.Len()
		if len(p) > remaining {
			_, _ = c.buf.Write(p[:remaining])
		} else {
			_, _ = c.buf.Write(p)
		}
	}
	return len(p), nil
}

func (c *usageCapture) Bytes() []byte {
	return c.buf.Bytes()
}

func parseTokenConsumption(header http.Header, body []byte) token.TokenConsumption {
	if len(body) == 0 {
		return token.TokenConsumption{}
	}

	contentType := strings.ToLower(header.Get("Content-Type"))
	if strings.Contains(contentType, "text/event-stream") || bytes.Contains(body, []byte("data:")) {
		if usage, ok := parseSSETokenConsumption(body); ok {
			return usage
		}
	}

	usage, _ := parseJSONTokenConsumption(body)
	return usage
}

func parseResponseModel(header http.Header, body []byte) string {
	if len(body) == 0 {
		return ""
	}

	contentType := strings.ToLower(header.Get("Content-Type"))
	if strings.Contains(contentType, "text/event-stream") || bytes.Contains(body, []byte("data:")) {
		if model := parseSSEResponseModel(body); model != "" {
			return model
		}
	}

	model, _ := parseJSONResponseModel(body)
	return model
}

func completedRequestModel(r *http.Request, routedModel string, header http.Header, body []byte) string {
	if r != nil && r.Method == http.MethodGet && r.URL != nil {
		path := strings.TrimRight(strings.ToLower(r.URL.Path), "/")
		if strings.HasSuffix(path, "/models") {
			return ""
		}
	}
	if model := parseResponseModel(header, body); model != "" {
		return model
	}
	return strings.TrimSpace(routedModel)
}

func parseSSEResponseModel(body []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), maxUsageCaptureBytes)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		if model, ok := parseJSONResponseModel([]byte(data)); ok {
			return model
		}
	}
	return ""
}

func parseJSONResponseModel(body []byte) (string, bool) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	var payload any
	if err := decoder.Decode(&payload); err != nil {
		return "", false
	}
	return findResponseModel(payload)
}

func findResponseModel(value any) (string, bool) {
	switch typed := value.(type) {
	case map[string]any:
		if model, ok := stringFromAny(typed["model"]); ok {
			return model, true
		}
		if response, ok := typed["response"]; ok {
			if model, modelOK := findResponseModel(response); modelOK {
				return model, true
			}
		}
		for _, child := range typed {
			if model, ok := findResponseModel(child); ok {
				return model, true
			}
		}
	case []any:
		for _, child := range typed {
			if model, ok := findResponseModel(child); ok {
				return model, true
			}
		}
	}
	return "", false
}

func parseSSETokenConsumption(body []byte) (token.TokenConsumption, bool) {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), maxUsageCaptureBytes)

	var found token.TokenConsumption
	ok := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		if usage, usageOK := parseJSONTokenConsumption([]byte(data)); usageOK {
			found = usage
			ok = true
		}
	}
	return found, ok
}

func stringFromAny(value any) (string, bool) {
	text, ok := value.(string)
	if !ok {
		return "", false
	}
	text = strings.TrimSpace(text)
	return text, text != ""
}

func parseJSONTokenConsumption(body []byte) (token.TokenConsumption, bool) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	var payload any
	if err := decoder.Decode(&payload); err != nil {
		return token.TokenConsumption{}, false
	}
	return findTokenConsumption(payload)
}

func findTokenConsumption(value any) (token.TokenConsumption, bool) {
	switch typed := value.(type) {
	case map[string]any:
		if usageValue, ok := typed["usage"]; ok {
			if usage, usageOK := consumeUsageObject(usageValue); usageOK {
				return usage, true
			}
			if usage, usageOK := findTokenConsumption(usageValue); usageOK {
				return usage, true
			}
		}
		if usage, ok := consumeUsageMap(typed); ok {
			return usage, true
		}

		var found token.TokenConsumption
		ok := false
		for _, child := range typed {
			if usage, usageOK := findTokenConsumption(child); usageOK {
				found = usage
				ok = true
			}
		}
		return found, ok
	case []any:
		var found token.TokenConsumption
		ok := false
		for _, child := range typed {
			if usage, usageOK := findTokenConsumption(child); usageOK {
				found = usage
				ok = true
			}
		}
		return found, ok
	default:
		return token.TokenConsumption{}, false
	}
}

func consumeUsageObject(value any) (token.TokenConsumption, bool) {
	usageMap, ok := value.(map[string]any)
	if !ok {
		return token.TokenConsumption{}, false
	}
	return consumeUsageMap(usageMap)
}

func consumeUsageMap(value map[string]any) (token.TokenConsumption, bool) {
	input := intFromKeys(value, "input_tokens", "prompt_tokens", "promptTokenCount")
	output := intFromKeys(value, "output_tokens", "completion_tokens", "candidatesTokenCount")
	total := intFromKeys(value, "total_tokens", "totalTokenCount")
	cacheCreation := cacheCreationTokensFromUsage(value)
	cacheRead := cacheReadTokensFromUsage(value)
	if total == 0 && (input > 0 || output > 0) {
		total = input + output
	}
	if total == 0 && (cacheCreation > 0 || cacheRead > 0) {
		total = cacheCreation + cacheRead
	}
	if input == 0 && output == 0 && total == 0 && cacheCreation == 0 && cacheRead == 0 {
		return token.TokenConsumption{}, false
	}
	return token.TokenConsumption{
		InputTokens:         input,
		OutputTokens:        output,
		TotalTokens:         total,
		CacheCreationTokens: cacheCreation,
		CacheReadTokens:     cacheRead,
	}, true
}

func cacheCreationTokensFromUsage(value map[string]any) int {
	total := intFromKeys(value, "cache_creation_input_tokens", "cache_creation_tokens")
	if cacheCreation, ok := mapFromAny(value["cache_creation"]); ok {
		total += sumIntFromKeys(cacheCreation, "ephemeral_5m_input_tokens", "ephemeral_1h_input_tokens")
	}
	return total
}

func cacheReadTokensFromUsage(value map[string]any) int {
	if parsed := intFromKeys(value, "cache_read_input_tokens", "cache_read_tokens"); parsed > 0 {
		return parsed
	}
	for _, key := range []string{"input_tokens_details", "prompt_tokens_details", "inputTokenDetails", "promptTokenDetails"} {
		if details, ok := mapFromAny(value[key]); ok {
			if parsed := intFromKeys(details, "cached_tokens", "cachedTokens"); parsed > 0 {
				return parsed
			}
		}
	}
	return 0
}

func intFromKeys(value map[string]any, keys ...string) int {
	for _, key := range keys {
		if parsed, ok := intFromAny(value[key]); ok {
			return parsed
		}
	}
	return 0
}

func sumIntFromKeys(value map[string]any, keys ...string) int {
	total := 0
	for _, key := range keys {
		if parsed, ok := intFromAny(value[key]); ok {
			total += parsed
		}
	}
	return total
}

func mapFromAny(value any) (map[string]any, bool) {
	typed, ok := value.(map[string]any)
	return typed, ok
}

func intFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil && parsed >= 0 {
			return int(parsed), true
		}
	case float64:
		if typed >= 0 {
			return int(typed), true
		}
	case int:
		if typed >= 0 {
			return typed, true
		}
	case int64:
		if typed >= 0 {
			return int(typed), true
		}
	}
	return 0, false
}
