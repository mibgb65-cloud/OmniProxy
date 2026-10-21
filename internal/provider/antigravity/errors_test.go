package antigravity

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

func TestRequestErrorClassification(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		status   int
		code     string
		wantHTTP int
	}{
		{"inference_bad_request", inferenceURL + "/v1internal:generateContent", 400, "invalid_request", 400},
		{"stream_bad_request", inferenceURL + "/v1internal:streamGenerateContent?alt=sse", 400, "invalid_request", 400},
		{"catalog_bad_request", inferenceURL + "/v1internal:fetchAvailableModels", 400, "invalid_request", 400},
		{"control_bad_request", controlURL + "/v1internal:loadCodeAssist", 400, "invalid_request", 400},
		{"identity_bad_request", userURL, 400, "invalid_request", 400},
		{"token_bad_request", tokenURL, 400, "reauth_required", 401},
		{"unauthorized", inferenceURL + "/v1internal:generateContent", 401, "reauth_required", 401},
		{"token_unauthorized", tokenURL, 401, "reauth_required", 401},
		{"forbidden", inferenceURL + "/v1internal:generateContent", 403, "access_denied", 403},
		{"rate_limited", inferenceURL + "/v1internal:generateContent", 429, "rate_limited", 429},
		{"server_error", inferenceURL + "/v1internal:generateContent", 500, "upstream_unavailable", 502},
		{"redirect", tokenURL, 302, "upstream_unavailable", 502},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			client := New(transportFunc(func(*http.Request) (*http.Response, error) {
				calls++
				return reply(tt.status, `{"error":"private-upstream-response"}`), nil
			}))
			response, err := client.request(context.Background(), tt.endpoint, []byte(`{}`), "private-access-token", "application/json")
			if response != nil || err == nil || ErrorCode(err) != tt.code {
				t.Fatalf("unexpected classification: response=%v error=%v", response != nil, err)
			}
			var failure *proxy.Error
			if !errors.As(ProxyError(err), &failure) || failure.StatusCode != tt.wantHTTP || failure.Code != tt.code {
				t.Fatalf("unexpected proxy error: %v", ProxyError(err))
			}
			if calls != 1 || strings.Contains(err.Error(), "private-") {
				t.Fatal("request retried or exposed upstream data")
			}
		})
	}
}
