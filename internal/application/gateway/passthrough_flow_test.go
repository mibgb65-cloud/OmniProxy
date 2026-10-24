package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPassthroughForwardsThroughConfiguredBase 断言 Codex 入口按 v1 语义转发：
// 使用设置里的上游基址、网关借出的令牌，并把 /v1/responses 归一成 /responses。
func TestPassthroughForwardsThroughConfiguredBase(t *testing.T) {
	s, db, _, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("协议适配器不应处理透明转发入口")
		return nil, nil
	}))
	if _, err := db.Exec(`UPDATE accounts SET provider_id='builtin_openai',auth_method='subscription_login',auth_email=id||'@example.invalid',auth_account_id='workspace-'||id,auth_state='authenticated' WHERE id='a';
		INSERT OR IGNORE INTO account_models(account_id,model,name,fetched_at,account_version) SELECT 'a','gpt-test','GPT','2026-01-01T00:00:00Z',version FROM account_catalog_versions`); err != nil {
		t.Fatal(err)
	}
	var gotPath, gotAuth, gotAccount, gotOriginator string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		gotAccount, gotOriginator = r.Header.Get("ChatGPT-Account-ID"), r.Header.Get("originator")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_passthrough"}`))
	}))
	defer upstream.Close()
	if _, err := db.Exec(`UPDATE codex_settings SET upstream_base_url=? WHERE id=1`, upstream.URL+"/backend-api/codex"); err != nil {
		t.Fatal(err)
	}
	s.SetSubscription(failoverSubscription{})
	s.SetCodexUpstreamBase(NewCodexSettingsService(db).UpstreamBaseURL)

	request, err := http.NewRequest(http.MethodPost, s.Info().Runtime.Address+"/codex/v1/responses",
		strings.NewReader(`{"model":"gpt-test","input":[{"role":"user","content":"Hi"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("originator", "codex_cli_rs")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if gotPath != "/backend-api/codex/responses" {
		t.Fatalf("upstream path = %q", gotPath)
	}
	if gotAuth != "Bearer placeholder-subscription-a" || gotAccount != "remote-a" {
		t.Fatalf("upstream auth = %q account = %q", gotAuth, gotAccount)
	}
	if gotOriginator != "codex_cli_rs" {
		t.Fatalf("客户端 originator 未透传: %q", gotOriginator)
	}
}

// TestPassthroughRejectsUnknownPrefixedPath 断言其它入口前缀仍按协议白名单返回 404。
func TestPassthroughRejectsUnknownPrefixedPath(t *testing.T) {
	s, _, _, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("未定义的路径不应到达上游")
		return nil, nil
	}))
	resp, err := http.Get(s.Info().Runtime.Address + "/claude/v1/unknown")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}
