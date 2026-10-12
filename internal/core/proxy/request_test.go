package proxy

import (
	"strings"
	"testing"
)

func TestParseRequestRejectsAmbiguousOrInvalidInputs(t *testing.T) {
	for _, body := range []string{
		`null`, `[]`, validBody + `{}`, `{"model":"deepseek-flash","model":"deepseek-v4-pro","messages":[]}`,
		`{"model":"unsafe\nmodel","messages":[{"role":"user","content":"hello"}]}`,
		`{"model":"deepseek-flash","messages":[]}`,
		`{"model":"deepseek-flash","messages":[{"role":"unknown","content":"hello"}]}`,
		`{"model":"deepseek-flash","messages":[{"role":"user","content":null}]}`,
		`{"model":"deepseek-flash","messages":[{"role":"user","content":{"x":"hello"}}]}`,
		strings.TrimSuffix(validBody, "}") + `,"stream":"true"}`,
		strings.TrimSuffix(validBody, "}") + `,"max_tokens":0}`,
		strings.TrimSuffix(validBody, "}") + `,"max_tokens":1.5}`,
		strings.TrimSuffix(validBody, "}") + `,"temperature":3}`,
		strings.TrimSuffix(validBody, "}") + `,"top_p":0}`,
		strings.TrimSuffix(validBody, "}") + `,"extra":` + strings.Repeat("[", 34) + `0` + strings.Repeat("]", 34) + `}`,
	} {
		if _, err := ParseRequest([]byte(body)); err == nil {
			t.Errorf("accepted invalid body: %.100s", body)
		}
	}
}

func TestParseRequestPreservesProviderOptions(t *testing.T) {
	body := strings.TrimSuffix(validBody, "}") + `,"stream":true,"thinking":{"type":"disabled"},"max_tokens":16}`
	req, err := ParseRequest([]byte(body))
	if err != nil || !req.Stream || req.Model != "deepseek-flash" || string(req.Body) != body {
		t.Fatalf("unexpected parsed request: %+v %v", req, err)
	}
}
