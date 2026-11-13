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

// system／assistant／工具结果里的图片仍按能力边界拒绝：这些位置无法安全地转成上游形状。
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
