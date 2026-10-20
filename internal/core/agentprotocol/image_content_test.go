package agentprotocol

import (
	"errors"
	"testing"
)

func TestImagesInClaudeToolResultsAndHistoryAreNotSilentlyRemoved(t *testing.T) {
	for _, content := range []string{
		`[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"private-image"}}]`,
		`[{"type":"tool_result","tool_use_id":"call_1","content":[{"type":"text","text":"private-tool-result"},{"type":"image","source":{"type":"base64","media_type":"image/png","data":"private-image"}}]}]`,
	} {
		source := []byte(`{"model":"deepseek-flash","messages":[{"role":"user","content":` + content + `},{"role":"assistant","content":"old reply"},{"role":"user","content":"text only now"}]}`)
		body, _, err := ToChat(source, "messages")
		if !errors.Is(err, ErrUnsupportedImage) || !errors.Is(err, ErrUnsupported) || body != nil {
			t.Fatal("image or image-bearing tool result was lost instead of explicitly rejected")
		}
	}
}
