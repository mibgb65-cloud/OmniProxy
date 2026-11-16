package workbuddyai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// withFastPolling 缩短登录轮询间隔，避免测试真的等秒级。
func withFastPolling(t *testing.T) {
	t.Helper()
	previous := pollInterval
	pollInterval = time.Millisecond
	t.Cleanup(func() { pollInterval = previous })
}

func TestBeginSignInReturnsAuthorizeURL(t *testing.T) {
	var method, path string
	var query url.Values
	var header http.Header
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		method, path, query, header = r.Method, r.URL.Path, r.URL.Query(), r.Header.Clone()
		io.WriteString(w, `{"code":0,"msg":"ok","data":{"state":"state-1","authUrl":"https://login.example.com/auth?from=plugin"}}`)
	})
	state, err := client.BeginSignIn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPost || path != "/v2/plugin/auth/state" || query.Get("platform") != "workbuddy-ai" {
		t.Fatalf("request = %s %s?%s", method, path, query.Encode())
	}
	for _, name := range []string{"X-No-Authorization", "X-No-User-Id", "X-No-Enterprise-Id", "X-No-Department-Info"} {
		if header.Get(name) != "true" {
			t.Errorf("%s = %q", name, header.Get(name))
		}
	}
	if header.Get("Authorization") != "" {
		t.Error("sign-in state request carried Authorization")
	}
	if state.State != "state-1" {
		t.Fatalf("state = %q", state.State)
	}
	authURL, err := url.Parse(state.AuthURL)
	if err != nil {
		t.Fatal(err)
	}
	if authURL.Scheme != "https" || authURL.Host != "login.example.com" {
		t.Fatalf("auth url = %s", state.AuthURL)
	}
	params := authURL.Query()
	if params.Get("from") != "plugin" || params.Get("version") != AppVersion {
		t.Fatalf("auth params = %s", params.Encode())
	}
	if session := params.Get("loginSessionId"); len(session) != 32 {
		t.Fatalf("loginSessionId = %q", session)
	}
}

func TestBeginSignInAcceptsFlatResponse(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"state":"state-flat","authUrl":"https://login.example.com/b"}`)
	})
	state, err := client.BeginSignIn(context.Background())
	if err != nil || state.State != "state-flat" {
		t.Fatalf("state = %+v, err = %v", state, err)
	}
}

func TestBeginSignInRejectsUnsafeOrIncompleteResponses(t *testing.T) {
	cases := []struct {
		name string
		body string
		code string
	}{
		{name: "http auth url", body: `{"code":0,"data":{"state":"s","authUrl":"http://login.example.com"}}`, code: CodeInvalidResponse},
		{name: "no state", body: `{"code":0,"data":{"authUrl":"https://login.example.com"}}`, code: CodeInvalidResponse},
		{name: "no auth url", body: `{"code":0,"data":{"state":"s"}}`, code: CodeInvalidResponse},
		{name: "not json", body: `nope`, code: CodeInvalidResponse},
		{name: "business rejection", body: `{"code":11128,"msg":"unapproved channel"}`, code: CodeChannelRejected},
		{name: "unknown business code", body: `{"code":10001,"msg":"nope"}`, code: CodeUpstreamUnavailable},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				io.WriteString(w, test.body)
			})
			if _, err := client.BeginSignIn(context.Background()); ErrorCode(err) != test.code {
				t.Fatalf("err = %v, want %q", err, test.code)
			}
		})
	}
	for status, want := range map[int]string{http.StatusUnauthorized: CodeUnauthorized, http.StatusTooManyRequests: CodeRateLimited, http.StatusInternalServerError: CodeUpstreamUnavailable} {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) })
		if _, err := client.BeginSignIn(context.Background()); ErrorCode(err) != want {
			t.Fatalf("status %d => %v, want %q", status, err, want)
		}
	}
}

func TestWaitForTokenPollsPendingState(t *testing.T) {
	withFastPolling(t)
	requests := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/v2/plugin/auth/token" || r.URL.Query().Get("state") != "state-1" {
			t.Errorf("request = %s", r.URL.String())
		}
		if requests < 3 {
			io.WriteString(w, `{"code":11217,"msg":"sign-in not finished"}`)
			return
		}
		io.WriteString(w, `{"code":0,"data":{"accessToken":"access-new","refreshToken":"refresh-new","expiresAt":1900000000123,"refreshExpiresIn":3600,"domain":"d.example","tokenType":"Bearer"}}`)
	})
	token, err := client.WaitForToken(context.Background(), "state-1")
	if err != nil {
		t.Fatal(err)
	}
	if requests != 3 {
		t.Fatalf("requests = %d", requests)
	}
	if token.AccessToken != "access-new" || token.RefreshToken != "refresh-new" || token.Domain != "d.example" || token.TokenType != "Bearer" {
		t.Fatalf("token = %+v", token)
	}
	if token.ExpiresAt.UnixMilli() != 1900000000123 {
		t.Fatalf("expiresAt = %v", token.ExpiresAt)
	}
	if until := time.Until(token.RefreshExpiresAt); until < 59*time.Minute || until > 61*time.Minute {
		t.Fatalf("refreshExpiresAt = %v", token.RefreshExpiresAt)
	}
}

func TestWaitForTokenParsesAlternativeTimes(t *testing.T) {
	withFastPolling(t)
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"code":0,"data":{"accessToken":"a","expiresAt":"1900000000","refreshExpiresAt":"2029-01-02T03:04:05Z"}}`)
	})
	token, err := client.WaitForToken(context.Background(), "state-1")
	if err != nil {
		t.Fatal(err)
	}
	if token.ExpiresAt.Unix() != 1900000000 {
		t.Fatalf("seconds expiresAt = %v", token.ExpiresAt)
	}
	if token.RefreshExpiresAt.Format(time.RFC3339) != "2029-01-02T03:04:05Z" {
		t.Fatalf("rfc3339 refreshExpiresAt = %v", token.RefreshExpiresAt)
	}
}

