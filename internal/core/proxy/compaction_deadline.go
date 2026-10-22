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
