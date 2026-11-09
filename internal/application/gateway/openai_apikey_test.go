package gateway

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPassthroughAPIKeyUsesBearerAndProviderBase 断言 /codex/v1 在账号为
// OpenAI 按量 API Key 时改走 api.openai.com，且不发送订阅的账号头。
func TestPassthroughAPIKeyUsesBearerAndProviderBase(t *testing.T) {
	s, db, _, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("协议适配器不应处理透明转发入口")
		return nil, nil
	}))
	if _, err := db.Exec(`UPDATE accounts SET provider_id='builtin_openai',auth_method='api_key',auth_state='' WHERE id='a';
		INSERT OR IGNORE INTO account_models(account_id,model,name,fetched_at,account_version) SELECT 'a','gpt-test','gpt-test','2026-01-01T00:00:00Z',version FROM account_catalog_versions WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	var gotPath, gotAuth, gotAccount, gotOriginator string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		gotAccount, gotOriginator = r.Header.Get("ChatGPT-Account-ID"), r.Header.Get("originator")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_api_key"}`))
	}))
	defer upstream.Close()
	if _, err := db.Exec(`UPDATE providers SET base_url=? WHERE id='builtin_openai'`, upstream.URL+"/v1"); err != nil {
		t.Fatal(err)
	}

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
	if gotPath != "/v1/responses" || gotAuth != "Bearer placeholder-failover-a" {
		t.Fatalf("upstream path = %q auth = %q", gotPath, gotAuth)
	}
	if gotAccount != "" {
		t.Fatalf("API Key 请求不应发送 ChatGPT-Account-ID: %q", gotAccount)
	}
	if gotOriginator != "codex_cli_rs" {
		t.Fatalf("客户端 originator 未透传: %q", gotOriginator)
	}
}

func TestResponsesAPIKeyForwardsThroughNormalGateway(t *testing.T) {
	s, db, _, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("透明转发不应处理协议入口")
		return nil, nil
	}))
	if _, err := db.Exec(`UPDATE accounts SET provider_id='builtin_openai',auth_method='api_key',auth_state='' WHERE id='a';
		INSERT OR IGNORE INTO account_models(account_id,model,name,fetched_at,account_version) SELECT 'a','gpt-test','gpt-test','2026-01-01T00:00:00Z',version FROM account_catalog_versions WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	var gotPath, gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_api_key","object":"response","output":[],"usage":{"input_tokens":3,"output_tokens":2,"total_tokens":5}}`))
	}))
	defer upstream.Close()
	if _, err := db.Exec(`UPDATE providers SET base_url=? WHERE id='builtin_openai'`, upstream.URL+"/v1"); err != nil {
		t.Fatal(err)
	}
	// 生产环境同时存在订阅服务；API Key 目标不能被订阅分支截获。
	s.SetSubscription(failoverSubscription{})
	request, err := http.NewRequest(http.MethodPost, s.Info().Runtime.Address+"/v1/responses",
		strings.NewReader(`{"model":"gpt-test","input":"Hi","stream":false}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body = %s", resp.StatusCode, body)
	}
	if gotPath != "/v1/responses" || gotAuth != "Bearer placeholder-failover-a" {
		t.Fatalf("upstream path = %q auth = %q", gotPath, gotAuth)
	}
}

func TestFallbackSessionUsesAPIKeyPoolWithoutSubscription(t *testing.T) {
	s, db, _, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, nil
	}))
	if _, err := db.Exec(`UPDATE accounts SET provider_id='builtin_openai',auth_method='api_key',auth_state='' WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	session, err := s.openFallbackSession(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	state, ok := session.(*passthroughSession)
	if !ok || len(state.request.Candidates) == 0 {
		t.Fatal("fallback session did not retain candidates")
	}
	if state.request.Candidates[0].AccountID != "a" {
		t.Fatalf("fallback account = %q", state.request.Candidates[0].AccountID)
	}
}
