package antigravity

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

// 1x1 PNG 的真实 base64，避免用伪造数据绕过格式校验。
const tinyPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

func imageBody(content string) string {
	return `{"messages":[{"role":"user","content":` + content + `}]}`
}

func TestInlineImageBecomesUpstreamInlineData(t *testing.T) {
	native := envelopeOf(t, chatRequest("gemini-4-hydrogen-high", imageBody(
		`[{"type":"text","text":"what is this"},{"type":"image_url","image_url":{"url":"data:image/png;base64,`+tinyPNG+`"}}]`)), false)
	contents, _ := native["contents"].([]any)
	if len(contents) != 1 {
		t.Fatalf("contents = %d, want one turn", len(contents))
	}
	parts, _ := contents[0].(map[string]any)["parts"].([]any)
	if len(parts) != 2 {
		t.Fatalf("parts = %d, want text + image", len(parts))
	}
	if text, _ := parts[0].(map[string]any)["text"].(string); text != "what is this" {
		t.Fatalf("text part = %q", text)
	}
	inline, ok := parts[1].(map[string]any)["inlineData"].(map[string]any)
	if !ok {
		t.Fatalf("image part = %v", parts[1])
	}
	if inline["mimeType"] != "image/png" || inline["data"] != tinyPNG {
		t.Fatalf("inlineData = %v", inline)
	}
	// data: URL 前缀不能进上游。
	if strings.Contains(mustJSON(t, parts[1]), "data:") {
		t.Fatal("data URL prefix was forwarded")
	}
}

// 只发图片、不带文本的消息也是合法请求，不能被当成空消息拒绝。
func TestImageOnlyMessageIsAccepted(t *testing.T) {
	native := envelopeOf(t, chatRequest("gemini-4-hydrogen-high", imageBody(
		`[{"type":"image_url","image_url":"data:image/jpeg;base64,`+base64.StdEncoding.EncodeToString([]byte("jpeg-bytes"))+`"}]`)), false)
	contents, _ := native["contents"].([]any)
	parts, _ := contents[0].(map[string]any)["parts"].([]any)
	if len(parts) != 1 {
		t.Fatalf("parts = %v", parts)
	}
	inline, _ := parts[0].(map[string]any)["inlineData"].(map[string]any)
	if inline["mimeType"] != "image/jpeg" {
		t.Fatalf("mimeType = %v", inline["mimeType"])
	}
}

// 无法内联的图片必须明确失败，不能丢弃后当成纯文本请求。
func TestUnusableImagesFailExplicitly(t *testing.T) {
	for _, content := range []string{
		`[{"type":"image_url","image_url":{"url":"https://example.test/a.png"}}]`,
		`[{"type":"image_url","image_url":{"url":"data:image/tiff;base64,AAAA"}}]`,
		`[{"type":"image_url","image_url":{"url":"data:image/png;base64,not base64!"}}]`,
		`[{"type":"image_url","image_url":{"url":"data:image/png;base64,"}}]`,
		`[{"type":"image_url","image_url":{}}]`,
		`[{"type":"image_url"}]`,
	} {
		_, _, err := userParts([]byte(content))
		var failure *proxy.Error
		if !errors.As(err, &failure) || failure.Code != "unsupported_content" {
			t.Fatalf("%s accepted: %v", content, err)
		}
	}
}

// system／assistant 里的图片仍按能力边界拒绝：这些位置无法安全地转成上游形状。
// 工具结果的图片单独在 TestInlineImageInToolResultBecomesUserPart 里覆盖。
func TestImagesOutsideUserTurnsStayRejected(t *testing.T) {
	for _, body := range []string{
		`{"messages":[{"role":"system","content":[{"type":"image_url","image_url":{"url":"data:image/png;base64,` + tinyPNG + `"}}]},{"role":"user","content":"hi"}]}`,
		`{"messages":[{"role":"user","content":"hi"},{"role":"assistant","content":[{"type":"image_url","image_url":{"url":"data:image/png;base64,` + tinyPNG + `"}}]}]}`,
	} {
		if _, err := chatPayload(chatRequest("gemini-4-hydrogen-high", body), testCredentials(), false); err == nil {
			t.Fatalf("image outside a user turn accepted: %s", body)
		}
	}
}

// toolResultBody 构造一次读图工具轮：user 提出需求，assistant 调用 read_file，
// tool 结果带回文本与图片，最后一条 user 继续对话。
func toolResultBody(content string) string {
	return `{"messages":[` +
		`{"role":"user","content":"read before.png and compare it"},` +
		`{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"before.png\"}"}}]},` +
		`{"role":"tool","tool_call_id":"call_1","content":` + content + `},` +
		`{"role":"user","content":"what changed?"}]}`
}

