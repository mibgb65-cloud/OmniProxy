package modelcatalog

// Codex 目录只接受来自 official catalog 的档位词汇。上游（River／MiniMax 等）
// 使用 none／max 表达两端，Codex 没有同名档位，必须就近映射后才能在选择器里出现：
// 未映射的档位不会显示，用户只能看到唯一一个已认识的值，表现为“无法选择推理强度”。
// official catalog 还会为 GPT 模型声明 ultra（最高强度 + 自动任务委派），
// 它本身已是客户端词汇，必须原样发布，不能被当成未知值丢弃。
var codexEffortLevels = []string{"minimal", "low", "medium", "high", "xhigh", "ultra"}

// codexEffortAlias 是上游档位 → Codex 档位的就近映射。
// none（关闭思考）在 Codex 侧用最低档 low 表达；max 用客户端最高的 xhigh。
var codexEffortAlias = map[string]string{"none": "low", "max": "xhigh"}

// CodexEfforts 返回可在 Codex 目录中发布的档位，未知取值一律不发布。
func CodexEfforts(efforts []Effort) []Effort {
	result := make([]Effort, 0, len(efforts))
	seen := map[string]bool{}
	for _, effort := range efforts {
		level := codexEffortLevel(effort.Value)
		if level == "" || seen[level] {
			continue
		}
		seen[level] = true
		result = append(result, Effort{Value: level, Description: effort.Description})
	}
	return result
}

// CodexUpstreamEffort 把客户端选择的 Codex 档位还原为该模型的上游取值。
// 只接受本模型已确认的档位；未确认时返回 false，由调用方明确报错而不是猜测。
func CodexUpstreamEffort(efforts []Effort, level string) (string, bool) {
	for _, effort := range efforts {
		if codexEffortLevel(effort.Value) == level && level != "" {
			return effort.Value, true
		}
	}
	return "", false
}

func codexEffortLevel(value string) string {
	if alias, ok := codexEffortAlias[value]; ok {
		return alias
	}
	for _, level := range codexEffortLevels {
		if level == value {
			return level
		}
	}
	return ""
}

// UpstreamEffort 把客户端发来的档位还原为该模型的上游取值。
//
// 客户端可能直接发上游原生取值（Claude 系按目录枚举选择），也可能发 Codex 目录
// 发布后的就近档位；两种都只在本模型已确认的枚举内接受，其余明确报错。
func UpstreamEffort(efforts []Effort, value string) (string, bool) {
	for _, effort := range efforts {
		if effort.Value == value {
			return value, true
		}
	}
	return CodexUpstreamEffort(efforts, value)
}
