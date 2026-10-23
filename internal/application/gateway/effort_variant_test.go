package gateway

import (
	"strings"
	"testing"
)

func TestCarriedEffortReadsEachProtocol(t *testing.T) {
	cases := map[string]struct {
		protocol, body, want string
	}{
		"messages":  {"messages", `{"output_config":{"effort":"high","format":"kept"}}`, "high"},
		"responses": {"responses", `{"reasoning":{"effort":"low","summary":"auto"}}`, "low"},
		"chat":      {"chat", `{"reasoning_effort":"medium"}`, "medium"},
		"absent":    {"messages", `{"messages":[]}`, ""},
		"invalid":   {"messages", `{"output_config":[]}`, ""},
	}
	for name, tc := range cases {
		if got := carriedEffort(tc.protocol, []byte(tc.body)); got != tc.want {
			t.Fatalf("%s = %q, want %q", name, got, tc.want)
		}
	}
}

func TestRewriteEffortVariantReplacesModelAndDropsLevel(t *testing.T) {
	body, err := rewriteEffortVariant("messages", []byte(`{"model":"base","output_config":{"effort":"high","format":"kept"},"thinking":{"type":"adaptive"},"messages":[]}`), "base-high")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{`"model":"base-high"`, `"format":"kept"`, `"thinking"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("body missing %s: %s", want, text)
		}
	}
	if strings.Contains(text, `"effort"`) {
		t.Fatalf("effort field kept: %s", text)
	}

	body, err = rewriteEffortVariant("responses", []byte(`{"model":"base","reasoning":{"effort":"low"},"input":[]}`), "base-low")
	if err != nil || strings.Contains(string(body), `"effort"`) || !strings.Contains(string(body), `"model":"base-low"`) {
		t.Fatalf("responses rewrite failed: %s %v", body, err)
	}
	if strings.Contains(string(body), `"reasoning"`) {
		t.Fatalf("empty reasoning object kept: %s", body)
	}

	body, err = rewriteEffortVariant("chat", []byte(`{"model":"base","reasoning_effort":"medium","messages":[]}`), "base-medium")
	if err != nil || strings.Contains(string(body), `"reasoning_effort"`) || !strings.Contains(string(body), `"model":"base-medium"`) {
		t.Fatalf("chat rewrite failed: %s %v", body, err)
	}
}

// 目录折叠后，旧的档位变体 ID 必须还原成基础模型 + 档位，而不是 400。
func TestSetEffortVariantLevelRestoresBaseAndLevel(t *testing.T) {
	for _, tc := range []struct{ protocol, body string }{
		{"messages", `{"model":"gemini-3.8-flash-low","output_config":{"format":"kept"},"messages":[]}`},
		{"responses", `{"model":"gemini-3.8-flash-low","input":[]}`},
		{"chat", `{"model":"gemini-3.8-flash-low","messages":[]}`},
	} {
		body, err := setEffortVariantLevel(tc.protocol, []byte(tc.body), "gemini-3.8-flash", "low")
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		if !strings.Contains(text, `"model":"gemini-3.8-flash"`) || !strings.Contains(text, `"low"`) {
			t.Fatalf("%s: base model or level missing: %s", tc.protocol, text)
		}
		if tc.protocol == "messages" && !strings.Contains(text, `"format":"kept"`) {
			t.Fatalf("messages output_config lost other keys: %s", text)
		}
	}
}
