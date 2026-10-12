package codex

import (
	"net/url"
	"reflect"
	"testing"
)

func TestBrowserCompatibilityPreservesAuthorizationSecurityParameters(t *testing.T) {
	query := url.Values{
		"client_id": {"official-client"}, "redirect_uri": {"http://localhost:1455/auth/callback"},
		"response_type": {"code"}, "scope": {"openid profile email offline_access api.connectors.read api.connectors.invoke"},
		"code_challenge": {"fake-challenge"}, "code_challenge_method": {"S256"}, "state": {"fake-state"},
		"codex_cli_simplified_flow": {"true"}, "id_token_add_organizations": {"true"},
		"originator": {"omniproxy"}, "extra": {"first", "second"},
	}
	result, err := compatibleBrowserURL("https://auth.openai.com/oauth/authorize?" + query.Encode())
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(result)
	if err != nil {
		t.Fatal(err)
	}
	query.Set("originator", "codex_vscode")
	if parsed.Scheme != "https" || parsed.Host != "auth.openai.com" || parsed.Path != "/oauth/authorize" || !reflect.DeepEqual(parsed.Query(), query) {
		t.Fatal("compatibility changed authorization security parameters")
	}
}

func TestBrowserCompatibilityIsLimitedToLegacyOfficialEndpoint(t *testing.T) {
	for _, value := range []string{"https://chatgpt.com/auth?originator=omniproxy", "https://auth.openai.com/codex/device", "https://auth.openai.com/oauth/authorize-new?originator=omniproxy"} {
		result, err := compatibleBrowserURL(value)
		if err != nil || result != value {
			t.Fatal("changed another authorization flow")
		}
	}
	for _, value := range []string{"http://auth.openai.com/oauth/authorize", "https://auth.openai.com.evil.test/oauth/authorize", "https://auth.openai.com/oauth/authorize?state=%ZZ"} {
		if _, err := compatibleBrowserURL(value); err == nil {
			t.Fatal("invalid authorization location accepted")
		}
	}
}
