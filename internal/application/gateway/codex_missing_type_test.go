package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/provider/chatgpt"
)

const missingTypeStream = "data: {\"type\":\"response.created\",\"response\":{\"id\":\"one\"}}\n\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"OK\"}\n\ndata: {\"type\":\"response.completed\",\"response\":{\"id\":\"one\",\"output\":[],\"usage\":{\"input_tokens\":12,\"output_tokens\":4}}}\n\n"

func TestCodexMissingContentTypeThroughGateway(t *testing.T) {
	for _, transport := range []string{"http", "websocket"} {
		for _, scenario := range []string{"valid-sse", "html", "json", "explicit-html", "truncated-sse", "nested-error"} {
			t.Run(transport+"/"+scenario, func(t *testing.T) {
				s, db, observer, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("wrong adapter"); return nil, nil }))
				_, err := db.Exec(`UPDATE accounts SET provider_id='builtin_openai',auth_method='subscription_login',auth_email=id||'@example.invalid',auth_account_id='workspace',auth_state='authenticated'; INSERT INTO account_models(account_id,model,name,fetched_at,account_version) SELECT id,'gpt-test','GPT','2026-01-01T00:00:00Z',version FROM account_catalog_versions`)
				if err != nil {
					t.Fatal(err)
				}
				s.SetSubscription(failoverSubscription{})
				var calls atomic.Int32
				s.chatgpt = chatgpt.New(roundTripFunc(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					body, kind := missingTypeStream, ""
					switch scenario {
					case "html":
						body = "<!doctype html><html>private-upstream-marker</html>"
					case "json":
						body = `{"error":{"message":"private-upstream-marker"}}`
					case "explicit-html":
						kind = "text/html"
					case "nested-error":
						body = strings.SplitAfter(missingTypeStream, "\n\n")[0] + "data: {\"type\":\"response.failed\",\"response\":{\"id\":\"one\",\"error\":{\"message\":\"private-upstream-marker\"}}}\n\n"
					case "truncated-sse":
						body = strings.SplitAfter(missingTypeStream, "\n\n")[0]
					}
					return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {kind}}, Body: io.NopCloser(strings.NewReader(body))}, nil
				}))
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				address := s.Info().Runtime.Address + "/v1/responses"
				failed, completed := false, false
				if transport == "http" {
					resp, e := http.Post(address, "application/json", strings.NewReader(`{"model":"gpt-test","stream":true,"input":[]}`))
					if e != nil {
						t.Fatal(e)
					}
					raw, readErr := io.ReadAll(resp.Body)
					resp.Body.Close()
					completed = strings.Contains(string(raw), `"response.completed"`)
					failed = resp.StatusCode == 502 || readErr != nil
					if strings.Contains(string(raw), "private-upstream-marker") {
						t.Fatal("upstream body leaked")
					}
				} else {
					socket, _, e := websocket.Dial(ctx, strings.Replace(address, "http:", "ws:", 1), nil)
					if e != nil {
						t.Fatal(e)
					}
					defer socket.CloseNow()
					if e = socket.Write(ctx, websocket.MessageText, []byte(`{"type":"response.create","model":"gpt-test","input":[]}`)); e != nil {
						t.Fatal(e)
					}
					for !failed && !completed {
						_, raw, e := socket.Read(ctx)
						if e != nil {
							t.Fatal(e)
						}
						var event struct {
							Type   string
							Status int
						}
						if json.Unmarshal(raw, &event) != nil {
							t.Fatal("invalid downstream event")
						}
						completed = event.Type == "response.completed"
						failed = event.Type == "error" && event.Status == 502
						if strings.Contains(string(raw), "private-upstream-marker") {
							t.Fatal("upstream body leaked")
						}
					}
				}
				wantSuccess := scenario == "valid-sse"
				if completed != wantSuccess || failed == wantSuccess || calls.Load() != 1 {
					t.Fatal("incorrect result or replay", completed, failed, calls.Load())
				}
				var item activity.RequestItem
				deadline := time.Now().Add(2 * time.Second)
				for {
					page, e := observer.Requests(ctx, activity.RequestQuery{Limit: 10})
					if e != nil || len(page.Items) != 1 {
						t.Fatal("request missing", e)
					}
					item, e = observer.Request(ctx, page.Items[0].ID)
					if e != nil {
						t.Fatal(e)
					}
					if item.State != "pending" || time.Now().After(deadline) {
						break
					}
					time.Sleep(time.Millisecond)
				}
				if len(item.Attempts) != 1 || item.Attempts[0].StatusCode != 200 {
					t.Fatal("upstream status lost", item.Attempts)
				}
				if wantSuccess {
					if item.State != "success" || item.StatusCode != 200 || !item.UsageComplete || item.InputTokens != "12" || item.OutputTokens != "4" || item.FirstTokenMS == nil {
						t.Fatal("usage, first token or completion lost", item)
					}
				} else if item.State != "failed" {
					t.Fatal("malformed response counted as success", item.State)
				}
			})
		}
	}
}
