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
	for _, key := range []string{"service_tier", "metadata", "store"} {
		if _, exists := fields[key]; exists {
			t.Fatalf("%s not removed", key)
		}
	}
	// 档位必须在路由选型前保留，否则无法据此选择上游档位变体。
	if _, exists := fields["reasoning_effort"]; !exists {
		t.Fatal("reasoning_effort removed before routing")
	}
	if _, exists := fields["model"]; !exists {
		t.Fatal("model removed")
	}
	other, err := NormalizeClientBody("opencode", body)
	if err != nil || string(other) != string(body) {
		t.Fatal("other client body changed")
	}
}

func TestStripClientEffortRunsAfterRouting(t *testing.T) {
	chat := []byte(`{"model":"gemini-4-argon-high","messages":[{"role":"user","content":"hi"}],"reasoning_effort":"high","temperature":0.2}`)
	got, err := StripClientEffort("antigravity", "chat", chat)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(got, &fields) != nil {
		t.Fatal("invalid stripped body")
	}
	if _, exists := fields["reasoning_effort"]; exists {
		t.Fatal("reasoning_effort forwarded to the subscription adapter")
	}
	for _, key := range []string{"model", "messages", "temperature"} {
		if _, exists := fields[key]; !exists {
			t.Fatalf("%s removed with the effort field", key)
		}
	}
	// Messages 协议把档位放在 output_config.effort，只删除该字段，容器其余内容保留。
	messages := []byte(`{"model":"claude-opus-4-6","output_config":{"effort":"high","format":"text"},"messages":[]}`)
	stripped, err := StripClientEffort("antigravity", "messages", messages)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		OutputConfig map[string]json.RawMessage `json:"output_config"`
	}
	if json.Unmarshal(stripped, &parsed) != nil {
		t.Fatal("invalid messages body")
	}
	if _, exists := parsed.OutputConfig["effort"]; exists {
		t.Fatal("messages effort forwarded")
	}
	if _, exists := parsed.OutputConfig["format"]; !exists {
		t.Fatal("unrelated output_config field removed")
	}
	// 其他客户端的正文原样返回。
	plain := []byte(`{"model":"gemini-4-argon-high","reasoning_effort":"high"}`)
	untouched, err := StripClientEffort("codex", "chat", plain)
	if err != nil || string(untouched) != string(plain) {
		t.Fatal("other client body changed")
	}
}

func TestAntigravityParseKeepsChatProtocol(t *testing.T) {
	body := []byte(`{"model":"gemini-4-argon","messages":[{"role":"user","content":"hi"}],"reasoning_effort":"high","stream":true}`)
	request, err := ParseClientRequest("antigravity", "/v1/chat/completions", body)
	if err != nil {
		t.Fatal(err)
	}
	if request.Protocol != "chat" {
		t.Fatalf("antigravity protocol = %q，转发层会判定为不支持", request.Protocol)
	}
	if request.Effort != "high" || !request.Stream {
		t.Fatalf("effort/stream lost: %q/%v", request.Effort, request.Stream)
	}
	if parsed, err := ParseRequest(body); err != nil || parsed.Protocol != "chat" {
		t.Fatalf("plain chat protocol = %q, %v", parsed.Protocol, err)
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
