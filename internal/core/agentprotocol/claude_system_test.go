package agentprotocol

import (
	"encoding/json"
	"testing"
)

func TestClaudeCodeSystemReminderPreservesPositionAndText(t *testing.T) {
	for _, content := range []string{`"reminder"`, `[{"type":"text","text":"reminder","cache_control":{"type":"ephemeral"}}]`} {
		body := []byte(`{"model":"deepseek-flash","system":[{"type":"text","text":"instructions"}],"max_tokens":32000,"messages":[{"role":"user","content":"hello"},{"role":"system","content":` + content + `}]}`)
		chat, _, err := ToChat(body, "messages")
		if err != nil {
			t.Fatal("Claude Code system reminder rejected", err)
		}
		var result struct {
			Messages []struct{ Role, Content string }
		}
		if json.Unmarshal(chat, &result) != nil || len(result.Messages) != 3 || result.Messages[0].Role != "system" || result.Messages[0].Content != "instructions" || result.Messages[1].Role != "user" || result.Messages[2].Role != "system" || result.Messages[2].Content != "reminder" {
			t.Fatal("system reminder changed role, order or text")
		}
	}
	for _, content := range []string{`[{"type":"tool_use","id":"call","name":"Read","input":{}}]`, `[{"type":"image"}]`, `{}`, `42`} {
		body := []byte(`{"messages":[{"role":"system","content":` + content + `}]}`)
		if _, _, err := ToChat(body, "messages"); err == nil {
			t.Fatal("unsupported system content silently accepted")
		}
	}
}
