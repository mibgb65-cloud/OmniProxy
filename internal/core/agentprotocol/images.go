package agentprotocol

import "strings"

// imageLimit 是单个图片字段的有界容量；Responses 已在入口限制 16 MiB，
// 这里防止转换过程放大或重复嵌套不必要的数据。
const imageLimit = 16 << 20

// chatImage 把 Messages `image` 块或 Responses `input_image` 块转成上游 Chat
// `image_url` 块。只接受已核对的形式，其他结构明确失败，不猜测也不丢弃。
//
// Messages：{"type":"image","source":{"type":"base64","media_type":"image/png","data":"..."}}
//
//	{"type":"image","source":{"type":"url","url":"https://..."}}
//
// Responses：{"type":"input_image","image_url":"data:image/png;base64,...","detail":"high"}
//
//	{"type":"input_image","image_url":{"url":"https://..."}}
func chatImage(part Object) (Object, error) {
	var url string
	switch text(part["type"]) {
	case "image":
		source := object(part["source"])
		if source == nil {
			return nil, ErrUnsupportedImage
		}
		switch text(source["type"]) {
		case "base64":
			media := text(source["media_type"])
			data := text(source["data"])
			if !imageMedia(media) || data == "" || len(data) > imageLimit || !base64Alpha(data) {
				return nil, ErrUnsupportedImage
			}
			url = "data:" + media + ";base64," + data
		case "url":
			url = text(source["url"])
		default:
			// Files API 文件引用、其他 provider 专有源无法验证，不做静默转换。
			return nil, ErrUnsupportedImage
		}
	case "input_image", "image_url":
		switch value := part["image_url"].(type) {
		case string:
			url = value
		case map[string]any:
			url = text(value["url"])
		}
	default:
		return nil, ErrUnsupportedImage
	}
	if url == "" || len(url) > imageLimit || strings.ContainsFunc(url, func(r rune) bool { return r < 0x20 }) {
		return nil, ErrUnsupportedImage
	}
	image := Object{"url": url}
	if detail := text(part["detail"]); detail != "" && detailValid(detail) {
		image["detail"] = detail
	}
	return Object{"type": "image_url", "image_url": image}, nil
}

// imageMedia 只接受 River 与 DeepSeek 官方公布的图片格式。
func imageMedia(value string) bool {
	switch value {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return true
	}
	return false
}

// detailValid 只透传官方公布的四档，其他值交由上游拒绝，不由本地改写语义。
func detailValid(value string) bool {
	switch value {
	case "low", "high", "auto", "original":
		return true
	}
	return false
}

// base64Alpha 验证 data 段只含 base64 字符，避免把任意文本当作图片转发。
func base64Alpha(value string) bool {
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '+' || c == '/' || c == '=' || c == '\n' || c == '\r' {
			continue
		}
		return false
	}
	return true
}

// chatBlocks 把内容块数组转成 Chat content 数组；没有图片时返回 nil，
// 调用方继续使用纯文本字符串形式，保持既有请求形态不变。
func chatBlocks(parts []any) ([]any, error) {
	blocks := make([]any, 0, len(parts))
	hasImage := false
	for _, entry := range parts {
		part := object(entry)
		switch text(part["type"]) {
		case "text", "input_text", "output_text":
			blocks = append(blocks, Object{"type": "text", "text": text(part["text"])})
		case "image", "input_image", "image_url":
			image, err := chatImage(part)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, image)
			hasImage = true
		default:
			return nil, ErrUnsupported
		}
	}
	if !hasImage {
		return nil, nil
	}
	return blocks, nil
}

// toolResultContent 保留工具结果的文本部分，并在含图片时给出完整 content 数组。
func toolResultContent(raw any) (string, []any, error) {
	if s, ok := raw.(string); ok {
		return s, nil, nil
	}
	parts := array(raw)
	if blocks, err := chatBlocks(parts); err != nil {
		return "", nil, err
	} else if blocks != nil {
		return "", blocks, nil
	}
	s, err := content(raw)
	if err != nil {
		return "", nil, err
	}
	return s, nil, nil
}
