package codex

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestClientAuthFilePreservesOAuthAndRemovesAPIKey(t *testing.T) {
	original := []byte(`{"OPENAI_API_KEY":"synthetic-api","tokens":{"id_token":"synthetic-id","access_token":"synthetic-access","refresh_token":"synthetic-refresh","account_id":"synthetic-account"},"last_refresh":"2026-10-20T00:00:00Z"}`)
	normalized, err := ClientAuthFile(original, "synthetic-account")
	if err != nil {
		t.Fatal(err)
	}
	defer clear(normalized)
	var fields map[string]json.RawMessage
	if json.Unmarshal(normalized, &fields) != nil || string(fields["auth_mode"]) != `"chatgpt"` || fields["OPENAI_API_KEY"] != nil ||
		!bytes.Contains(normalized, []byte("synthetic-refresh")) || !bytes.Contains(normalized, []byte("2026-10-20T00:00:00Z")) {
		t.Fatal("OAuth file contract lost or API mode retained")
	}
}

func TestClientAuthFileRejectsIncompleteOrWrongIdentity(t *testing.T) {
	for _, input := range []string{
		`null`, `{}`, `{"auth_mode":"apikey"}`,
		`{"tokens":{"access_token":"synthetic","account_id":"synthetic-account"}}`,
		`{"tokens":{"id_token":"synthetic","access_token":"synthetic","refresh_token":"synthetic","account_id":"different"}}`,
		`{"tokens":{"id_token":"synthetic","access_token":"bad\nvalue","refresh_token":"synthetic","account_id":"synthetic-account"}}`,
		strings.Repeat("x", 128<<10+1),
	} {
		if _, err := ClientAuthFile([]byte(input), "synthetic-account"); err == nil || strings.Contains(err.Error(), "synthetic") {
			t.Fatal("invalid authorization accepted or exposed")
		}
	}
	if _, err := ClientAuthFile([]byte(`{}`), ""); err == nil {
		t.Fatal("unbound identity accepted")
	}
}
