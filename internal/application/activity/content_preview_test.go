package activity

import (
	"strings"
	"testing"
)

func TestContentPreviewExtractsConversationText(t *testing.T) {
	chat := `{"model":"m","messages":[
		{"role":"system","content":"very long system prompt"},
		{"role":"user","content":"你好，帮我看看这段代码"},
		{"role":"assistant","content":[{"type":"text","text":"可以，先发代码"}]}]}`
	preview := contentPreview(chat)
	if strings.Contains(preview, "system prompt") || !strings.Contains(preview, "你好") || !strings.Contains(preview, "先发代码") {
		t.Fatalf("chat preview = %q", preview)
	}
	responses := `{"input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"解释一下这个错误"}]}]}`
	if preview := contentPreview(responses); !strings.Contains(preview, "解释一下这个错误") {
		t.Fatalf("responses preview = %q", preview)
	}
	if preview := contentPreview(`{"input":"直接文本"}`); preview != "直接文本" {
		t.Fatalf("string input preview = %q", preview)
	}
	if preview := contentPreview(`{"messages":[{"role":"user","content":[{"type":"text","text":"第一行\n第二行"}]}]}`); preview != "第一行 第二行" {
		t.Fatalf("whitespace preview = %q", preview)
	}
	if preview := contentPreview("not json content"); preview != "not json content" {
		t.Fatalf("fallback preview = %q", preview)
	}
}

func TestContentPreviewBoundsLength(t *testing.T) {
	long := strings.Repeat("长", previewLimit+20)
	preview := contentPreview(`{"messages":[{"role":"user","content":"` + long + `"}]}`)
	if !strings.HasSuffix(preview, "…") || len([]rune(preview)) != previewLimit+1 {
		t.Fatalf("preview length %d", len([]rune(preview)))
	}
}
