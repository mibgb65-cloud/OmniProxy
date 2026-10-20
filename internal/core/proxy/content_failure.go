package proxy

// UnsupportedImage 保留既有稳定错误码，另用固定安全文案说明不可转换的图片历史。
// 不使用请求正文、图片地址或原始上游消息生成诊断信息。
func UnsupportedImage() error {
	return &Error{Code: "unsupported_content", StatusCode: 400, imageContent: true}
}

func (e *Error) clientMessage() string {
	if e.imageContent {
		return "unsupported_content: This model route cannot process images, including images in earlier messages or tool results. Switch to a vision-capable model, or start a new text-only conversation and avoid screenshot/image tools. No image content was removed or sent upstream."
	}
	return e.Code
}
