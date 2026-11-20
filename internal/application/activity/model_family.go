package activity

import (
	"context"
	"database/sql"
	"sort"
	"strings"

	catalog "omniproxy/internal/core/modelcatalog"
)

// modelFamily 把一个模型的推理档位变体映射回基础模型名（`gemini-4-neon-high` → `gemini-4-neon`）。
//
// 上游用独立模型 ID 表达推理档位；它们在目录里是同一个模型的档位变体（ADR-022），
// 统计里也应该是一个模型的不同思考程度，不能各占一行。
// 是否合并只看证据，不按名字猜：
//   - 同一基础名已出现两个及以上档位后缀（上游确实按档位发布型号）；
//   - 或基础名本身也是一个已知模型（档位变体有明确的基础模型）。
func modelFamilies(names []string) map[string]string {
	levels := map[string]map[string]bool{}
	variants := map[string]string{}
	known := map[string]bool{}
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		known[strings.ToLower(trimmed)] = true
		base, level, ok := catalog.EffortVariantBaseName(trimmed)
		if !ok {
			continue
		}
		if levels[base] == nil {
			levels[base] = map[string]bool{}
		}
		levels[base][level] = true
		variants[trimmed] = base
	}
	families := map[string]string{}
	for variant, base := range variants {
		// 目录里的档位词汇只是候选：没有第二个档位、基础名也不存在时不认定它是档位。
		if len(levels[base]) < 2 && !known[strings.ToLower(base)] {
			continue
		}
		families[variant] = base
	}
	return families
}

// modelFamilyName 返回模型在统计中的显示名；未认定家族时保持原始 ID。
func modelFamilyName(families map[string]string, model string) string {
	if base, ok := families[strings.TrimSpace(model)]; ok {
		return base
	}
	return model
}

type modelQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// loadModelFamilies 汇总已记录用量与已同步目录里的模型 ID，再计算家族映射。
// 已记录模型覆盖历史（即使某型号已从目录消失），目录补充「已发布但还没用过」的档位变体。
// 目录是补充证据：旧库还没有目录表时只按已记录模型判断，不因此让整页用量读取失败。
func loadModelFamilies(ctx context.Context, db modelQuerier) (map[string]string, error) {
	names, err := queryModelNames(ctx, db, `SELECT model FROM usage_provider_models`)
	if err != nil {
		return nil, err
	}
	if catalogNames, catalogErr := queryModelNames(ctx, db, `SELECT model FROM account_models`); catalogErr == nil {
		names = append(names, catalogNames...)
	}
	return modelFamilies(names), nil
}

func queryModelNames(ctx context.Context, db modelQuerier, query string) ([]string, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	names := []string{}
	for rows.Next() {
		var name string
		if rows.Scan(&name) != nil {
			return nil, ErrUnavailable
		}
		names = append(names, name)
	}
	if rows.Err() != nil {
		return nil, ErrUnavailable
	}
	return names, nil
}

// modelTotals 是归并过程中的整数累计；只在内部使用，输出仍走 summary 的十进制字符串。
type modelTotals struct {
	input    int64
	output   int64
	requests int64
	complete int64
}

func (t *modelTotals) add(input, output, requests, complete int64) {
	t.input += input
	t.output += output
	t.requests += requests
	t.complete += complete
}

func (t modelTotals) summary() Summary {
	return summary(t.input, t.output, t.requests, t.complete)
}

func (t modelTotals) weightsMore(other modelTotals) bool {
	return t.input+t.output > other.input+other.output
}

// variantNames 返回被合并的原始型号 ID；与显示名完全一致（没有归并）时不返回。
func variantNames(names map[string]bool, display string) []string {
	if len(names) == 0 {
		return nil
	}
	if len(names) == 1 {
		for name := range names {
			if name == display {
				return nil
			}
		}
	}
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}
