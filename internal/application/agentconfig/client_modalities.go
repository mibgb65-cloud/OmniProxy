package agentconfig

import (
	"slices"

	catalog "omniproxy/internal/core/modelcatalog"
)

// clientInputModalities 返回该模型在客户端目录中可安全声明的输入模态。
//
// 只公布上游已确认的图片能力；未声明或未确认时保持纯文本，绝不因同目录其他
// 模型支持图片就放宽。网关已按 TASK-20261023-003 转发 River 图片，客户端若不
// 声明 image 会在本地拦截附图，因此这里必须如实反映，而不是固定 text。
func clientInputModalities(entry catalog.Entry) []string {
	if slices.Contains(entry.Metadata.InputModalities, "image") {
		return []string{"text", "image"}
	}
	return []string{"text"}
}

// supportsClientImage 是 Goose `supports_vision` 这类布尔字段的判定。
func supportsClientImage(entry catalog.Entry) bool {
	return slices.Contains(entry.Metadata.InputModalities, "image")
}
