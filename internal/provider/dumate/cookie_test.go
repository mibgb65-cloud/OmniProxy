package dumate

import (
	"strings"
	"testing"
)

func sessionCookies() []Cookie {
	return []Cookie{{Name: SessionCookie, Value: "session-token", Domain: ".baidu.com"}}
}

func TestParseCookiesKeepsFirstDuplicateAndStripsQuotes(t *testing.T) {
	cookies, err := ParseCookies(`a=1; b="two"; broken; =empty; a=2; c=has=equals`)
	if err != nil {
		t.Fatal(err)
	}
	if len(cookies) != 3 {
		t.Fatalf("cookies = %+v", cookies)
	}
	if cookies[0].Name != "a" || cookies[0].Value != "1" || cookies[1].Value != "two" || cookies[2].Value != "has=equals" {
		t.Fatalf("parsed cookies = %+v", cookies)
	}
	if cookies[0].Domain != PastedDomain {
		t.Fatal("pasted cookies must carry the shared domain")
	}
}

func TestParseCookiesRejectsUnsafeInput(t *testing.T) {
	for _, raw := range []string{"", "no-equals-sign", "\x00a=b", strings.Repeat("a", maxCookieInput+1)} {
		if _, err := ParseCookies(raw); err == nil {
			t.Fatalf("ParseCookies(%q) accepted unsafe input", raw[:min(len(raw), 16)])
		}
	}
	if _, err := ParseCookies("name=bad\r\nInjected: x"); err == nil {
		t.Fatal("control characters accepted")
	}
}

func TestValidateRequiresSessionCookie(t *testing.T) {
	if err := Validate([]Cookie{{Name: "only", Value: "x"}}); err == nil {
		t.Fatal("missing session cookie accepted")
	}
	if err := Validate(sessionCookies()); err != nil {
		t.Fatal(err)
	}
	if err := Validate(nil); err == nil {
		t.Fatal("empty cookie set accepted")
	}
}

func TestHeaderForFiltersDomainsAndQuotesSeparators(t *testing.T) {
	cookies := []Cookie{
		{Name: "a", Value: "plain", Domain: ".baidu.com"},
		{Name: "b", Value: `"quoted"`, Domain: "console.bce.baidu.com"},
		{Name: "c", Value: "has space", Domain: ".baidu.com"},
		{Name: "d", Value: "other-site", Domain: ".example.com"},
		{Name: "e", Value: "colon:value", Domain: ""},
	}
	header := HeaderFor(cookies, "console.bce.baidu.com")
	if strings.Contains(header, "other-site") {
		t.Fatalf("foreign domain leaked: %s", header)
	}
	want := `a=plain; b=quoted; c="has space"; e="colon:value"`
	if header != want {
		t.Fatalf("header = %q, want %q", header, want)
	}
	if HeaderFor(cookies, "elsewhere.example") != `e="colon:value"` {
		t.Fatal("domain matching is wrong")
	}
}

func TestCSRFOnlyFromMatchingHost(t *testing.T) {
	cookies := []Cookie{{Name: CSRFCookie, Value: "token-1", Domain: ".baidu.com"}}
	if CSRF(cookies, "console.bce.baidu.com") != "token-1" {
		t.Fatal("csrf token not found for subdomain")
	}
	if CSRF(cookies, "example.com") != "" {
		t.Fatal("csrf token leaked to another host")
	}
}

func TestHeaderForRejectsOversizedHeader(t *testing.T) {
	cookies := make([]Cookie, 0, maxCookies)
	for i := 0; i < maxCookies; i++ {
		cookies = append(cookies, Cookie{Name: "k" + strings.Repeat("x", 8), Value: strings.Repeat("v", 100)})
	}
	if HeaderFor(cookies, "console.bce.baidu.com") != "" {
		t.Fatal("oversized header was assembled")
	}
}

func TestCredentialsValidRequiresSourceAndSession(t *testing.T) {
	if (Credentials{Cookies: sessionCookies(), Source: SourceManual, AccountID: "1001"}).Valid() != true {
		t.Fatal("valid credentials rejected")
	}
	if (Credentials{Cookies: sessionCookies(), Source: "unknown"}).Valid() {
		t.Fatal("unknown source accepted")
	}
	if (Credentials{Cookies: []Cookie{{Name: "a", Value: "b"}}, Source: SourceDesktop}).Valid() {
		t.Fatal("credentials without session cookie accepted")
	}
}
