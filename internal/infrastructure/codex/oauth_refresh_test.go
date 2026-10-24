package codex

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// testJWT 生成只有 exp 的假 JWT，用于覆盖过期判断而不需要真实签名。
func testJWT(exp time.Time) string {
	payload, _ := json.Marshal(map[string]any{"exp": exp.Unix()})
	return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

func TestAuthNeedsRefreshDetectsExpiringAccessToken(t *testing.T) {
	now := time.Now()
	expiring := `{"tokens":{"access_token":"` + testJWT(now.Add(5*time.Minute)) + `","account_id":"acct"}}`
	fresh := `{"tokens":{"access_token":"` + testJWT(now.Add(4*time.Hour)) + `","account_id":"acct"}}`
	if !AuthNeedsRefresh(expiring, now) {
		t.Fatal("临近过期的访问令牌必须续期")
	}
	if AuthNeedsRefresh(fresh, now) {
		t.Fatal("仍有效的访问令牌不应续期")
	}
	if !AuthNeedsRefresh("{not json", now) {
		t.Fatal("无法解析的授权必须报告需要续期")
	}
}

func TestRefreshAuthJSONReplacesAccessToken(t *testing.T) {
	var gotForm string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(io.LimitReader(r.Body, 4096))
		gotForm = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"new-access","refresh_token":"r2","expires_in":3600}`))
	}))
	defer upstream.Close()

	previous := oauthTokenURL
	oauthTokenURL = upstream.URL
	t.Cleanup(func() { oauthTokenURL = previous })

	raw := `{"tokens":{"access_token":"` + testJWT(time.Now().Add(-time.Hour)) + `","refresh_token":"r1","account_id":"acct"}}`
	updated, changed, err := RefreshAuthJSON(context.Background(), upstream.Client(), raw, time.Now())
	if err != nil || !changed {
		t.Fatalf("refresh failed: %v changed=%v", err, changed)
	}
	var parsed struct {
		Tokens map[string]string `json:"tokens"`
	}
	if json.Unmarshal([]byte(updated), &parsed) != nil {
		t.Fatal("refreshed auth is not JSON")
	}
	if parsed.Tokens["access_token"] != "new-access" || parsed.Tokens["refresh_token"] != "r2" || parsed.Tokens["account_id"] != "acct" {
		t.Fatalf("unexpected refreshed tokens: %+v", parsed.Tokens)
	}
	for _, field := range []string{"grant_type=refresh_token", "refresh_token=r1", "client_id=" + oauthClientID} {
		if !strings.Contains(gotForm, field) {
			t.Fatalf("token request missing %q: %s", field, gotForm)
		}
	}
}

func TestAccessTokenFromAuthRequiresAccountID(t *testing.T) {
	access, refresh, account, ok := accessTokenFromAuth(`{"tokens":{"access_token":"a","refresh_token":"r","account_id":"acct"}}`)
	if !ok || access != "a" || refresh != "r" || account != "acct" {
		t.Fatalf("unexpected parse: %q %q %q %v", access, refresh, account, ok)
	}
	if _, _, _, ok := accessTokenFromAuth(`{"tokens":{"access_token":"a"}}`); ok {
		t.Fatal("缺少账号标识的授权不能被接受")
	}
}
