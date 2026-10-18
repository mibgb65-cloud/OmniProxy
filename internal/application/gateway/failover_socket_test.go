package gateway

import (
	"context"
	"github.com/coder/websocket"
	"net/http"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/provider/chatgpt"
	"strings"
	"testing"
	"time"
)

type failoverSubscription struct{}

func (failoverSubscription) UseToken(_ context.Context, id string, consume func([]byte, string) error) error {
	return consume([]byte("placeholder-subscription-"+id), "remote-"+id)
}

func TestCodexSocketFailoverBeforeEventsOnly(t *testing.T) {
	for _, broken := range []bool{false, true} {
		t.Run(map[bool]string{false: "reject-then-success", true: "no-replay-after-created"}[broken], func(t *testing.T) {
			s, db, observer, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("wrong adapter"); return nil, nil }))
			_, err := db.Exec(`UPDATE accounts SET provider_id='builtin_openai',auth_method='subscription_login',auth_email=id||'@example.invalid',auth_account_id='workspace',auth_state='authenticated'; INSERT INTO account_models(account_id,model,name,fetched_at,account_version) SELECT id,'gpt-test','GPT','2026-01-01T00:00:00Z',version FROM account_catalog_versions`)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			s.SetSubscription(failoverSubscription{})
			s.chatgpt = chatgpt.New(roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 && !broken {
					return upstreamReply(429, `{"error":{}}`, false), nil
				}
				body := "data: {\"type\":\"response.created\",\"response\":{\"id\":\"one\"}}\n\n"
				if !broken {
					body += "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"one\",\"output\":[],\"usage\":{\"input_tokens\":12,\"output_tokens\":4}}}\n\n"
				}
				return upstreamReply(200, body, true), nil
			}))
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			socket, _, err := websocket.Dial(ctx, strings.Replace(s.Info().Runtime.Address, "http:", "ws:", 1)+"/v1/responses", nil)
			if err != nil {
				t.Fatal(err)
			}
			defer socket.CloseNow()
			if err = socket.Write(ctx, websocket.MessageText, []byte(`{"type":"response.create","model":"gpt-test","input":[{"role":"user","content":"Hi"}]}`)); err != nil {
				t.Fatal(err)
			}
			completed, failed := false, false
			for !completed && !failed {
				_, raw, readErr := socket.Read(ctx)
				if readErr != nil {
					t.Fatal(readErr)
				}
				completed = strings.Contains(string(raw), `"response.completed"`)
				failed = strings.Contains(string(raw), `"type":"error"`)
			}
			if broken && (calls != 1 || !failed) || !broken && (calls != 2 || !completed) {
				t.Fatal("wrong socket attempts", calls, completed, failed)
			}
			page, err := observer.Requests(ctx, activity.RequestQuery{Limit: 10})
			if err != nil || len(page.Items) != 1 {
				t.Fatal("request lost", err)
			}
			item, err := observer.Request(ctx, page.Items[0].ID)
			if err != nil || item.AttemptCount != calls {
				t.Fatal("attempts lost", item, err)
			}
			if broken && (item.State != "failed" || item.Attempts[0].State != "failed") {
				t.Fatal("incomplete socket counted as success", item)
			}
		})
	}
}