func TestWaitForTokenRetriesEmptyAndRateLimited(t *testing.T) {
	withFastPolling(t)
	requests := 0
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		requests++
		switch requests {
		case 1:
			w.WriteHeader(http.StatusTooManyRequests)
		case 2:
			io.WriteString(w, `{"code":0,"data":{}}`)
		default:
			io.WriteString(w, `{"code":0,"data":{"accessToken":"a-final"}}`)
		}
	})
	token, err := client.WaitForToken(context.Background(), "state-1")
	if err != nil || token.AccessToken != "a-final" || requests != 3 {
		t.Fatalf("token = %+v, requests = %d, err = %v", token, requests, err)
	}
}

func TestWaitForTokenCancellation(t *testing.T) {
	withFastPolling(t)
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"code":11217,"msg":"pending"}`)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err := client.WaitForToken(ctx, "state-1"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
}

func TestWaitForTokenWindowExpiry(t *testing.T) {
	withFastPolling(t)
	previous := signInWindow
	signInWindow = 20 * time.Millisecond
	t.Cleanup(func() { signInWindow = previous })
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"code":11217,"msg":"pending"}`)
	})
	_, err := client.WaitForToken(context.Background(), "state-1")
	if ErrorCode(err) != CodeSignInExpired || !SignInExpired(err) {
		t.Fatalf("err = %v", err)
	}
}

func TestWaitForTokenMapsFailures(t *testing.T) {
	for _, test := range []struct {
		status int
		body   string
		want   string
	}{
		{status: http.StatusBadRequest, body: `{"code":10085,"msg":"ua"}`, want: CodeUserAgentRefused},
		{status: http.StatusBadRequest, body: `{"code":11128}`, want: CodeChannelRejected},
		{status: http.StatusBadRequest, body: `{"code":11159}`, want: CodeSignatureInvalid},
		{status: http.StatusBadRequest, body: `{"code":11158}`, want: CodeModelNotEntitled},
		{status: http.StatusUnauthorized, want: CodeUnauthorized},
		{status: http.StatusBadRequest, body: `{"code":99999}`, want: CodeUpstreamUnavailable},
		{status: http.StatusOK, body: `{"code":0,"data":{"refreshToken":"r"}}`, want: CodeInvalidResponse},
		{status: http.StatusOK, body: `{"code":0,"data":{"accessToken":"bad token"}}`, want: CodeInvalidResponse},
	} {
		withFastPolling(t)
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(test.status)
			io.WriteString(w, test.body)
		})
		if _, err := client.WaitForToken(context.Background(), "state-1"); ErrorCode(err) != test.want {
			t.Fatalf("status %d body %s => %v, want %q", test.status, test.body, err, test.want)
		}
	}
	if _, err := New(nil).WaitForToken(context.Background(), ""); proxyCode(err) != "invalid_request" {
		t.Fatalf("empty state => %v", err)
	}
}

