package proxy

import (
	"encoding/json"
	"strings"
	"testing"
)

// 上游已支持 image_url；Chat 入口必须保留图片块，同时不放宽纯文本归一化规则。
func TestChatKeepsImageBlocksAndStillNormalizesText(t *testing.T) {
	image := "data:image/png;base64,iVBORw0KGgo="
	body := `{"model":"deepseek-flash","messages":[
		{"role":"user","content":[{"type":"text","text":"hello"},{"type":"text","text":"world"},{"type":"image_url","image_url":{"url":"` + image + `","detail":"high"}}]},
		{"role":"assistant","content":"plain"},
		{"role":"user","content":[{"type":"text","text":"later"},{"type":"text","text":"question"}]}]}`
	request, err := ParseRequest([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Messages []map[string]any `json:"messages"`
	}
	if json.Unmarshal(request.Body, &config) != nil {
		t.Fatal("body became invalid JSON")
	}
	blocks, ok := config.Messages[0]["content"].([]any)
	if !ok || len(blocks) != 3 {
		t.Fatalf("image block dropped: %v", config.Messages[0]["content"])
	}
	entry := blocks[2].(map[string]any)
	if entry["type"] != "image_url" || entry["image_url"].(map[string]any)["url"] != image || entry["image_url"].(map[string]any)["detail"] != "high" {
		t.Fatalf("image block changed: %v", entry)
	}
	// 纯文本消息仍按既有规则合并成一个字符串。
	if config.Messages[2]["content"] != "later\nquestion" {
		t.Fatalf("text normalization regressed: %v", config.Messages[2]["content"])
	}
}

// 含图片但仍需拒绝未知块类型、越界数量和损坏字段。
func TestChatImageBlocksStillEnforceBounds(t *testing.T) {
	image := `{"type":"image_url","image_url":{"url":"data:image/png;base64,iVBORw0KGgo="}}`
	for _, content := range []string{
		`[` + image + `,{"type":"input_audio","input_audio":{"data":"AAAA"}}]`,
		`[{"type":"image_url"}]`,
		`[{"type":"image_url","image_url":{"url":""}}]`,
		`[{"type":"image_url","image_url":"data:image/png;base64,iVBORw0KGgo="}]`,
		`[` + strings.Repeat(image+`,`, 128) + image + `]`,
	} {
		if _, err := ParseProtocolRequest([]byte(`{"model":"deepseek-flash","messages":[{"role":"user","content":`+content+`}]}`), "/v1/chat/completions"); err == nil {
			t.Fatalf("unsupported image content accepted: %s", content)
		}
	}
}
