package proxy

import (
	"encoding/json"
	"strings"
	"testing"
)

func chatBodyWithModel(t *testing.T, model string, messages int) []byte {
	t.Helper()
	payload := map[string]any{
		"model":  model,
		"stream": true,
	}
	list := make([]map[string]string, 0, messages)
	for i := 0; i < messages; i++ {
		list = append(list, map[string]string{
			"role":    "user",
			"content": strings.Repeat("token ", 200),
		})
	}
	payload["messages"] = list
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestNormalizeRequestBodyModelRewritesOnlyWhenNeeded(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		body       string
		routeModel string
		want       string
		changed    bool
	}{
		{"already matching", `{"model":"gpt-5.5","stream":true}`, "gpt-5.5", "gpt-5.5", false},
		{"rewritten to route model", `{"model":"gpt-4","stream":true}`, "gpt-5.5", "gpt-5.5", true},
		{"alias normalised", `{"model":"deepseek-v4-pro[1m]"}`, "deepseek-v4-pro", "deepseek-v4-pro", true},
		{"absent model", `{"stream":true}`, "gpt-5.5", "", false},
		{"non-string model", `{"model":42}`, "gpt-5.5", "", false},
		{"empty route model", `{"model":"gpt-4"}`, "", "gpt-4", false},
	} {
		got, changed := normalizeRequestBodyModel([]byte(testCase.body), testCase.routeModel)
		if changed != testCase.changed {
			t.Fatalf("%s: expected changed=%v, got %v", testCase.name, testCase.changed, changed)
		}
		if !changed {
			if string(got) != testCase.body {
				t.Fatalf("%s: body must be returned untouched, got %s", testCase.name, got)
			}
			continue
		}
		var decoded struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
		}
		if err := json.Unmarshal(got, &decoded); err != nil {
			t.Fatalf("%s: %v", testCase.name, err)
		}
		if decoded.Model != testCase.want {
			t.Fatalf("%s: expected model %q, got %q", testCase.name, testCase.want, decoded.Model)
		}
	}
}

func TestNormalizeRequestBodyModelPreservesOtherFields(t *testing.T) {
	body := []byte(`{"model":"gpt-4","stream":true,"messages":[{"role":"user","content":"hi"}],"max_tokens":128}`)
	got, changed := normalizeRequestBodyModel(body, "gpt-5.5")
	if !changed {
		t.Fatal("expected the model to be rewritten")
	}
	var decoded map[string]any
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["model"] != "gpt-5.5" || decoded["stream"] != true {
		t.Fatalf("unexpected rewritten payload: %v", decoded)
	}
	if _, ok := decoded["messages"].([]any); !ok {
		t.Fatalf("messages were lost: %v", decoded)
	}
	if decoded["max_tokens"].(float64) != 128 {
		t.Fatalf("max_tokens was lost: %v", decoded)
	}
}

// The unchanged path runs on every forward attempt. Decoding the whole payload
// into map[string]any allocates once per JSON value, so guard against the probe
// being replaced by a full decode again.
func TestNormalizeRequestBodyModelDoesNotDecodeUnchangedBodies(t *testing.T) {
	body := chatBodyWithModel(t, "gpt-5.5", 40)

	allocs := testing.AllocsPerRun(20, func() {
		if _, changed := normalizeRequestBodyModel(body, "gpt-5.5"); changed {
			t.Fatal("expected no rewrite for a matching model")
		}
	})
	if allocs > 32 {
		t.Fatalf("expected the unchanged path to skip decoding the body, got %.0f allocations", allocs)
	}
}
