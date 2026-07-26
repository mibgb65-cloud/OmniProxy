package proxy

import (
	"bytes"
	"encoding/json"
	"strings"
)

func normalizeUpstreamModelID(model string) string {
	model = strings.TrimSpace(model)
	switch strings.ToLower(model) {
	case "deepseek-v4-pro[1m]":
		return "deepseek-v4-pro"
	default:
		return model
	}
}

func normalizeRequestBodyModel(body []byte, routeModel string) ([]byte, bool) {
	// This runs on every forward attempt, and agent payloads reach megabytes.
	// Decoding into map[string]any allocates a copy of the entire request, so
	// the model is probed on its own first and the full decode only happens
	// when the body actually has to be rewritten. A pointer distinguishes an
	// absent model from an empty one, matching the previous type assertion.
	var probe struct {
		Model *string `json:"model"`
	}
	if err := json.Unmarshal(body, &probe); err != nil || probe.Model == nil {
		return body, false
	}

	model := *probe.Model
	normalized := normalizeUpstreamModelID(model)
	target := normalizeUpstreamModelID(routeModel)
	if target == "" {
		target = normalized
	}
	if target == "" || target == strings.TrimSpace(model) {
		return body, false
	}

	var payload map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return body, false
	}
	payload["model"] = target
	updated, err := json.Marshal(payload)
	if err != nil {
		return body, false
	}
	return updated, true
}
