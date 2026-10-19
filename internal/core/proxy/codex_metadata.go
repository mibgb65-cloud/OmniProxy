package proxy

import (
	"encoding/json"
	"net/http"
	"strings"
)

const codexCompactionV2 = "remote_compaction_v2"
const maxCodexMetadataBytes = 8192

// CodexMetadata 只承载订阅 Responses 的协议标记，不得进入日志或用于认证／上游地址选择。
type CodexMetadata struct{ BetaFeatures, TurnMetadata string }

// ValidCodexMetadata 限定功能枚举和有界 JSON，防止头注入及重复字段的解释差异。
func ValidCodexMetadata(value CodexMetadata) bool {
	if value.BetaFeatures != "" && value.BetaFeatures != codexCompactionV2 {
		return false
	}
	if value.TurnMetadata == "" {
		return true
	}
	if value.BetaFeatures == "" || len(value.TurnMetadata) > maxCodexMetadataBytes {
		return false
	}
	for _, ch := range []byte(value.TurnMetadata) {
		if ch < 32 || ch == 127 {
			return false
		}
	}
	if validateJSONSize([]byte(value.TurnMetadata), maxCodexMetadataBytes) != nil {
		return false
	}
	var object map[string]json.RawMessage
	return json.Unmarshal([]byte(value.TurnMetadata), &object) == nil && object != nil
}

func readCodexMetadata(header http.Header) (CodexMetadata, error) {
	var result CodexMetadata
	if len(header.Values("X-Codex-Beta-Features")) > 1 || len(header.Values("X-Codex-Turn-Metadata")) > 1 {
		return result, Failure("invalid_request", 400)
	}
	flags := header.Get("X-Codex-Beta-Features")
	if !ValidClientUserAgent(flags) {
		return result, Failure("invalid_request", 400)
	}
	for _, flag := range strings.FieldsFunc(flags, func(r rune) bool { return r == ',' || r == ' ' }) {
		if flag == codexCompactionV2 {
			result.BetaFeatures = codexCompactionV2
		}
	}
	if result.BetaFeatures != "" {
		result.TurnMetadata = header.Get("X-Codex-Turn-Metadata")
	}
	if !ValidCodexMetadata(result) {
		return CodexMetadata{}, Failure("invalid_request", 400)
	}
	return result, nil
}

// WebSocket 后续帧优先使用本帧元数据，不能把上个压缩回合的标记套到下一次推理。
func frameCodexMetadata(base CodexMetadata, raw json.RawMessage) (CodexMetadata, error) {
	if base.BetaFeatures == "" || len(raw) == 0 {
		return base, nil
	}
	var values map[string]json.RawMessage
	if json.Unmarshal(raw, &values) != nil {
		return CodexMetadata{}, Failure("invalid_request", 400)
	}
	if value, ok := values["x-codex-turn-metadata"]; ok {
		if json.Unmarshal(value, &base.TurnMetadata) != nil || !ValidCodexMetadata(base) {
			return CodexMetadata{}, Failure("invalid_request", 400)
		}
	}
	return base, nil
}
