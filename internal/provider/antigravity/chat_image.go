package antigravity

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"omniproxy/internal/core/proxy"
)

// chatImageLimit 是单个内联图片 base64 的长度上限，与其它协议的图片边界一致。
const chatImageLimit = 16 << 20

// chatImageMedia 只接受上游已公布的图片格式；其余明确失败，不猜测。
func chatImageMedia(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "image/png", "image/jpeg", "image/webp", "image/gif", "image/heic", "image/heif":
		return true
	}
	return false
}

// inlineImage 把 Chat 的 image_url 块转成 Gemini 的 inlineData part。
//
// 只接受 data URL：上游不会代为下载 http(s) 图片，静默丢弃或改成文本都会改变用户语义，
// 因此这类形式按图片能力边界明确失败。
func inlineImage(raw json.RawMessage) (object, error) {
	var block struct {
		Type     string          `json:"type"`
		ImageURL json.RawMessage `json:"image_url"`
	}
	if json.Unmarshal(raw, &block) != nil {
		return nil, proxy.Failure("unsupported_content", 400)
	}
	var url string
	if json.Unmarshal(block.ImageURL, &url) != nil {
		var nested struct {
			URL    string `json:"url"`
			Detail string `json:"detail"`
		}
		if json.Unmarshal(block.ImageURL, &nested) != nil {
			return nil, proxy.UnsupportedImage()
		}
		// detail 只影响本地视觉模型的采样精度，Gemini 不接受该字段。
		url = nested.URL
	}
	if !strings.HasPrefix(url, "data:") || len(url) > chatImageLimit {
		return nil, proxy.UnsupportedImage()
	}
	header, data, ok := strings.Cut(strings.TrimPrefix(url, "data:"), ",")
	if !ok || !strings.HasSuffix(strings.ToLower(header), ";base64") {
		return nil, proxy.UnsupportedImage()
	}
	media := header[:len(header)-len(";base64")]
	if !chatImageMedia(media) || data == "" || len(data) > chatImageLimit {
		return nil, proxy.UnsupportedImage()
	}
	if _, err := base64.StdEncoding.DecodeString(data); err != nil {
		return nil, proxy.UnsupportedImage()
	}
	return object{"inlineData": object{"mimeType": strings.ToLower(media), "data": data}}, nil
}

// userParts 把用户消息内容转成 Gemini parts：文本与内联图片都保留。
// 文本同时返回拼接结果，供 system 指纹与会话标识使用。
func userParts(raw json.RawMessage) (string, []object, error) {
	if string(raw) == "null" || len(raw) == 0 {
		return "", nil, nil
	}
	var value string
	if json.Unmarshal(raw, &value) == nil {
		if value == "" {
			return "", nil, nil
		}
		return value, []object{{"text": value}}, nil
	}
	var blocks []json.RawMessage
	if json.Unmarshal(raw, &blocks) != nil || len(blocks) > 1000 {
		return "", nil, proxy.Failure("unsupported_content", 400)
	}
	parts := []object{}
	texts := []string{}
	for _, block := range blocks {
		var head struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if json.Unmarshal(block, &head) != nil {
			return "", nil, proxy.Failure("unsupported_content", 400)
		}
		switch head.Type {
		case "text":
			if head.Text == "" {
				continue
			}
			texts = append(texts, head.Text)
			parts = append(parts, object{"text": head.Text})
		case "image_url", "input_image":
			image, err := inlineImage(block)
			if err != nil {
				return "", nil, err
			}
			parts = append(parts, image)
		default:
			return "", nil, proxy.Failure("unsupported_content", 400)
		}
	}
	return strings.Join(texts, ""), parts, nil
}