func TestReadIdentityUsesSignInState(t *testing.T) {
	withFastPolling(t)
	var accountHeader http.Header
	var accountQuery url.Values
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/plugin/auth/token":
			io.WriteString(w, `{"code":0,"data":{"accessToken":"access-1","domain":"d.example"}}`)
		case "/v2/plugin/login/account":
			accountHeader, accountQuery = r.Header.Clone(), r.URL.Query()
			io.WriteString(w, `{"code":0,"data":{"uid":"uid-1","nickname":"Tester","enterpriseId":"ent-1","planName":"Pro"}}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	token, err := client.WaitForToken(context.Background(), "state-9")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := client.ReadIdentity(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if accountQuery.Get("state") != "state-9" {
		t.Fatalf("identity query = %s", accountQuery.Encode())
	}
	if accountHeader.Get("Authorization") != "Bearer access-1" || accountHeader.Get("X-Domain") != "d.example" {
		t.Fatalf("identity headers = %v", accountHeader)
	}
	if identity.UID != "uid-1" || identity.Name != "Tester" || identity.EnterpriseID != "ent-1" || identity.Plan != "Pro" {
		t.Fatalf("identity = %+v", identity)
	}
}

func TestReadIdentityToleratesMissingFields(t *testing.T) {
	var query url.Values
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		io.WriteString(w, `{"code":0,"data":{"uid":"uid-only"}}`)
	})
	identity, err := client.ReadIdentity(context.Background(), Token{AccessToken: "access-1"})
	if err != nil {
		t.Fatal(err)
	}
	if identity.UID != "uid-only" || identity.EnterpriseID != "" || identity.Name != "" || identity.Plan != "" {
		t.Fatalf("identity = %+v", identity)
	}
	if query.Has("state") {
		t.Fatalf("unexpected state query: %s", query.Encode())
	}
	if _, err := client.ReadIdentity(context.Background(), Token{}); ErrorCode(err) != CodeCredentialUnavailable {
		t.Fatalf("empty token => %v", err)
	}
}

func TestReadIdentityMapsRejection(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	if _, err := client.ReadIdentity(context.Background(), Token{AccessToken: "access-1"}); ErrorCode(err) != CodeUnauthorized {
		t.Fatalf("err = %v", err)
	}
}

func TestRefreshExchangesTokens(t *testing.T) {
	var method, path string
	var header http.Header
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		method, path, header = r.Method, r.URL.Path, r.Header.Clone()
		io.WriteString(w, `{"code":0,"data":{"accessToken":"access-2","refreshToken":"refresh-2","expiresIn":3600,"domain":"d.example"}}`)
	})
	token, err := client.Refresh(context.Background(), testCredentials())
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPost || path != "/v2/plugin/auth/token/refresh" {
		t.Fatalf("request = %s %s", method, path)
	}
	if header.Get("X-Refresh-Token") != "refresh-token-1" || header.Get("X-Auth-Refresh-Source") != "plugin" || header.Get("X-Domain") != "www.workbuddy.ai" {
		t.Fatalf("refresh headers = %v", header)
	}
	// 续期也是业务请求：带完整签名头。
	if header.Get("Authorization") != "Bearer access-token-1" || header.Get("X-User-Id") != "uid-1" || header.Get("X-Product") != "SaaS" {
		t.Fatalf("refresh signature headers = %v", header)
	}
	if header.Get("User-Agent") != bizUserAgent {
		t.Fatalf("refresh user agent = %q", header.Get("User-Agent"))
	}
	if token.AccessToken != "access-2" || token.RefreshToken != "refresh-2" || token.Domain != "d.example" {
		t.Fatalf("token = %+v", token)
	}
	if until := time.Until(token.ExpiresAt); until < 59*time.Minute || until > 61*time.Minute {
		t.Fatalf("expiresAt = %v", token.ExpiresAt)
	}
}

func TestRefreshWithoutValidAccessOmitsAuthorization(t *testing.T) {
	var header http.Header
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		header = r.Header.Clone()
		io.WriteString(w, `{"code":0,"data":{"accessToken":"access-3"}}`)
	})
	cred := testCredentials()
	cred.UID = ""
	if _, err := client.Refresh(context.Background(), cred); err != nil {
		t.Fatal(err)
	}
	if header.Get("Authorization") != "" || header.Get("X-User-Id") != "" {
		t.Fatalf("invalid credential sent identity headers: %v", header)
	}
	if header.Get("X-Domain") != "www.workbuddy.ai" || header.Get("X-Refresh-Token") != "refresh-token-1" {
		t.Fatalf("refresh headers = %v", header)
	}
}

func TestRefreshMapsLoginFailures(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   string
		signIn bool
	}{
		{name: "unauthorized", status: http.StatusUnauthorized, want: CodeSignInExpired, signIn: true},
		{name: "forbidden", status: http.StatusForbidden, want: CodeSignInExpired, signIn: true},
		{name: "bad request", status: http.StatusBadRequest, want: CodeRefreshRejected, signIn: true},
		{name: "gone", status: http.StatusGone, want: CodeRefreshRejected, signIn: true},
		{name: "business rejection", status: http.StatusOK, body: `{"code":10001,"msg":"refresh token invalid"}`, want: CodeRefreshRejected, signIn: true},
		{name: "rate limited", status: http.StatusTooManyRequests, want: CodeRateLimited},
		{name: "server error", status: http.StatusInternalServerError, want: CodeUpstreamUnavailable},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				io.WriteString(w, test.body)
			})
			_, err := client.Refresh(context.Background(), testCredentials())
			if ErrorCode(err) != test.want {
				t.Fatalf("code = %q, want %q", ErrorCode(err), test.want)
			}
			if SignInExpired(err) != test.signIn {
				t.Fatalf("sign-in = %v, want %v", SignInExpired(err), test.signIn)
			}
		})
	}
	cred := testCredentials()
	cred.RefreshToken = ""
	if _, err := New(nil).Refresh(context.Background(), cred); ErrorCode(err) != CodeSignInExpired || !SignInExpired(err) {
		t.Fatalf("missing refresh token => %v", err)
	}
}

func TestRefreshRejectsIncompleteToken(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"code":0,"data":{"refreshToken":"only-refresh"}}`)
	})
	if _, err := client.Refresh(context.Background(), testCredentials()); ErrorCode(err) != CodeInvalidResponse {
		t.Fatalf("err = %v", err)
	}
}

