package proxy

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/klauspost/compress/zstd"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseTokenConsumptionFromSSE(t *testing.T) {
	body := []byte(strings.Join([]string{
		`event: response.output_text.delta`,
		`data: {"type":"response.output_text.delta","delta":"hello"}`,
		``,
		`event: response.completed`,
		`data: {"type":"response.completed","response":{"model":"gpt-5.5","usage":{"input_tokens":20,"output_tokens":8,"total_tokens":28,"input_tokens_details":{"cached_tokens":12},"cache_creation_input_tokens":3}}}`,
		``,
		`data: [DONE]`,
	}, "\n"))

	usage := parseTokenConsumption(http.Header{"Content-Type": []string{"text/event-stream"}}, body)
	if usage.TotalTokens != 28 || usage.InputTokens != 20 || usage.OutputTokens != 8 {
		t.Fatalf("unexpected usage: %#v", usage)
	}
	if usage.CacheReadTokens != 12 || usage.CacheCreationTokens != 3 {
		t.Fatalf("unexpected cache usage: %#v", usage)
	}
	if model := parseResponseModel(http.Header{"Content-Type": []string{"text/event-stream"}}, body); model != "gpt-5.5" {
		t.Fatalf("expected response model gpt-5.5, got %q", model)
	}
}

func TestCompletedRequestModelLeavesModelListUnlabelled(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/codex/v1/models?client_version=0.145.0", nil)
	body := []byte(`{"data":[{"model":"gpt-5.6-terra"}]}`)

	if model := completedRequestModel(req, "gpt-5.5", http.Header{"Content-Type": []string{"application/json"}}, body); model != "" {
		t.Fatalf("expected model list response to have no recorded model, got %q", model)
	}
}

func stringsReader(value string) io.Reader {
	return strings.NewReader(value)
}

func TestReadProxyRequestBodyDecodesZstd(t *testing.T) {
	var compressed bytes.Buffer
	encoder, err := zstd.NewWriter(&compressed)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"model":"gpt-5.5","input":"hi"}`)
	if _, err := encoder.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := encoder.Close(); err != nil {
		t.Fatal(err)
	}

	body, decoded, err := readProxyRequestBody(io.NopCloser(bytes.NewReader(compressed.Bytes())), "zstd")
	if err != nil {
		t.Fatal(err)
	}
	if !decoded {
		t.Fatal("expected zstd request body to be decoded")
	}
	if !bytes.Equal(body, raw) {
		t.Fatalf("unexpected decoded body: %q", string(body))
	}
}

type repeatingReader struct{}

func (repeatingReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	return len(p), nil
}

func TestReadProxyResponseBodyRejectsOversizedPayload(t *testing.T) {
	_, err := readProxyResponseBody(io.LimitReader(repeatingReader{}, maxProxyResponseBodyBytes+1))
	if !errors.Is(err, errResponseBodyTooLarge) {
		t.Fatalf("expected oversized response error, got %v", err)
	}
}

func codexAuthJSONForServiceTest(t *testing.T, email string) string {
	return codexAuthJSONForServiceTestWithCredentials(t, email, "account-123", "codex-access-token")
}

func codexAuthJSONForServiceTestWithCredentials(t *testing.T, email string, accountID string, accessToken string) string {
	t.Helper()

	payload, err := json.Marshal(map[string]any{
		"https://api.openai.com/profile": map[string]string{"email": email},
	})
	if err != nil {
		t.Fatal(err)
	}
	idToken := "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
	authJSON, err := json.Marshal(map[string]any{
		"tokens": map[string]string{
			"access_token": accessToken,
			"account_id":   accountID,
			"id_token":     idToken,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(authJSON)
}
