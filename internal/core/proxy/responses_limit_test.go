package proxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResponsesImageCapacityRemainsBounded(t *testing.T) {
	if requestBodyLimit("/v1/responses") != MaxResponsesRequestBytes || requestBodyLimit("/v1/chat/completions") != MaxRequestBytes {
		t.Fatal("protocol limits changed unexpectedly")
	}
	base := `{"model":"gpt-test","input":[]}`
	within := []byte(base + strings.Repeat(" ", MaxResponsesRequestBytes-len(base)))
	if _, err := ParseProtocolRequest(within, "/v1/responses"); err != nil {
		t.Fatal("Responses boundary rejected", err)
	}
	oversized := append(within, ' ')
	if _, err := ParseProtocolRequest(oversized, "/v1/responses"); err == nil {
		t.Fatal("oversized Responses JSON accepted")
	}
	server := New(Options{})
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:0/v1/responses", strings.NewReader(string(oversized)))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, req)
	if response.Code != http.StatusRequestEntityTooLarge || !strings.Contains(response.Body.String(), "request_too_large") {
		t.Fatal("HTTP Responses boundary not enforced", response.Code)
	}
}

func TestLargeResponsesStillRejectDuplicateFields(t *testing.T) {
	body := []byte(`{"model":"gpt-test","stream":true,"stream":false,"input":"` + strings.Repeat("x", 3<<20) + `"}`)
	if _, err := ParseProtocolRequest(body, "/v1/responses"); err == nil {
		t.Fatal("larger body bypassed duplicate-field protection")
	}
}
