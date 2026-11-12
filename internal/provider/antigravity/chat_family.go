package antigravity

import (
	"strconv"
	"strings"
)

// 模型族：同一段会话整形逻辑，但生成参数、工具配置与签名规则按族区分。
const (
	familyGemini = "gemini"
	familyClaude = "claude"
	familyGPTOSS = "gpt-oss"
	familyOther  = "other"
)

// skipSignature 告诉 Google 跳过对「不是本轮生成」的调用签名校验：
// Claude 模型的工具调用没有 Google 签名，未签名的历史步骤同样需要它。
const skipSignature = "skip_thought_signature_validator"

// thinkingBudgets 是各推理档位在 Gemini 2.5 上的思考预算，与参考实现一致。
var thinkingBudgets = map[string]int{
	"low":    4096,
	"medium": 10000,
	"high":   24000,
	"xhigh":  32000,
	"max":    32000,
}

const defaultThinkingBudget = 10000

func familyOf(model string) string {
	name := strings.ToLower(model)
	switch {
	case strings.HasPrefix(name, "gemini-"):
		return familyGemini
	case strings.HasPrefix(name, "claude-"):
		return familyClaude
	case strings.HasPrefix(name, "gpt-oss"):
		return familyGPTOSS
	}
	return familyOther
}

// generationOf 是模型名里的 Gemini 主版本，0 表示名字里没有版本。
func generationOf(model string) int {
	name := strings.ToLower(model)
	index := strings.Index(name, "gemini-")
	if index < 0 {
		return 0
	}
	rest := name[index+len("gemini-"):]
	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0
	}
	version, _ := strconv.Atoi(rest[:end])
	return version
}

// budgetOf 是档位对应的思考预算；空档位沿用参考实现的默认值。
func budgetOf(effort string) int {
	if budget, ok := thinkingBudgets[strings.ToLower(strings.TrimSpace(effort))]; ok {
		return budget
	}
	return defaultThinkingBudget
}

// levelOf 是档位对应的 Gemini 3+ 思考级别：Flash 才接受 medium，其余落在 high。
func levelOf(effort, model string) string {
	name := strings.ToLower(model)
	switch strings.ToLower(strings.TrimSpace(effort)) {
	case "minimal", "none", "low":
		return "low"
	case "medium":
		if strings.Contains(name, "flash") || generationOf(model) > 3 {
			return "medium"
		}
		return "high"
	}
	return "high"
}

// thinkingOptions 按模型族决定 thinkingConfig：
//   - gpt-oss 与非 thinking 的 Claude 模型不下发；
//   - Gemini 3+ 用 thinkingLevel，2.5 用 thinkingBudget；
//   - Claude thinking 用预算，且不得吃掉输出上限。
//
// 客户端未指定档位时不下发档位，只保留 includeThoughts。
func thinkingOptions(effort, model, family string, maxTokens int64) object {
	level := strings.ToLower(strings.TrimSpace(effort))
	if family == familyGPTOSS {
		return nil
	}
	if family == familyClaude && !strings.Contains(strings.ToLower(model), "thinking") {
		return nil
	}
	if level == "" && family != familyClaude {
		return nil
	}
	config := object{"includeThoughts": true}
	if generationOf(model) >= 3 || strings.HasPrefix(strings.ToLower(model), "gemini-pro-agent") {
		if level != "" {
			config["thinkingLevel"] = levelOf(level, model)
		}
		return config
	}
	budget := budgetOf(level)
	if family == familyClaude && maxTokens > 0 && int64(budget) >= maxTokens {
		budget = int(maxTokens) - 1
		if budget < 1024 {
			return nil
		}
	}
	config["thinkingBudget"] = budget
	return config
}
