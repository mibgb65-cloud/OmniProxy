package proxy

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

const compactResult = `{"id":"cmp_test","object":"response.compaction","output":[{"type":"compaction","encrypted_content":"opaque-qa-only"}],"usage":{"input_tokens":20,"output_tokens":2}}`

func TestCompactionHTTPAndScopedPathsPreserveOpaqueResult(t *testing.T) {
	key := "opk_" + strings.Repeat("a", 64)
	calls := 0
	server := New(Options{Port: 0, Models: []string{"gpt-test"}, Authorize: func(_ context.Context, k string) bool { return k == key }, Upstream: upstreamFunc(func(_ context.Context, r Request, c func(Response) error) error {
		calls++
		if r.Protocol != "responses" || r.Stream || !r.Compaction {
			t.Error("compaction must stay non-streaming Responses")
		}
		return c(Response{StatusCode: 200, ContentType: "application/json", Body: strings.NewReader(compactResult)})
	})})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop(context.Background())
	for _, path := range []string{"/v1/responses/compact", "/agent/codex/" + key + "/v1/responses/compact"} {
		req, _ := http.NewRequest("POST", server.Status().Address+path, strings.NewReader(`{"model":"gpt-test","input":[]}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-OmniProxy-Key", key)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != 200 || string(raw) != compactResult {
			t.Fatal("compaction endpoint did not preserve the opaque JSON result", resp.StatusCode)
		}
	}
	if calls != 2 {
		t.Fatal("compaction was not forwarded exactly once per request")
	}
}

func TestCompactionRejectsInvalidBoundariesBeforeUpstream(t *testing.T) {
	key := "opk_" + strings.Repeat("a", 64)
	server := New(Options{Port: 0, Models: []string{"gpt-test"}, Authorize: func(_ context.Context, k string) bool { return k == key }, Upstream: upstreamFunc(func(context.Context, Request, func(Response) error) error {
		t.Error("invalid compaction reached upstream")
		return nil
	})})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop(context.Background())
	for _, test := range []struct {
		name, method, path, body, auth, kind string
		status                               int
	}{
		{"missing key", "POST", "/v1/responses/compact", `{"model":"gpt-test","input":[]}`, "", "application/json", 401},
		{"wrong method", "GET", "/v1/responses/compact", "", key, "application/json", 405},
		{"streaming", "POST", "/v1/responses/compact", `{"model":"gpt-test","input":[],"stream":true}`, key, "application/json", 400},
		{"missing input", "POST", "/v1/responses/compact", `{"model":"gpt-test"}`, key, "application/json", 400},
		{"null input", "POST", "/v1/responses/compact", `{"model":"gpt-test","input":null}`, key, "application/json", 400},
		{"text input", "POST", "/v1/responses/compact", `{"model":"gpt-test","input":"wrong-shape"}`, key, "application/json", 400},
		{"duplicate model", "POST", "/v1/responses/compact", `{"model":"gpt-test","model":"gpt-other","input":[]}`, key, "application/json", 400},
		{"wrong media type", "POST", "/v1/responses/compact", `{"model":"gpt-test","input":[]}`, key, "text/plain", 415},
		{"arbitrary scoped suffix", "POST", "/agent/codex/" + key + "/v1/responses/compact/extra", `{"model":"gpt-test","input":[]}`, key, "application/json", 404},
		{"arbitrary nested endpoint", "POST", "/agent/codex/" + key + "/v1/models/compact", `{"model":"gpt-test","input":[]}`, key, "application/json", 404},
	} {
		t.Run(test.name, func(t *testing.T) {
			req, _ := http.NewRequest(test.method, server.Status().Address+test.path, strings.NewReader(test.body))
			req.Header.Set("Content-Type", test.kind)
			req.Header.Set("X-OmniProxy-Key", test.auth)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != test.status {
				t.Fatal("unexpected boundary status", resp.StatusCode)
			}
		})
	}
}

func TestCompactionResultCannotBeReplacedByOrdinaryResponse(t *testing.T) {
	for _, body := range []string{`{"object":"response","output":[]}`, `{"object":"response.compaction","output":[]}`, `{"object":"response.compaction","output":[{"type":"compaction","encrypted_content":""}]}`, `{"object":"response.compaction","output":[{"type":"message","content":"not compressed"}]}`} {
		if validCompactionResult([]byte(body)) {
			t.Fatal("ordinary or empty output fabricated a compaction result")
		}
	}
	if requestBodyLimit("/v1/responses/compact") != MaxResponsesRequestBytes {
		t.Fatal("long context compaction received the smaller Chat limit")
	}
}
