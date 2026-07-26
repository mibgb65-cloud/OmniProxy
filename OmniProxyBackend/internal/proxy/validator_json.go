package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"omniproxy/internal/token"
	"strconv"
	"strings"
	"time"
)

func (v *Validator) queryJSON(ctx context.Context, selected token.Token, target string) ([]byte, bool) {
	return v.queryJSONWithHeaders(ctx, selected, target, nil)
}

func (v *Validator) queryJSONWithHeaders(ctx context.Context, selected token.Token, target string, extra map[string]string) ([]byte, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, false
	}
	req.Header.Set("Accept", "application/json")
	if err := applyAuth(req.Header, selected); err != nil {
		return nil, false
	}
	for key, value := range extra {
		req.Header.Set(key, value)
	}

	resp, err := v.clientForToken(selected).Do(req)
	if err != nil {
		return nil, false
	}
	defer closeBody(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, false
	}
	return body, true
}

func usageWindowFromLimit(value map[string]any) (int, int, int64, bool) {
	limit, ok := floatFromAny(value["limit"])
	if !ok || limit <= 0 {
		return 0, 0, 0, false
	}
	remainingValue, ok := floatFromAny(value["remaining"])
	if !ok {
		return 0, 0, 0, false
	}
	used := percent(((limit - remainingValue) / limit) * 100)
	remaining := 100 - used
	if remaining < 0 {
		remaining = 0
	}
	resetAt := unixSecondsFromAny(value["resetTime"])
	return used, remaining, resetAt, true
}

func joinURLPath(baseURL string, path string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	out := *base
	out.Path = singleJoiningSlash(base.Path, path)
	out.RawQuery = ""
	return out.String(), nil
}

func decodeObject(body []byte) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var payload map[string]any
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func floatFromAny(value any) (float64, bool) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func boolFromAny(value any, fallback bool) bool {
	typed, ok := value.(bool)
	if !ok {
		return fallback
	}
	return typed
}

func unixSecondsFromAny(value any) int64 {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil {
			return normalizeUnixSeconds(parsed)
		}
	case float64:
		return normalizeUnixSeconds(int64(typed))
	case int64:
		return normalizeUnixSeconds(typed)
	case int:
		return normalizeUnixSeconds(int64(typed))
	case string:
		text := strings.TrimSpace(typed)
		if parsed, err := strconv.ParseInt(text, 10, 64); err == nil {
			return normalizeUnixSeconds(parsed)
		}
		if parsed, err := time.Parse(time.RFC3339Nano, text); err == nil {
			return parsed.Unix()
		}
		for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04"} {
			if parsed, err := time.ParseInLocation(layout, text, time.Local); err == nil {
				return parsed.Unix()
			}
		}
	}
	return 0
}

func normalizeUnixSeconds(value int64) int64 {
	if value > 1_000_000_000_000 {
		return value / 1000
	}
	return value
}
