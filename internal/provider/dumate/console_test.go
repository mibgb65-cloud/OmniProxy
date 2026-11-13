package dumate

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func consoleCredentials() Credentials {
	return Credentials{Cookies: []Cookie{
		{Name: SessionCookie, Value: "session-token"},
		{Name: CSRFCookie, Value: "csrf-1"},
	}, Source: SourceManual, AccountID: "12345"}
}

func testConsole(t *testing.T, handler http.HandlerFunc) *Console {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	console, err := NewConsoleAt(server.URL, server.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	console.http.Timeout = 2 * time.Second
	return console
}

func okResult(w http.ResponseWriter, result string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"code":0,"success":true,"result":`+result+`}`)
}

func TestConsoleReadsIdentityTokenAndPoints(t *testing.T) {
	console := testConsole(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Cookie"); got != "bce-sessionid=session-token; bce-user-info=csrf-1" {
			t.Errorf("cookie header = %q", got)
		}
		switch r.URL.Path {
		case userInfoPath:
			okResult(w, `{"bceAccountId":12345,"displayName":"Tester"}`)
		case tokenPath:
			okResult(w, `{"token":"tok-1","expired_at":"2030-01-02T03:04:05Z"}`)
		case "/api/dumate/points/quota_overview":
			if query := r.URL.Query(); query.Get("timezone") != "Asia/Shanghai" || query.Get("clientType") != "desktop" || query.Get("ignoreLoginBonus") != "true" {
				t.Errorf("overview query = %v", query)
			}
			okResult(w, `{"totalPoints":"100.50","usedPoints":12.5,"subscription":[{"status":"active","totalPoints":"50","expireDate":1893456000},{"status":"expired","totalPoints":10,"expireDate":null}],"incremental":[]}`)
		case loginBonusPath:
			okResult(w, `{"hasIssued":true,"totalTimes":"7","totalPoints":88}`)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	})
	ctx := context.Background()
	credentials := consoleCredentials()
	identity, err := console.UserInfo(ctx, credentials)
	if err != nil || identity.AccountID != "12345" || identity.DisplayName != "Tester" {
		t.Fatalf("identity = (%+v, %v)", identity, err)
	}
	token, err := console.GatewayToken(ctx, credentials)
	if err != nil || token.Value != "tok-1" || token.ExpiresAt.Year() != 2030 {
		t.Fatalf("token = (%+v, %v)", token, err)
	}
	overview, err := console.Overview(ctx, credentials)
	if err != nil || overview.TotalPoints != 100.5 || overview.UsedPoints != 12.5 || len(overview.Packs) != 2 {
		t.Fatalf("overview = (%+v, %v)", overview, err)
	}
	if overview.Packs[0].ExpireDate == nil || *overview.Packs[0].ExpireDate != 1893456000 || overview.Packs[1].ExpireDate != nil {
		t.Fatalf("packs = %+v", overview.Packs)
	}
	bonus, err := console.LoginBonus(ctx, credentials)
	if err != nil || !bonus.HasIssued || bonus.TotalTimes != 7 || bonus.TotalPoints != 88 {
		t.Fatalf("bonus = (%+v, %v)", bonus, err)
	}
}

func TestConsoleClaimSendsCSRFAndXHRHeaders(t *testing.T) {
	seen := false
	console := testConsole(t, func(w http.ResponseWriter, r *http.Request) {
		seen = true
		if r.Method != http.MethodPost || r.URL.Path != claimBonusPath {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("csrftoken") != "csrf-1" || r.Header.Get("X-Requested-With") != "XMLHttpRequest" {
			t.Errorf("csrf headers = %v", r.Header)
		}
		if r.Header.Get("Origin") == "" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("headers = %v", r.Header)
		}
		okResult(w, `{}`)
	})
	if err := console.ClaimLoginBonus(context.Background(), consoleCredentials()); err != nil || !seen {
		t.Fatalf("claim = %v seen=%v", err, seen)
	}
}

func TestConsoleMapsFailedResponsesToFixedCodes(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		ctype   string
		body    string
		signIn  bool
		code    string
		maximum int
	}{
		{name: "html login page", status: 200, ctype: "text/html", body: "<!DOCTYPE html><html>login</html>", signIn: true},
		{name: "redirect to login", status: 302, ctype: "", body: "", signIn: true},
		{name: "business sign-in", status: 200, ctype: "application/json", body: `{"code":1,"message":"登录凭证已过期"}`, signIn: true},
		{name: "business rejection", status: 200, ctype: "application/json", body: `{"code":2,"message":"quota denied"}`, code: CodeRejected},
		{name: "not json", status: 200, ctype: "application/json", body: `plain text`, code: CodeInvalidResponse},
		{name: "rate limited", status: 429, ctype: "application/json", body: `{}`, code: CodeRateLimited},
		{name: "server error", status: 503, ctype: "application/json", body: `{}`, code: CodeConsoleUnavailable},
		{name: "bad request", status: 400, ctype: "application/json", body: `{}`, code: CodeConsoleUnavailable},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			console := testConsole(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", testCase.ctype)
				w.WriteHeader(testCase.status)
				io.WriteString(w, testCase.body)
			})
			_, err := console.Overview(context.Background(), consoleCredentials())
			if testCase.signIn {
				if !SignInExpired(err) {
					t.Fatalf("err = %v, want sign-in expired", err)
				}
				return
			}
			if code := ErrorCode(err); code != testCase.code {
				t.Fatalf("code = %q, want %q", code, testCase.code)
			}
		})
	}
}

func TestConsoleRejectsOversizedAndMissingCookies(t *testing.T) {
	console := testConsole(t, func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, strings.Repeat("x", maxConsoleBytes+10))
	})
	if code := ErrorCode(consoleError(t, console)); code != CodeConsoleUnavailable {
		t.Fatalf("oversized body code = %q", code)
	}
	empty := Credentials{Cookies: []Cookie{{Name: "other", Value: "v", Domain: ".example.com"}}, Source: SourceManual}
	if _, err := console.Overview(context.Background(), empty); ErrorCode(err) != CodeCredentialUnavailable {
		t.Fatalf("missing cookie = %v", err)
	}
}

func consoleError(t *testing.T, console *Console) error {
	t.Helper()
	_, err := console.Overview(context.Background(), consoleCredentials())
	if err == nil {
		t.Fatal("expected an error")
	}
	return err
}

func TestParseExpiryFallsBackToTokenTTL(t *testing.T) {
	now := time.Now()
	for _, raw := range []string{"", "not-a-date", "2001-01-01T00:00:00Z", "-5"} {
		expiry := parseExpiry(raw)
		if expiry.Before(now.Add(TokenTTL-time.Minute)) || expiry.After(now.Add(TokenTTL+time.Minute)) {
			t.Fatalf("parseExpiry(%q) = %v", raw, expiry)
		}
	}
	if expiry := parseExpiry("1893456000"); expiry.Unix() != 1893456000 {
		t.Fatalf("unix seconds = %v", expiry)
	}
}
