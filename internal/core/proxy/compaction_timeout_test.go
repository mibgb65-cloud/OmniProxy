package proxy

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCompactionUsesDedicatedDeadlineAndNormalRequestsStayBounded(t *testing.T) {
	for _, tc := range []struct {
		name, path, metadata string
		compact              bool
	}{
		{"normal", "/v1/responses", "", false},
		{"feature_only", "/v1/responses", `{}`, false},
		{"ordinary_turn", "/v1/responses", `{"request_kind":"inference"}`, false},
		{"v2", "/v1/responses", `{"compaction":{"implementation":"responses_compaction_v2"}}`, true},
		{"legacy", "/v1/responses/compact", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			finished := make(chan RequestEvent, 1)
			server := New(Options{Port: 0, Models: []string{"gpt-test"}, RequestTimeout: time.Second,
				Observer: firstTokenObserver{finished},
				Upstream: upstreamFunc(func(ctx context.Context, r Request, consume func(Response) error) error {
					deadline, ok := ctx.Deadline()
					if !ok || tc.compact && time.Until(deadline) < 5*time.Minute || !tc.compact && time.Until(deadline) > time.Second {
						t.Error("wrong request deadline", tc.name)
					}
					body := `{"output":[]}`
					if r.Compaction {
						body = compactResult
					}
					return consume(Response{StatusCode: 200, ContentType: "application/json", Body: strings.NewReader(body)})
				}),
			})
			if _, err := server.Start(); err != nil {
				t.Fatal(err)
			}
			defer server.Stop(context.Background())
			req, err := http.NewRequest(http.MethodPost, server.Status().Address+tc.path, strings.NewReader(`{"model":"gpt-test","input":[{"role":"user","content":"QA"}]}`))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			if tc.metadata != "" {
				req.Header.Set("X-Codex-Beta-Features", "remote_compaction_v2")
				req.Header.Set("X-Codex-Turn-Metadata", tc.metadata)
			}
			response, err := (&http.Client{Timeout: 2 * time.Second}).Do(req)
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			if response.StatusCode != 200 {
				t.Fatal("request failed", response.StatusCode)
			}
			select {
			case event := <-finished:
				want := "responses"
				if tc.compact {
					want = "responses/compact"
				}
				if event.Protocol != want {
					t.Fatal("compaction cannot be distinguished in retained records", event.Protocol)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("missing accounting event")
			}
		})
	}
}
