package proxy

import "encoding/json"

// 未返回缓存明细时保持未知；尤其不能将旧版没有采集的计数补成零。
func cachedInputUsage(fields map[string]json.RawMessage, protocol string, usage UsageSnapshot) *int64 {
	if !usage.InputKnown {
		return nil
	}
	var raw json.RawMessage
	switch protocol {
	case "responses", "chat":
		key := "input_tokens_details"
		if protocol == "chat" {
			key = "prompt_tokens_details"
		}
		var details map[string]json.RawMessage
		if json.Unmarshal(fields[key], &details) == nil {
			raw = details["cached_tokens"]
		}
		if protocol == "chat" && len(raw) == 0 {
			raw = fields["prompt_cache_hit_tokens"]
		}
	case "messages":
		raw = fields["cache_read_input_tokens"]
	}
	n, ok := usageCount(raw)
	if !ok || n > usage.Input {
		return nil
	}
	return &n
}
