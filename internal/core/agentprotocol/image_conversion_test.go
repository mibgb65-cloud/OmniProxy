package agentprotocol

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func decodeChat(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var request struct {
		Messages []map[string]any `json:"messages"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		t.Fatal(err)
	}
	return request.Messages
}

func TestMessagesImagesBecomeUpstreamImageURLBlocks(t *testing.T) {
	png := "data:image/png;base64,iVBORw0KGgo="
	source := []byte(`{"model":"test","messages":[
		{"role":"user","content":[
			{"type":"text","text":"describe"},
			{"type":"image","source":{"type":"base64","media_type":"image/png","data":"iVBORw0KGgo="}},
			{"type":"image","source":{"type":"url","url":"https://example.test/a.webp"}}
		]},
		{"role":"assistant","content":"ok"},
		{"role":"user","content":"now text only"}]}`)
	body, _, err := ToChat(source, "messages")
	if err != nil {
		t.Fatal(err)
	}
	messages := decodeChat(t, body)
	if len(messages) != 3 {
		t.Fatalf("message count changed: %d", len(messages))
	}
	blocks, ok := messages[0]["content"].([]any)
	if !ok || len(blocks) != 3 {
		t.Fatalf("image blocks not preserved: %v", messages[0]["content"])
	}
	first := blocks[1].(map[string]any)
	image := first["image_url"].(map[string]any)
	if first["type"] != "image_url" || image["url"] != png {
		t.Fatalf("base64 image not converted: %v", first)
	}
	second := blocks[2].(map[string]any)["image_url"].(map[string]any)
	if second["url"] != "https://example.test/a.webp" {
		t.Fatalf("url image changed: %v", second)
	}
	if messages[1]["content"] != "ok" || messages[2]["content"] != "now text only" {
		t.Fatal("plain text messages changed")
	}
}

func TestToolResultImagesSurviveConversion(t *testing.T) {
	source := []byte(`{"model":"test","messages":[{"role":"user","content":[
		{"type":"tool_result","tool_use_id":"shot","content":[
			{"type":"text","text":"screenshot"},
			{"type":"image","source":{"type":"base64","media_type":"image/jpeg","data":"/9j/4A=="}}
		]}]}]}`)
	body, _, err := ToChat(source, "messages")
	if err != nil {
		t.Fatal(err)
	}
	messages := decodeChat(t, body)
	if len(messages) != 1 || messages[0]["role"] != "tool" || messages[0]["tool_call_id"] != "shot" {
		t.Fatalf("tool association lost: %v", messages)
	}
	blocks, ok := messages[0]["content"].([]any)
	if !ok || len(blocks) != 2 {
		t.Fatalf("tool result image dropped: %v", messages[0]["content"])
	}
	if blocks[0].(map[string]any)["text"] != "screenshot" {
		t.Fatal("tool result text changed")
	}
	image := blocks[1].(map[string]any)["image_url"].(map[string]any)
	if image["url"] != "data:image/jpeg;base64,/9j/4A==" {
		t.Fatalf("tool result image not converted: %v", image)
	}
}

func TestResponsesImageInputKeepsDetailAndToolOutput(t *testing.T) {
	source := []byte(`{"model":"test","input":[
		{"role":"user","content":[
			{"type":"input_text","text":"look"},
			{"type":"input_image","image_url":"https://example.test/b.png","detail":"low"}]},
		{"type":"function_call_output","call_id":"c1","output":[
			{"type":"input_text","text":"captured"},
			{"type":"input_image","image_url":"data:image/png;base64,iVBORw0KGgo=","detail":"high"}]}]}`)
	body, _, err := ToChat(source, "responses")
	if err != nil {
		t.Fatal(err)
	}
	messages := decodeChat(t, body)
	if len(messages) != 2 {
		t.Fatalf("item count changed: %d", len(messages))
	}
	user := messages[0]["content"].([]any)[1].(map[string]any)["image_url"].(map[string]any)
	if user["url"] != "https://example.test/b.png" || user["detail"] != "low" {
		t.Fatalf("detail lost: %v", user)
	}
	tool := messages[1]["content"].([]any)
	if messages[1]["tool_call_id"] != "c1" || len(tool) != 2 {
		t.Fatalf("tool output image dropped: %v", messages[1])
	}
	if tool[1].(map[string]any)["image_url"].(map[string]any)["detail"] != "high" {
		t.Fatal("tool output detail lost")
	}
}

func TestUnverifiableImageFormsFailExplicitly(t *testing.T) {
	for _, content := range []string{
		`[{"type":"image","source":{"type":"file","file_id":"file-api-1"}}]`,
		`[{"type":"image","source":{"type":"base64","media_type":"image/tiff","data":"AAAA"}}]`,
		`[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"not base64!"}}]`,
		`[{"type":"image","source":{"type":"base64","media_type":"image/png","data":""}}]`,
		`[{"type":"image"}]`,
		`[{"type":"input_file","file_id":"f1"}]`,
		`[{"type":"input_audio","input_audio":{"data":"AAAA"}}]`,
	} {
		source := []byte(`{"model":"test","messages":[{"role":"user","content":` + content + `}]}`)
		if _, _, err := ToChat(source, "messages"); !errors.Is(err, ErrUnsupportedImage) && !errors.Is(err, ErrUnsupported) {
			t.Fatalf("unverifiable content accepted: %s / %v", content, err)
		}
	}
}

func TestOversizedImageBase64Rejected(t *testing.T) {
	data := strings.Repeat("A", imageLimit+1)
	source := []byte(`{"model":"test","messages":[{"role":"user","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"` + data + `"}}]}]}`)
	if _, _, err := ToChat(source, "messages"); !errors.Is(err, ErrUnsupportedImage) {
		t.Fatal("oversized image accepted")
	}
}

func TestSystemImagesStillFailInsteadOfBeingDropped(t *testing.T) {
	source := []byte(`{"model":"test","system":[{"type":"image","source":{"type":"url","url":"https://example.test/a.png"}}],"messages":[{"role":"user","content":"hi"}]}`)
	if _, _, err := ToChat(source, "messages"); !errors.Is(err, ErrUnsupportedImage) {
		t.Fatal("system image silently dropped")
	}
}
