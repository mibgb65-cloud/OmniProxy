package proxy

import "encoding/json"

func responseProtocol(request Request) string {
	if request.Compaction {
		return "responses/compact"
	}
	return request.Protocol
}

// 压缩结果是不可解释的后续上下文；只校验外层类型，不能解密、改写或把普通回答冒充压缩。
func validCompactionResult(body []byte) bool {
	var result struct {
		Object string `json:"object"`
		Output []struct {
			Type      string `json:"type"`
			Encrypted string `json:"encrypted_content"`
		} `json:"output"`
	}
	if json.Unmarshal(body, &result) != nil || result.Object != "response.compaction" {
		return false
	}
	for _, item := range result.Output {
		if item.Type == "compaction" && item.Encrypted != "" {
			return true
		}
	}
	return false
}