// toolResultParts 找到包含 functionResponse 的那个 user 回合里的 parts。
func toolResultParts(t *testing.T, native map[string]any) []any {
	t.Helper()
	contents, _ := native["contents"].([]any)
	for _, value := range contents {
		turn, _ := value.(map[string]any)
		parts, _ := turn["parts"].([]any)
		for _, part := range parts {
			entry, _ := part.(map[string]any)
			if _, ok := entry["functionResponse"]; ok {
				return parts
			}
		}
	}
	t.Fatal("tool result turn missing")
	return nil
}

// agent 通过工具结果读到的图片必须变成同一 user 回合里 functionResponse 之后的
// inlineData，否则模型只能看到路径文本（Antigravity 读 PNG 只拿到占位提示）。
func TestInlineImageInToolResultBecomesUserPart(t *testing.T) {
	native := envelopeOf(t, chatRequest("gemini-4-hydrogen-high", toolResultBody(
		`[{"type":"text","text":"before.png"},{"type":"image_url","image_url":{"url":"data:image/png;base64,`+tinyPNG+`"}}]`)), false)
	parts := toolResultParts(t, native)
	if len(parts) != 2 {
		t.Fatalf("parts = %v, want functionResponse + inlineData", parts)
	}
	response, _ := parts[0].(map[string]any)["functionResponse"].(map[string]any)
	if response["id"] != "call_1" || response["name"] != "read_file" {
		t.Fatalf("functionResponse = %v", response)
	}
	if result, _ := response["response"].(map[string]any)["result"]; result != "before.png" {
		t.Fatalf("functionResponse result = %v", result)
	}
	inline, ok := parts[1].(map[string]any)["inlineData"].(map[string]any)
	if !ok || inline["mimeType"] != "image/png" || inline["data"] != tinyPNG {
		t.Fatalf("inlineData = %v", parts[1])
	}
	if strings.Contains(mustJSON(t, parts[1]), "data:") {
		t.Fatal("data URL prefix was forwarded in a tool result")
	}
}

// 只有图片、没有文本的工具结果也是合法结果，文本部分留空即可。
func TestImageOnlyToolResultIsAccepted(t *testing.T) {
	native := envelopeOf(t, chatRequest("gemini-4-hydrogen-high", toolResultBody(
		`[{"type":"image_url","image_url":{"url":"data:image/jpeg;base64,`+base64.StdEncoding.EncodeToString([]byte("jpeg-bytes"))+`"}}]`)), false)
	parts := toolResultParts(t, native)
	if len(parts) != 2 {
		t.Fatalf("parts = %v, want functionResponse + inlineData", parts)
	}
	response, _ := parts[0].(map[string]any)["functionResponse"].(map[string]any)
	if result, _ := response["response"].(map[string]any)["result"]; result != "" {
		t.Fatalf("empty tool text became %v", result)
	}
	inline, _ := parts[1].(map[string]any)["inlineData"].(map[string]any)
	if inline["mimeType"] != "image/jpeg" {
		t.Fatalf("mimeType = %v", inline["mimeType"])
	}
}

// 工具结果里无法内联的图片形式与未知内容块仍然明确失败，不能静默降级成文本。
func TestUnusableToolResultContentFailsExplicitly(t *testing.T) {
	for _, content := range []string{
		`[{"type":"image_url","image_url":{"url":"https://example.test/a.png"}}]`,
		`[{"type":"text","text":"ok"},{"type":"audio","audio":{"data":"x"}}]`,
	} {
		if _, err := chatPayload(chatRequest("gemini-4-hydrogen-high", toolResultBody(content)), testCredentials(), false); err == nil {
			t.Fatalf("tool result accepted: %s", content)
		}
	}
}

// 其它未知内容块不能被静默忽略。
func TestUnknownContentBlockStillFails(t *testing.T) {
	_, _, err := userParts([]byte(`[{"type":"text","text":"hi"},{"type":"audio","audio":{"data":"x"}}]`))
	var failure *proxy.Error
	if !errors.As(err, &failure) || failure.Code != "unsupported_content" {
		t.Fatalf("unknown block accepted: %v", err)
	}
}

// 历史里已有图片时必须能原样带回去，而不是只能在首轮使用。
func TestEarlierImageTurnsSurvive(t *testing.T) {
	body := `{"messages":[{"role":"user","content":[{"type":"text","text":"look"},{"type":"image_url","image_url":{"url":"data:image/png;base64,` + tinyPNG + `"}}]},{"role":"assistant","content":"a cat"},{"role":"user","content":"and now?"}]}`
	native := envelopeOf(t, chatRequest("gemini-4-hydrogen-high", body), false)
	if count := strings.Count(mustJSON(t, native["contents"]), "inlineData"); count != 1 {
		t.Fatalf("history image lost: %d", count)
	}
	var raw map[string]any
	_ = json.Unmarshal([]byte(mustJSON(t, native)), &raw)
}
