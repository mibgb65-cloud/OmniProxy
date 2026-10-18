package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/provider/chatgpt"
)

func TestCodexSocketRejectedRequestAndCooldownAccounting(t *testing.T) {
	for _, status := range []int{400, 429} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			s, db, observer, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("wrong adapter"); return nil, nil }))
			_, err := db.Exec(`UPDATE accounts SET provider_id='builtin_openai',enabled=(id='a'),auth_method='subscription_login',auth_email=id||'@example.invalid',auth_account_id='workspace',auth_state='authenticated'; INSERT INTO account_models(account_id,model,name,fetched_at,account_version) SELECT id,'gpt-test','GPT','2026-01-01T00:00:00Z',version FROM account_catalog_versions`)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			s.SetSubscription(failoverSubscription{})
			s.chatgpt = chatgpt.New(roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				return upstreamReply(status, `{"error":{"message":"private-rejection"}}`, false), nil
			}))
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			for requestIndex := 0; requestIndex < 2; requestIndex++ {
				socket, _, err := websocket.Dial(ctx, strings.Replace(s.Info().Runtime.Address, "http:", "ws:", 1)+"/v1/responses", nil)
				if err != nil {
					t.Fatal(err)
				}
				defer socket.CloseNow()
				if err = socket.Write(ctx, websocket.MessageText, []byte(`{"type":"response.create","model":"gpt-test","input":[]}`)); err != nil {
					t.Fatal(err)
				}
				_, raw, err := socket.Read(ctx)
				if err != nil {
					t.Fatal(err)
				}
				var event struct {
					Status int
					Error  struct{ Code string }
				}
				if json.Unmarshal(raw, &event) != nil || event.Status != status {
					t.Fatal("socket status does not match upstream", event)
				}
				want := "upstream_rejected_request"
				if status == 429 {
					want = "upstream_rate_limited"
					if requestIndex == 1 {
						want = "account_pool_cooling"
					}
				}
				if event.Error.Code != want || strings.Contains(string(raw), "private-rejection") {
					t.Fatal("incorrect or unsafe socket code", event)
				}
			}
			page, err := observer.Requests(ctx, activity.RequestQuery{Limit: 10})
			if err != nil || len(page.Items) != 2 {
				t.Fatal("requests missing", err)
			}
			for _, row := range page.Items {
				item, err := observer.Request(ctx, row.ID)
				if err != nil || item.StatusCode != status || item.State != "failed" {
					t.Fatal("request status lost", item, err)
				}
				for _, attempt := range item.Attempts {
					if attempt.StatusCode != status {
						t.Fatal("attempt status lost", attempt)
					}
				}
			}
			wantCalls := 2
			if status == 429 {
				wantCalls = 1
			}
			if calls != wantCalls {
				t.Fatal("wrong retries during rejection/cooldown", calls)
			}
		})
	}
}
