package proxy

import (
	"strings"
	"testing"
)

func TestResponsesAcceptsImageBodyAboveLegacyTextLimit(t *testing.T) {
	// 只验证透明 JSON 通道和大小，不将占位 Base64 作为有效图片验收。
	body := []byte(`{"model":"gpt-test","stream":true,"input":[{"role":"user","content":[{"type":"input_image","image_url":"data:image/png;base64,` + strings.Repeat("A", 3<<20) + `"}]}]}`)
	request, err := ParseProtocolRequest(body, "/v1/responses")
	if err != nil {
		t.Fatal("Responses image body rejected above legacy 2 MiB limit", err)
	}
	if string(request.Body) != string(body) {
		t.Fatal("image body was changed")
	}
	if _, err := ParseRequest(body); err == nil {
		t.Fatal("Responses capacity leaked into Chat Completions")
	}
}
