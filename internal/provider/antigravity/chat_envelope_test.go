package antigravity

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

func envelopeOf(t *testing.T, request proxy.Request, asWritten bool) map[string]any {
	t.Helper()
	built, err := chatPayload(request, testCredentials(), asWritten)
	if err != nil {
		t.Fatalf("payload: %v", err)
	}
	var envelope map[string]any
	if err := json.Unmarshal(built.body, &envelope); err != nil {
		t.Fatalf("envelope: %v", err)
	}
	native, _ := envelope["request"].(map[string]any)
	if native == nil {
		t.Fatal("envelope has no request")
	}
	return native
}

func chatRequest(model, body string) proxy.Request {
	return proxy.Request{Protocol: "chat", Model: model, Body: []byte(body)}
}

func TestGenerationOptionsByModelFamily(t *testing.T) {
	const tool = `,"tools":[{"type":"function","function":{"name":"look","parameters":{"type":"object"}}}]`
	for _, test := range []struct {
		name        string
		model       string
		effort      string
		maxTokens   int
		wantOutput  int
		wantLevel   string
		wantBudget  int
		wantThought bool
	}{
		{"gemini3_level", "gemini-3.8-flash-high", "high", 32000, 0, "high", 0, true},
		// Hydrogen 等新一代模型自带 -xhigh 变体；档位必须原样透传，不能降成 high。
		{"gemini4_xhigh", "gemini-4-hydrogen-xhigh", "xhigh", 32000, 0, "xhigh", 0, true},
		{"gemini3_medium_pro", "gemini-3.1-pro-high", "medium", 32000, 0, "high", 0, true},
		{"gemini25_budget", "gemini-2.5-flash", "low", 32000, 0, "", 4096, true},
		{"claude_thinking_budget", "claude-opus-4-6-thinking", "high", 2000, 2000, "", 1999, true},
		{"claude_default_budget", "claude-opus-4-6-thinking", "", 0, 42000, "", 10000, true},
		{"claude_plain", "claude-sonnet-4-6", "", 64, 64, "", 0, false},
		{"gpt_oss_none", "gpt-oss-120b-medium", "high", 32000, 0, "", 0, false},
		{"gemini_without_effort", "gemini-3.8-flash-high", "", 32000, 0, "", 0, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := `{"messages":[{"role":"user","content":"hi"}]` + tool
			if test.maxTokens > 0 {
				body += `,"max_tokens":` + jsonNumber(test.maxTokens)
			}
			body += `}`
			request := chatRequest(test.model, body)
			request.Effort = test.effort
			native := envelopeOf(t, request, false)
			config, _ := native["generationConfig"].(map[string]any)
			output, hasOutput := config["maxOutputTokens"]
			if test.wantOutput == 0 && hasOutput || test.wantOutput > 0 && output != float64(test.wantOutput) {
				t.Fatalf("maxOutputTokens = %v, want %d", output, test.wantOutput)
			}
			thinking, hasThinking := config["thinkingConfig"].(map[string]any)
			if hasThinking != test.wantThought {
				t.Fatalf("thinkingConfig present = %t, want %t", hasThinking, test.wantThought)
			}
			if !hasThinking {
				return
			}
			if thinking["includeThoughts"] != true {
				t.Fatal("includeThoughts missing")
			}
			if test.wantLevel != "" && thinking["thinkingLevel"] != test.wantLevel {
				t.Fatalf("thinkingLevel = %v, want %s", thinking["thinkingLevel"], test.wantLevel)
			}
			if test.wantBudget != 0 && thinking["thinkingBudget"] != float64(test.wantBudget) {
				t.Fatalf("thinkingBudget = %v, want %d", thinking["thinkingBudget"], test.wantBudget)
			}
		})
	}
}

func jsonNumber(value int) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func TestToolChoiceModes(t *testing.T) {
	for _, test := range []struct {
		name  string
		model string
		raw   string
		mode  string
	}{
		{"auto_gemini", "gemini-3.8-flash", `"auto"`, "AUTO"},
		{"auto_claude_validated", "claude-opus-4-6-thinking", `"auto"`, "VALIDATED"},
		{"required", "gemini-3.8-flash", `"required"`, "ANY"},
		{"named", "gemini-3.8-flash", `{"type":"function","function":{"name":"look"}}`, "ANY"},
		{"none", "gemini-3.8-flash", `"none"`, "NONE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := `{"messages":[{"role":"user","content":"hi"}],"tools":[{"type":"function","function":{"name":"look","parameters":{"type":"object"}}}],"tool_choice":` + test.raw + `}`
			native := envelopeOf(t, chatRequest(test.model, body), false)
			config, _ := native["toolConfig"].(map[string]any)
			calling, _ := config["functionCallingConfig"].(map[string]any)
			if calling["mode"] != test.mode {
				t.Fatalf("mode = %v, want %s", calling["mode"], test.mode)
			}
		})
	}
}

func TestSessionIDFollowsTheFirstUserTurn(t *testing.T) {
	first := envelopeOf(t, chatRequest("gemini-3.8-flash", `{"messages":[{"role":"system","content":"s"},{"role":"user","content":"Weather in Paris?"}]}`), false)
	second := envelopeOf(t, chatRequest("gemini-3.8-flash", `{"messages":[{"role":"user","content":"Weather in Paris?"},{"role":"assistant","content":"Sunny"}]}`), false)
	if first["sessionId"] == "" || first["sessionId"] != second["sessionId"] {
		t.Fatalf("session id is not stable: %v vs %v", first["sessionId"], second["sessionId"])
	}
	other := envelopeOf(t, chatRequest("gemini-3.8-flash", `{"messages":[{"role":"user","content":"Other question"}]}`), false)
	if other["sessionId"] == first["sessionId"] {
		t.Fatal("different first turn shares the session id")
	}
}

