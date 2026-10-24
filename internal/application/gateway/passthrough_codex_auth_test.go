package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"omniproxy/internal/application/account"
)

// codexAuthStore 保存网关自己保管的 auth.json，模拟 DPAPI 凭据文件。
type codexAuthStore struct{ values map[string][]byte }

func (s *codexAuthStore) Put(body []byte) (string, error) {
	ref := "own-ref-" + string(rune('0'+len(s.values)))
	s.values[ref] = append([]byte(nil), body...)
	return ref, nil
}

func (s *codexAuthStore) Get(ref string) ([]byte, error) {
	return append([]byte(nil), s.values[ref]...), nil
}
func (s *codexAuthStore) Delete(ref string) error { delete(s.values, ref); return nil }

func poolAuthJSON(t *testing.T, accountID, access, refresh string) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"auth_mode": "chatgpt",
		"tokens": map[string]string{
			"access_token": access, "refresh_token": refresh, "account_id": accountID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// TestPassthroughUsesGatewayOwnedCodexAuth 断言池账号（网关自己保管的 auth.json）
// 直接提供上游身份：登录由网关拥有，不再借客户端自己的官方登录。
func TestPassthroughUsesGatewayOwnedCodexAuth(t *testing.T) {
	s, db, _, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("协议适配器不应处理透明转发入口")
		return nil, nil
	}))
	if _, err := db.Exec(`UPDATE accounts SET provider_id='builtin_openai',auth_method='codex_auth_json',auth_email='pool@example.invalid',auth_account_id='acct-1',auth_state='authenticated',credential_ref='own-ref-a' WHERE id='a';
		INSERT OR IGNORE INTO account_models(account_id,model,name,fetched_at,account_version) SELECT 'a','gpt-test','GPT','2026-01-01T00:00:00Z',version FROM account_catalog_versions`); err != nil {
		t.Fatal(err)
	}
	store := &codexAuthStore{values: map[string][]byte{"own-ref-a": poolAuthJSON(t, "acct-1", "pool-access-token", "pool-refresh-token")}}
	s.accounts = account.NewService(db, store)

	if _, ok := s.accounts.(codexAuthAccess); !ok {
		t.Fatalf("accounts %T does not implement codexAuthAccess", s.accounts)
	}
	var gotAuth, gotAccount string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotAccount = r.Header.Get("Authorization"), r.Header.Get("ChatGPT-Account-ID")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_pool"}`))
	}))
	defer upstream.Close()
	if _, err := db.Exec(`UPDATE codex_settings SET upstream_base_url=? WHERE id=1`, upstream.URL+"/backend-api/codex"); err != nil {
		t.Fatal(err)
	}
	s.SetCodexUpstreamBase(NewCodexSettingsService(db).UpstreamBaseURL)

	request, err := http.NewRequest(http.MethodPost, s.Info().Runtime.Address+"/codex/v1/responses",
		strings.NewReader(`{"model":"gpt-test","input":[{"role":"user","content":"Hi"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer client-official-login-token")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, body)
	}
	if gotAuth != "Bearer pool-access-token" || gotAccount != "acct-1" {
		t.Fatalf("upstream auth = %q account = %q（必须使用池账号而不是客户端登录）", gotAuth, gotAccount)
	}
	// 池账号的凭据没有被续期改坏：仍是导入时那一份。
	if got := string(store.values["own-ref-a"]); !strings.Contains(got, `"access_token":"pool-access-token"`) {
		t.Fatalf("protected credential changed unexpectedly: %s", got)
	}
	_ = context.Background()
}
