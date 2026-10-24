package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

type slowFirstSubscription struct{ slowID string }

func (s slowFirstSubscription) UseToken(ctx context.Context, id string, consume func([]byte, string) error) error {
	if id == s.slowID {
		<-ctx.Done()
		return ctx.Err()
	}
	return consume([]byte("test-only-token"), "workspace-"+id)
}

func TestSocketSlowCredentialCanSwitchBeforeTotalBudgetExpires(t *testing.T) {
	s, db, _, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("透明握手不应走协议适配器")
		return nil, nil
	}))
	if _, err := db.Exec(`UPDATE accounts SET provider_id='builtin_openai',auth_method='subscription_login',auth_email=id||'@example.invalid',auth_account_id='workspace-'||id,auth_state='authenticated'`); err != nil {
		t.Fatal(err)
	}
	targets, err := s.passthroughFallbackTargets(context.Background())
	if err != nil || len(targets) < 2 {
		t.Fatal("缺少同池候选", err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("ChatGPT-Account-ID") == "workspace-"+targets[0].accountID {
			t.Error("仍使用了准备超时的账号")
		}
		socket, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer socket.CloseNow()
		kind, data, err := socket.Read(r.Context())
		if err == nil {
			_ = socket.Write(r.Context(), kind, data)
		}
	}))
	defer upstream.Close()
	if _, err := db.Exec(`UPDATE codex_settings SET upstream_base_url=? WHERE id=1`, upstream.URL); err != nil {
		t.Fatal(err)
	}
	s.SetCodexUpstreamBase(NewCodexSettingsService(db).UpstreamBaseURL)
	s.SetSubscription(slowFirstSubscription{targets[0].accountID})
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	start := time.Now()
	url := strings.Replace(s.Info().Runtime.Address, "http:", "ws:", 1) + "/codex/v1/responses"
	socket, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatal("单账号准备慢时未在总预算内换号", err)
	}
	defer socket.CloseNow()
	if time.Since(start) > 5*time.Second {
		t.Fatal("凭据准备超出合计预算")
	}
	if err := socket.Write(ctx, websocket.MessageText, []byte("test-round")); err != nil {
		t.Fatal(err)
	}
	if _, data, err := socket.Read(ctx); err != nil || string(data) != "test-round" {
		t.Fatal("换号后的中继失败", err)
	}
}
