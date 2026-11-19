package antigravity

import (
	"strings"
	"time"
)

// QuotaGroupForModel 依据模型标识与官方分组名判断模型属于哪个额度分组。
//
// 官方只返回分组展示名（Gemini Models／Claude and GPT models／Gemini 4 Argon）和各窗口剩余，
// 不返回模型与分组的对应关系，因此只能按命名归类：
//   - 模型标识含 argon：Argon 是 Gemini 家族里的独立额度池，必须优先于 Gemini 主组；
//   - 模型以 claude／gpt 开头，或含 claude-／gpt-：第三方模型组；
//   - 其余含 gemini 的模型：Gemini 主组（不含 argon 的分组）。
//
// 无法判断时返回空字符串；调用方必须按「未知即不预筛」处理，不能猜测。
func QuotaGroupForModel(model string, groups []QuotaGroup) string {
	lower := strings.ToLower(strings.TrimSpace(model))
	if lower == "" {
		return ""
	}
	switch {
	case strings.Contains(lower, "argon"):
		return groupNamed(groups, "argon")
	case strings.HasPrefix(lower, "claude"), strings.HasPrefix(lower, "gpt"),
		strings.Contains(lower, "claude-"), strings.Contains(lower, "gpt-"):
		return groupNamed(groups, "claude", "gpt")
	case strings.Contains(lower, "gemini"):
		return geminiGroup(groups)
	}
	return ""
}

func groupNamed(groups []QuotaGroup, tokens ...string) string {
	for _, group := range groups {
		name := strings.ToLower(group.Name)
		for _, token := range tokens {
			if strings.Contains(name, token) {
				return group.Name
			}
		}
	}
	return ""
}

// geminiGroup 只返回 Gemini 主组：Argon 是独立额度池，不能并到主组。
func geminiGroup(groups []QuotaGroup) string {
	for _, group := range groups {
		name := strings.ToLower(group.Name)
		if strings.Contains(name, "gemini") && !strings.Contains(name, "argon") {
			return group.Name
		}
	}
	return ""
}

// QuotaExhaustedUntil 返回该模型所属分组最早可再用的时间。
//
// 组内任一窗口剩余为 0 时，必须等这些窗口全部重置，因此取其中最晚的重置时间；比例保留原始精度，
// 只有确为 0 才算耗尽。没有确为 0 的窗口、或耗尽窗口都缺少可解析的未来重置时间时返回 false：
// 无法确定恢复时间就不预筛，交给 429 的反应式冷却。
func QuotaExhaustedUntil(quota Quota, model string, now time.Time) (time.Time, bool) {
	group := QuotaGroupForModel(model, quota.Groups)
	if group == "" {
		return time.Time{}, false
	}
	var until time.Time
	found := false
	for _, item := range quota.Groups {
		if item.Name != group {
			continue
		}
		for _, bucket := range item.Buckets {
			if bucket.RemainingPercent == nil || *bucket.RemainingPercent > 0 {
				continue
			}
			reset, err := time.Parse(time.RFC3339Nano, bucket.ResetsAt)
			if err != nil || !reset.After(now) {
				continue
			}
			found = true
			if reset.After(until) {
				until = reset
			}
		}
	}
	if !found {
		return time.Time{}, false
	}
	return until, true
}
