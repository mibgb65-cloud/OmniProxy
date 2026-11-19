package workbuddyai

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestCredentialsValidityBounds(t *testing.T) {
	valid := testCredentials()
	valid.ExpiresAt = time.Now().Add(time.Hour)
	valid.RefreshExpiresAt = time.Now().Add(24 * time.Hour)
	valid.TokenType = "Bearer"
	valid.EnterpriseID = "ent-1"
	if !valid.Valid() || !valid.SignedIn() {
		t.Fatalf("valid credentials rejected: %+v", valid)
	}
	cases := map[string]Credentials{
		"empty":            {},
		"no uid":           {AccessToken: "access"},
		"no access":        {UID: "uid"},
		"space in token":   {AccessToken: "bad token", UID: "uid"},
		"token too long":   {AccessToken: strings.Repeat("a", maxTokenBytes+1), UID: "uid"},
		"control in token": {AccessToken: "access\x1f", UID: "uid"},
		"control in uid":   {AccessToken: "access", UID: "uid\n"},
		"uid too long":     {AccessToken: "access", UID: strings.Repeat("u", maxUIDBytes+1)},
		"domain too long":  {AccessToken: "access", UID: "uid", Domain: strings.Repeat("d", maxDomainBytes+1)},
		"bad refresh":      {AccessToken: "access", UID: "uid", RefreshToken: "refresh\r\nX-Injected: 1"},
		"bad token type":   {AccessToken: "access", UID: "uid", TokenType: strings.Repeat("t", maxTokenTypeBytes+1)},
		"bad enterprise":   {AccessToken: "access", UID: "uid", EnterpriseID: strings.Repeat("e", maxEnterpriseBytes+1)},
		"control in name":  {AccessToken: "access", UID: "uid", AccountName: "bad\x00name"},
		"invalid utf8":     {AccessToken: "access", UID: "uid", AccountName: string([]byte{0xff, 0xfe})},
		"control in plan":  {AccessToken: "access", UID: "uid", Plan: "Pro\n"},
		"long plan":        {AccessToken: "access", UID: "uid", Plan: strings.Repeat("p", maxPlanBytes+1)},
	}
	for name, credentials := range cases {
		if credentials.Valid() {
			t.Errorf("%s accepted", name)
		}
	}
	if (Credentials{AccessToken: "access", UID: "uid"}).SignedIn() != true {
		t.Fatal("access token not reported as signed in")
	}
	if (Credentials{UID: "uid"}).SignedIn() {
		t.Fatal("missing access token reported as signed in")
	}
}

func TestCredentialsMarshalParseRoundTrip(t *testing.T) {
	original := testCredentials()
	original.ExpiresAt = time.Unix(1900000000, 0).UTC()
	original.RefreshExpiresAt = time.Unix(1900086400, 0).UTC()
	original.TokenType = "Bearer"
	original.EnterpriseID = "ent-1"
	data, err := original.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > maxCredentialBytes {
		t.Fatalf("marshalled %d bytes", len(data))
	}
	if !strings.Contains(string(data), `"accessToken":"access-token-1"`) || !strings.Contains(string(data), `"uid":"uid-1"`) {
		t.Fatalf("marshalled body = %s", data)
	}
	parsed, err := ParseCredentials(data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed != original {
		t.Fatalf("round trip = %+v", parsed)
	}
	if !parsed.ExpiresAt.Equal(original.ExpiresAt) {
		t.Fatalf("expiry = %v", parsed.ExpiresAt)
	}
	clone := parsed.Clone()
	if clone != parsed {
		t.Fatalf("clone = %+v", clone)
	}
}

func TestCredentialsMarshalRejectsInvalid(t *testing.T) {
	if _, err := (Credentials{UID: "uid"}).Marshal(); ErrorCode(err) != CodeCredentialUnavailable {
		t.Fatalf("marshal invalid => %v", err)
	}
}

func TestParseCredentialsRejectsBadBodies(t *testing.T) {
	validAccess := `{"accessToken":"access","uid":"uid"}`
	cases := map[string][]byte{
		"empty":          nil,
		"oversized":      make([]byte, maxCredentialBytes+1),
		"not json":       []byte(`{`),
		"array":          []byte(`[]`),
		"missing uid":    []byte(`{"accessToken":"access"}`),
		"missing access": []byte(`{"uid":"uid"}`),
		"bad access":     []byte(`{"accessToken":"bad token","uid":"uid"}`),
		"control char":   []byte(`{"accessToken":"access","uid":"uid","accountName":"a\u0000b"}`),
		"null":           []byte(`null`),
	}
	for name, data := range cases {
		if _, err := ParseCredentials(data); ErrorCode(err) != CodeCredentialUnavailable {
			t.Errorf("%s => %v", name, err)
		}
	}
	parsed, err := ParseCredentials([]byte(validAccess))
	if err != nil || parsed.UID != "uid" || parsed.AccessToken != "access" {
		t.Fatalf("valid body => (%+v, %v)", parsed, err)
	}
	if _, ok := mustJSONMap(t, []byte(validAccess))["accessToken"]; !ok {
		t.Fatal("test fixture lost accessToken")
	}
}

func mustJSONMap(t *testing.T, data []byte) map[string]json.RawMessage {
	t.Helper()
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	return fields
}
