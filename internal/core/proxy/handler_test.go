package proxy

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNormalAndStreamingResponses(t *testing.T) {
	server := testServer(t, upstreamFunc(func(ctx context.Context, req Request, consume func(Response) error) error {
		if req.Stream {
			return consume(Response{StatusCode: 200, ContentType: "text/event-stream", Body: strings.NewReader("data: {\"choices\":[{\"delta\":{\"content\":\"OK\"}}]}\n\ndata: [DONE]\n\n")})
		}
		return successfulUpstream(ctx, req, consume)
	}), 0)
	status, body := request(t, server, validBody)
	if status != 200 || !strings.Contains(body, `"content":"OK"`) {
		t.Fatalf("non-stream: %d %s", status, body)
	}
	status, body = request(t, server, strings.TrimSuffix(validBody, "}")+`,"stream":true}`)
	if status != 200 || !strings.HasSuffix(body, "data: [DONE]\n\n") {
		t.Fatalf("stream: %d %s", status, body)
	}
}

func TestSafeUpstreamError(t *testing.T) {
	secret := "sk-placeholder-not-a-real-api-key"
	server := testServer(t, upstreamFunc(func(ctx context.Context, req Request, consume func(Response) error) error {
		return consume(Response{StatusCode: 401, ContentType: "application/json", Body: strings.NewReader(secret)})
	}), 0)
	status, body := request(t, server, validBody)
	if status != 401 || !strings.Contains(body, "upstream_authentication_failed") || strings.Contains(body, secret) {
		t.Fatalf("unsafe error: %d %s", status, body)
	}
}

func TestErrorEnvelopeWithStatusOKIsSanitized(t *testing.T) {
	secret := "placeholder-private-error-content"
	server := testServer(t, upstreamFunc(func(ctx context.Context, req Request, consume func(Response) error) error {
		return consume(Response{StatusCode: 200, ContentType: "application/json", Body: strings.NewReader(`{"error":{"message":"` + secret + `"}}`)})
	}), 0)
	status, body := request(t, server, validBody)
	if status != 502 || strings.Contains(body, secret) {
		t.Fatalf("unsafe error envelope: %d %s", status, body)
	}
}

func TestLocalRequestProtection(t *testing.T) {
	server := testServer(t, upstreamFunc(successfulUpstream), 0)
	for _, tc := range []struct{ name, host, origin, fetch string }{
		{name: "cross origin", origin: "https://attacker.example"},
		{name: "DNS rebinding", host: "attacker.example:3001"},
		{name: "cross site", fetch: "cross-site"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, server.Status().Address+"/v1/chat/completions", strings.NewReader(validBody))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			if tc.host != "" {
				req.Host = tc.host
			}
			req.Header.Set("Origin", tc.origin)
			req.Header.Set("Sec-Fetch-Site", tc.fetch)
			response, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			if response.StatusCode != 403 {
				t.Fatalf("expected 403, got %d", response.StatusCode)
			}
		})
	}
}

func TestInputAndRouteBoundaries(t *testing.T) {
	server := testServer(t, upstreamFunc(successfulUpstream), 0)
	for _, tc := range []struct {
		path, method, contentType, body string
		status                          int
	}{
		{"/v1/chat/completions", http.MethodPost, "application/json", strings.Repeat("a", MaxRequestBytes+1), 413},
		{"/v1/chat/completions", http.MethodPost, "text/plain", validBody, 415},
		{"/v1/chat/completions", http.MethodGet, "application/json", "", 405},
		{"/v1/chat/completions?target=attacker", http.MethodPost, "application/json", validBody, 400},
		{"/v1/responses", http.MethodPost, "application/json", validBody, 400},
		{"/v1/unknown", http.MethodPost, "application/json", validBody, 404},
	} {
		req, err := http.NewRequest(tc.method, server.Status().Address+tc.path, strings.NewReader(tc.body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", tc.contentType)
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, response.Body)
		response.Body.Close()
		if response.StatusCode != tc.status {
			t.Errorf("%s %s: got %d, want %d", tc.method, tc.path, response.StatusCode, tc.status)
		}
	}
}

func TestModelList(t *testing.T) {
	server := testServer(t, upstreamFunc(successfulUpstream), 0)
	response, err := http.Get(server.Status().Address + "/v1/models")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 || !strings.Contains(string(body), "deepseek-flash") {
		t.Fatalf("models: %d %s", response.StatusCode, body)
	}
}
