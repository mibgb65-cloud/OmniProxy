package gateway

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"io"
	"net/http"
	"omniproxy/internal/application/activity"
	"strings"
	"sync"
	"testing"
	"time"
)

type quotaAwareSubscription struct {
	mu      sync.Mutex
	blocked map[string]time.Time
}

func (s *quotaAwareSubscription) UseToken(ctx context.Context, id string, consume func([]byte, string) error) error {
	return (failoverSubscription{}).UseToken(ctx, id, consume)
}
func (s *quotaAwareSubscription) QuotaBlockedUntil(id, version string, now time.Time) (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	at := s.blocked[id]
	return at, at.After(now)
}
func (s *quotaAwareSubscription) RecordQuotaRejection(id, version string, until time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blocked[id] = until
}

func subscriptionQuotaFixture(t *testing.T) (*Service, *sql.DB, *activity.Service, *quotaAwareSubscription) {
	t.Helper()
	s, db, observer, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("wrong adapter"); return nil, nil }))
	_, err := db.Exec(`UPDATE accounts SET provider_id='builtin_openai',enabled=(id IN ('a','b')),auth_method='subscription_login',auth_state='authenticated',auth_email=id||'@example.invalid',auth_account_id='remote-'||id; INSERT INTO account_models(account_id,model,name,fetched_at,account_version) SELECT id,'gpt-test','GPT','2026-01-01T00:00:00Z',version FROM account_catalog_versions WHERE id='a'`)
	if err != nil {
		t.Fatal(err)
	}
	access := &quotaAwareSubscription{blocked: map[string]time.Time{}}
	s.SetSubscription(access)
	return s, db, observer, access
}

func requestQuotaGateway(t *testing.T, s *Service, transport, model string) (bool, int, string) {
	t.Helper()
	address := s.Info().Runtime.Address + "/v1/responses"
	if transport == "http" {
		response, err := http.Post(address, "application/json", strings.NewReader(fmt.Sprintf(`{"model":%q,"stream":true,"input":[]}`, model)))
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if strings.Contains(string(raw), "private-quota-marker") {
			t.Fatal("raw quota error leaked")
		}
		return strings.Contains(string(raw), `"response.completed"`), response.StatusCode, string(raw)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	socket, _, err := websocket.Dial(ctx, strings.Replace(address, "http:", "ws:", 1), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer socket.CloseNow()
	socket.SetReadLimit(4 << 20)
	if err = socket.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf(`{"type":"response.create","model":%q,"input":[]}`, model))); err != nil {
		t.Fatal(err)
	}
	for {
		_, raw, err := socket.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "private-quota-marker") {
			t.Fatal("raw quota error leaked")
		}
		if strings.Contains(string(raw), `"response.completed"`) {
			return true, 200, string(raw)
		}
		if strings.Contains(string(raw), `"type":"error"`) {
			var failure struct {
				Status int `json:"status"`
			}
			json.Unmarshal(raw, &failure)
			return false, failure.Status, string(raw)
		}
	}
}
