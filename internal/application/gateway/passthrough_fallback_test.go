package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// partialSubscription 只让指定账号借不出令牌，其余账号照常借出。
type partialSubscription struct{ failID string }

func (p partialSubscription) UseToken(_ context.Context, id string, consume func([]byte, string) error) error {
	if id == p.failID {
		return errors.New("private-credential-error")
	}
	return consume([]byte("placeholder-subscription-"+id), "remote-"+id)
}

// failingSubscription 让所有账号都借不出令牌。
type failingSubscription struct{}

func (failingSubscription) UseToken(context.Context, string, func([]byte, string) error) error {
	return errors.New("private-credential-error")
}

// TestPassthroughSocketRotatesUnusableCredential 断言 WebSocket 握手没有正文模型时，
// 第一个账号借不出凭据不再掐断整条通路，而是按候选换下一个账号。
func TestPassthroughSocketRotatesUnusableCredential(t *testing.T) {
	s, db, _, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("协议适配器不应处理透明转发入口")
		return nil, nil
	}))
	if _, err := db.Exec(`UPDATE accounts SET provider_id='builtin_openai',auth_method='subscription_login',auth_email=id||'@example.invalid',auth_account_id='workspace-'||id,auth_state='authenticated'`); err != nil {
		t.Fatal(err)
	}
	targets, err := s.passthroughFallbackTargets(context.Background())
	if err != nil || len(targets) < 2 {
		t.Fatalf("回退候选未覆盖全部账号: %v %v", targets, err)
	}
	broken := targets[0].accountID

	gotAccount := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccount <- r.Header.Get("ChatGPT-Account-ID")
		socket, acceptErr := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if acceptErr != nil {
			return
		}
		defer socket.CloseNow()
		kind, data, readErr := socket.Read(r.Context())
		if readErr != nil {
			return
		}
		_ = socket.Write(r.Context(), kind, data)
	}))
	defer upstream.Close()
	if _, err = db.Exec(`UPDATE codex_settings SET upstream_base_url=? WHERE id=1`, upstream.URL+"/backend-api/codex"); err != nil {
		t.Fatal(err)
	}
	s.SetCodexUpstreamBase(NewCodexSettingsService(db).UpstreamBaseURL)
	s.SetSubscription(partialSubscription{failID: broken})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dialURL := strings.Replace(s.Info().Runtime.Address, "http:", "ws:", 1) + "/codex/v1/responses"
	socket, _, err := websocket.Dial(ctx, dialURL, nil)
	if err != nil {
		t.Fatalf("第一个账号借不出凭据时未换号: %v", err)
	}
	defer socket.CloseNow()
	if err = socket.Write(ctx, websocket.MessageText, []byte(`{"type":"response.create"}`)); err != nil {
		t.Fatal(err)
	}
	if _, data, readErr := socket.Read(ctx); readErr != nil || string(data) != `{"type":"response.create"}` {
		t.Fatalf("中继失败: %v %q", readErr, data)
	}
	select {
	case account := <-gotAccount:
		if account == "workspace-"+broken {
			t.Fatalf("借不出凭据的账号仍被使用: %q", account)
		}
	case <-ctx.Done():
		t.Fatal("上游未收到握手")
	}
}

// TestPassthroughSocketReportsCredentialUnavailable 断言所有账号都借不出凭据时，
// 客户端拿到的是 credential_unavailable，而不是会被当成上游抖动的 502。
func TestPassthroughSocketReportsCredentialUnavailable(t *testing.T) {
	s, db, _, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("协议适配器不应处理透明转发入口")
		return nil, nil
	}))
	if _, err := db.Exec(`UPDATE accounts SET provider_id='builtin_openai',auth_method='subscription_login',auth_email=id||'@example.invalid',auth_account_id='workspace-'||id,auth_state='authenticated'`); err != nil {
		t.Fatal(err)
	}
	s.SetSubscription(failingSubscription{})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dialURL := strings.Replace(s.Info().Runtime.Address, "http:", "ws:", 1) + "/codex/v1/responses"
	socket, resp, err := websocket.Dial(ctx, dialURL, nil)
	if socket != nil {
		socket.CloseNow()
	}
	if err == nil || resp == nil {
		t.Fatalf("expected rejection, err=%v resp=%v", err, resp)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &body) != nil {
		t.Fatalf("unexpected body %q", raw)
	}
	if resp.StatusCode != http.StatusServiceUnavailable || body.Error.Code != "credential_unavailable" {
		t.Fatalf("status = %d code = %q", resp.StatusCode, body.Error.Code)
	}
}
