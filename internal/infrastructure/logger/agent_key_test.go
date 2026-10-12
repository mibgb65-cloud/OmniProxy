package logger

import (
	"strings"
	"testing"
)

func TestLocalAgentKeysAreRedactedInURLs(t *testing.T) {
	key := "opk_" + strings.Repeat("a", 64)
	redactor := NewRedactor()
	if strings.Contains(redactor.Redact("ws://127.0.0.1:3001/agent/codex/"+key+"/v1/responses"), key) {
		t.Fatal("local key leaked")
	}
}
