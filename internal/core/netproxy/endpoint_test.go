package netproxy

import (
	"errors"
	"testing"
)

func TestParseEndpoint(t *testing.T) {
	cases := []struct {
		text     string
		scheme   string
		host     string
		port     int
		username string
		password string
	}{
		{text: "10808", scheme: SchemeAuto, host: "127.0.0.1", port: 10808},
		{text: "mixed:10808", scheme: SchemeAuto, host: "127.0.0.1", port: 10808},
		{text: "socks:1087", scheme: SchemeSOCKS5, host: "127.0.0.1", port: 1087},
		{text: "http:10809", scheme: SchemeHTTP, host: "127.0.0.1", port: 10809},
		{text: " 127.0.0.1:10809 ", scheme: SchemeAuto, host: "127.0.0.1", port: 10809},
		{text: "socks5h://127.0.0.1:10808", scheme: SchemeSOCKS5, host: "127.0.0.1", port: 10808},
		{text: "http://user:pass@192.168.1.2:7890", scheme: SchemeHTTP, host: "192.168.1.2", port: 7890, username: "user", password: "pass"},
		{text: "mixed://10.0.0.5", scheme: SchemeAuto, host: "10.0.0.5", port: DefaultSOCKSPort},
	}
	for _, testCase := range cases {
		endpoint, err := ParseEndpoint(testCase.text)
		if err != nil {
			t.Fatalf("ParseEndpoint(%q): %v", testCase.text, err)
		}
		if endpoint.Scheme != testCase.scheme || endpoint.Host != testCase.host || endpoint.Port != testCase.port {
			t.Fatalf("ParseEndpoint(%q) = %#v", testCase.text, endpoint)
		}
		if endpoint.Username != testCase.username || endpoint.Password != testCase.password {
			t.Fatalf("ParseEndpoint(%q) credentials = %q/%q", testCase.text, endpoint.Username, endpoint.Password)
		}
	}

	invalid := []string{"", "   ", "127.0.0.1", "mixed:", ":10808", "10808:80", "ftp://1.2.3.4:1080", "127.0.0.1:70000", "http://:pass@127.0.0.1:1080", "127.0.0.1:1080/path"}
	for _, text := range invalid {
		if _, err := ParseEndpoint(text); !errors.Is(err, ErrInvalidEndpoint) {
			t.Fatalf("ParseEndpoint(%q) expected ErrInvalidEndpoint, got %v", text, err)
		}
	}
}

func TestEndpointStringRedactsCredentials(t *testing.T) {
	endpoint := Endpoint{Scheme: SchemeHTTP, Host: "127.0.0.1", Port: 10809, Username: "user", Password: "secret"}
	if got := endpoint.String(); got != "http://127.0.0.1:10809" {
		t.Fatalf("String() = %q", got)
	}
	if got := Redact("http://user:secret@127.0.0.1:10809"); got != "http://***@127.0.0.1:10809" {
		t.Fatalf("Redact() = %q", got)
	}
	if got := Redact("plain text"); got != "plain text" {
		t.Fatalf("Redact() = %q", got)
	}
}

func TestDecide(t *testing.T) {
	configured := Snapshot{
		Version:  "v1",
		Enabled:  true,
		Endpoint: Endpoint{Scheme: SchemeAuto, Host: "127.0.0.1", Port: 10808},
		NoProxy:  []string{"localhost", "*.internal", "192.168.0.10:8080"},
		Modes:    map[string]string{"prov_proxy": ModeProxy, "prov_direct": ModeDirect, "prov_inherit": ModeInherit},
		HostModes: map[string]string{
			"api.host-mode.example": ModeProxy,
		},
	}

	cases := []struct {
		name       string
		snapshot   Snapshot
		scope      Scope
		target     string
		useProxy   bool
		reason     string
		providerID string
	}{
		{name: "selected provider", snapshot: configured, scope: Scope{ProviderID: "prov_proxy"}, target: "api.openai.com:443", useProxy: true, reason: ReasonSelected, providerID: "prov_proxy"},
		{name: "direct provider", snapshot: configured, scope: Scope{ProviderID: "prov_direct"}, target: "api.openai.com:443", reason: ReasonDirect, providerID: "prov_direct"},
		{name: "inherit global on", snapshot: configured, scope: Scope{ProviderID: "prov_inherit"}, target: "api.openai.com:443", useProxy: true, reason: ReasonInherited, providerID: "prov_inherit"},
		{name: "inherit global off", snapshot: func() Snapshot { value := configured; value.Enabled = false; return value }(), scope: Scope{ProviderID: "prov_inherit"}, target: "api.deepseek.com:443", reason: ReasonDisabled, providerID: "prov_inherit"},
		{name: "no_proxy wildcard", snapshot: configured, scope: Scope{ProviderID: "prov_inherit"}, target: "api.internal:443", reason: ReasonNoProxy, providerID: "prov_inherit"},
		{name: "no_proxy host port", snapshot: configured, scope: Scope{ProviderID: "prov_inherit"}, target: "192.168.0.10:8080", reason: ReasonNoProxy, providerID: "prov_inherit"},
		{name: "loopback stays direct", snapshot: configured, scope: Scope{}, target: "127.0.0.1:11434", reason: ReasonNoProxy},
		{name: "explicit proxy overrides loopback", snapshot: configured, scope: Scope{ProviderID: "prov_proxy"}, target: "127.0.0.1:11434", useProxy: true, reason: ReasonSelected, providerID: "prov_proxy"},
		{name: "host fallback", snapshot: configured, scope: Scope{}, target: "api.host-mode.example:443", useProxy: true, reason: ReasonSelected},
		{name: "not configured", snapshot: func() Snapshot { value := configured; value.Endpoint = Endpoint{}; return value }(), scope: Scope{ProviderID: "prov_proxy"}, target: "api.openai.com:443", reason: ReasonNotConfigured, providerID: "prov_proxy"},
	}
	for _, testCase := range cases {
		decision := Decide(testCase.snapshot, testCase.scope, testCase.target)
		if decision.UseProxy != testCase.useProxy || decision.Reason != testCase.reason || decision.ProviderID != testCase.providerID {
			t.Fatalf("%s: %#v", testCase.name, decision)
		}
	}
}