func TestHistoryRepairsInsteadOfRejecting(t *testing.T) {
	startModel := envelopeOf(t, chatRequest("gemini-3.8-flash", `{"messages":[{"role":"assistant","content":"Hello"},{"role":"user","content":"Hi"}]}`), false)
	contents, _ := startModel["contents"].([]any)
	opening, _ := contents[0].(map[string]any)
	if opening["role"] != "user" || !strings.Contains(partsText(t, opening), continuedPrompt) {
		t.Fatal("history starting with the model was not repaired")
	}
	ended := envelopeOf(t, chatRequest("gemini-3.8-flash", `{"messages":[{"role":"user","content":"Hi"},{"role":"assistant","content":"Hello"}]}`), false)
	contents, _ = ended["contents"].([]any)
	last, _ := contents[len(contents)-1].(map[string]any)
	if last["role"] != "user" || !strings.Contains(partsText(t, last), modelLastPrompt) {
		t.Fatal("history ending with the model was not repaired")
	}
	unanswered := envelopeOf(t, chatRequest("gemini-3.8-flash", `{"messages":[{"role":"user","content":"Hi"},{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"look","arguments":"{}"}}]}]}`), false)
	contents, _ = unanswered["contents"].([]any)
	found := ""
	for _, value := range contents {
		content, _ := value.(map[string]any)
		parts, _ := content["parts"].([]any)
		for _, part := range parts {
			entry, _ := part.(map[string]any)
			if response, ok := entry["functionResponse"].(map[string]any); ok {
				found, _ = response["response"].(map[string]any)["result"].(string)
			}
		}
	}
	if found != interruptedToolNote {
		t.Fatalf("unanswered call was not answered: %q", found)
	}
	duplicate := envelopeOf(t, chatRequest("gemini-3.8-flash", `{"messages":[{"role":"user","content":"Hi"},{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"look","arguments":"{}"}}]},{"role":"tool","tool_call_id":"call_1","content":"one"},{"role":"tool","tool_call_id":"call_1","content":"two"}]}`), false)
	if got := strings.Count(mustJSON(t, duplicate["contents"]), `"functionResponse"`); got != 1 {
		t.Fatalf("duplicate answers were kept: %d", got)
	}
	merged := envelopeOf(t, chatRequest("gemini-3.8-flash", `{"messages":[{"role":"user","content":"one"},{"role":"user","content":"two"}]}`), false)
	if got := len(merged["contents"].([]any)); got != 1 {
		t.Fatalf("consecutive user turns were not merged: %d", got)
	}
}

func TestSystemPromptIsCloakedUnlessSentAsWritten(t *testing.T) {
	body := `{"messages":[{"role":"system","content":"` + refusalSystem + `"},{"role":"user","content":"hi"}]}`
	cloakedSystem := partsText(t, mustContent(t, envelopeOf(t, chatRequest("gemini-3.8-flash", body), false)["systemInstruction"].(map[string]any)))
	if strings.Contains(cloakedSystem, refusalSystem) || !strings.Contains(cloakedSystem, zeroWidth) {
		t.Fatal("refusal line was not cloaked by default")
	}
	raw := partsText(t, mustContent(t, envelopeOf(t, chatRequest("gemini-3.8-flash", body), true)["systemInstruction"].(map[string]any)))
	if !strings.Contains(raw, refusalSystem) || strings.Contains(raw, zeroWidth) {
		t.Fatal("as-written mode did not send the prompt unchanged")
	}
}

func TestUnsignedClaudeCallGetsSkipMarker(t *testing.T) {
	var sent map[string]any
	client := New(transportFunc(func(r *http.Request) (*http.Response, error) {
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &sent)
		return reply(http.StatusOK, `{"response":{"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}]}}`), nil
	}))
	request := chatRequest("claude-opus-4-6-thinking", `{"messages":[{"role":"user","content":"Hi"},{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"look","arguments":"{}"}}]},{"role":"tool","tool_call_id":"call_1","content":"one"}]}`)
	request.LastCandidate = true
	if err := forwardOnce(t, client, request); err != nil {
		t.Fatal(err)
	}
	contents := sent["request"].(map[string]any)["contents"].([]any)
	call := contents[1].(map[string]any)["parts"].([]any)[0].(map[string]any)
	if call["thoughtSignature"] != skipSignature {
		t.Fatalf("unsigned call was sent without the skip marker: %v", call["thoughtSignature"])
	}
}

func partsText(t *testing.T, content map[string]any) string {
	t.Helper()
	var out strings.Builder
	parts, _ := content["parts"].([]any)
	for _, part := range parts {
		entry, _ := part.(map[string]any)
		if text, ok := entry["text"].(string); ok {
			out.WriteString(text)
		}
	}
	return out.String()
}

func mustContent(t *testing.T, value any) map[string]any {
	t.Helper()
	content, ok := value.(map[string]any)
	if !ok {
		t.Fatal("missing content")
	}
	return content
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}