func TestDecodeBusinessRejectsEmptyBody(t *testing.T) {
	if _, err := decodeBusiness(nil); ErrorCode(err) != CodeInvalidResponse {
		t.Fatalf("err = %v", err)
	}
	if _, err := decodeBusiness([]byte(`{"code":0,"data":{"a":1}}`)); err != nil {
		t.Fatalf("data envelope => %v", err)
	}
	raw, err := decodeBusiness([]byte(`{"a":1}`))
	if err != nil || strings.TrimSpace(string(raw)) != `{"a":1}` {
		t.Fatalf("plain body => (%s, %v)", raw, err)
	}
}

func TestNumericCodeForms(t *testing.T) {
	cases := map[string]int{
		`11128`:           11128,
		`"11128"`:         11128,
		`"11158:missing"`: 11158,
		`0`:               0,
	}
	for text, want := range cases {
		var value any
		if err := json.Unmarshal([]byte(text), &value); err != nil {
			t.Fatal(err)
		}
		got, found := numericCode(value)
		if !found || got != want {
			t.Errorf("%s => (%d, %v)", text, got, found)
		}
	}
	for _, text := range []string{`"abc"`, `true`, `null`, `-1`, `1.5`} {
		var value any
		if err := json.Unmarshal([]byte(text), &value); err != nil {
			t.Fatal(err)
		}
		if _, found := numericCode(value); found {
			t.Errorf("%s parsed as business code", text)
		}
	}
}
