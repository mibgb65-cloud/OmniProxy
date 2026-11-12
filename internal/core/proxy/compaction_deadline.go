package proxy

import (
	"encoding/json"
	"time"
)

// IsCompaction 识别固定端点和经过校验的 V2 元数据，不按正文或仅按 beta 功能猜测。
func (request Request) IsCompaction() bool {
	if request.Protocol != "responses" {
		return false
	}
	if request.Compaction {
		return true
	}
	if request.CodexMetadata.BetaFeatures != codexCompactionV2 || !ValidCodexMetadata(request.CodexMetadata) {
		return false
	}
	var metadata struct {
		Compaction struct {
			Implementation string `json:"implementation"`
		} `json:"compaction"`
	}
	return json.Unmarshal([]byte(request.CodexMetadata.TurnMetadata), &metadata) == nil &&
		metadata.Compaction.Implementation == "responses_compaction_v2"
}

func (s *Server) requestTimeout(request Request) time.Duration {
	timeout := s.opts.RequestTimeout
	// 长历史压缩在响应头之后也可能数分钟无正文；只放宽明确压缩，普通推理时限不变。
	const compactionTimeout = 10 * time.Minute
	if request.IsCompaction() && timeout < compactionTimeout {
		return compactionTimeout
	}
	return timeout
}

// antigravityTimeout 给桌面代理会话的推理留出时间。
//
// Antigravity 带着完整工具集和历史发送请求，订阅的思考型模型可能先想数分钟再输出；
// 沿用普通请求的 120 秒会把正常会话截断成 502。这里只放宽这个客户端，其它客户端不变。
const antigravityTimeout = 10 * time.Minute

func (s *Server) clientRequestTimeout(client string, request Request) time.Duration {
	timeout := s.requestTimeout(request)
	if client == "antigravity" && timeout < antigravityTimeout {
		return antigravityTimeout
	}
	return timeout
}
