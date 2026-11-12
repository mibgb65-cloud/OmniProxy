package proxy

import (
	"encoding/json"
	"testing"
)

func TestNormalizeClientBodyDropsOnlyAntigravityFields(t *testing.T) {
	body := []byte(`{"model":"gemini-3.8-flash","messages":[{"role":"user","content":"hi"}],"reasoning_effort":"low","service_tier":"priority","metadata":{},"store":false}`)
	got, err := NormalizeClientBody("antigravity", body)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(got, &fields) != nil {
		t.Fatal("invalid normalized body")
	}
	for _, key := range []string{"reasoning_effort", "service_tier", "metadata", "store"} {
		if _, exists := fields[key]; exists {
			t.Fatalf("%s not removed", key)
		}
	}
	if _, exists := fields["model"]; !exists {
		t.Fatal("model removed")
	}
	other, err := NormalizeClientBody("opencode", body)
	if err != nil || string(other) != string(body) {
		t.Fatal("other client body changed")
	}
}

func TestAntigravityModelFormatAccepted(t *testing.T) {
	if !validModelQuery("/v1/models", "format=antigravity") {
		t.Fatal("antigravity catalog format rejected")
	}
	if validModelQuery("/v1/models", "format=other") {
		t.Fatal("unknown catalog format accepted")
	}
	if validModelQuery("/v1/chat/completions", "format=antigravity") {
		t.Fatal("catalog format accepted on chat path")
	}
}
