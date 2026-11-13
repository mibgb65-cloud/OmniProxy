# TASK-20261113-011: 变体型号档位优先在“只有变体、没有基础型号”的目录下失效

## Status

Completed

## Created

2026-11-13 11:40（Asia/Singapore）

## Request

> 还是有这个问题呢，你看看该怎么修复呢？是我的日志记录的问题吗？

截图：请求明细里 `gemini-4-hydrogen-xhigh` 的推理等级仍显示 `low`（2026-11-13 11:12）。

## Goal

- 目录里只有档位变体（没有基础型号）时，型号里的档位仍然生效：记录与上游
  `thinkingLevel` 都是 `xhigh`。
- 不改变家族名 + 档位、以及无法确认时保留客户端档位的行为。

## Background

TASK-20261113-004 的判定依赖 `catalog.EffortVariantBase`，它读的是**折叠后**目录里
`Metadata.EffortModels` 的映射。真实数据（本机生产库）是：

```text
gemini-4-hydrogen-high    Gemini 4 Hydrogen (High)
gemini-4-hydrogen-low     Gemini 4 Hydrogen (Low)
gemini-4-hydrogen-medium  Gemini 4 Hydrogen (Medium)
gemini-4-hydrogen-xhigh   Gemini 4 Hydrogen (XHigh)
```

上游只提供变体、**没有** `gemini-4-hydrogen` 这条基础型号，而 `FoldEffortVariants` 只有在
同名基础模型存在时才折叠（避免误合并不同型号），因此这些条目保持独立，`EffortModels` 为空，
`EffortVariantBase` 永远返回 false —— 档位优先的分支进不去，`request.Effort` 仍是客户端
正文里的 `low`。

## Scope

- `internal/core/modelcatalog/effort_variants.go`：新增从型号 ID 拆出基础名与档位的辅助函数。
- `internal/application/gateway/effort_variant.go`：目录未折叠时，用同族兄弟变体确认档位。
- 单元测试（只有变体的目录）。

## Out of Scope

- 不改目录折叠规则与前端模型列表。
- 不改其它客户端与协议。

## Existing Behavior

`gemini-4-hydrogen-xhigh` + `reasoning_effort: low` → 记录与下发都是 low。

## Expected Behavior

同族里存在 ≥2 个档位变体时，型号后缀即档位：记录与下发 xhigh。

---

# Analysis

## Related Modules

- `internal/core/modelcatalog/effort_variants.go`
- `internal/application/gateway/effort_variant.go`、`family_variant.go`

## Dependencies

无新增依赖。

## Risks

- 误判：要求同族里至少两个已知档位后缀，且只改档位、不改型号；无法确认时保持原行为。

---

# Implementation Plan

1. 创建本记录。
2. 新增 `EffortVariantBaseName`（型号 → 基础名 + 档位）。
3. `applyEffortVariant`：折叠映射优先，其次按同族兄弟档位确认，并移除正文里矛盾的档位字段。
4. 补测试：只有变体的目录下 effort 变 xhigh；单变体/无兄弟时不改。
5. `task verify`，提交并构建本地 Beta。

---

# Progress

- 2026-11-13 11:40：创建记录；用生产库快照确认真实目录里没有基础型号。

---

# Implementation Result

## Changes

- `internal/core/modelcatalog/effort_variants.go`：新增 `EffortVariantBaseName`，把型号拆成
  基础名与档位（`gemini-4-hydrogen-xhigh` → `gemini-4-hydrogen` / `xhigh`）。
- `internal/application/gateway/effort_variant.go`：折叠映射（`EffortVariantBase`）不可用时，
  用 `siblingEffortLevels` 统计同前缀下已确认的档位数量；≥2 即认定后缀是档位，写入
  `request.Effort` 并移除正文里矛盾的档位字段。只有一个后缀时不作判断。
- 测试：`TestVariantOnlyCatalogUsesModelLevel`（只有变体的真实目录形态，四种型号/档位组合）、
  `TestLoneVariantSuffixDoesNotOverrideClientLevel`（单变体不误判）。

## Files Changed

- `internal/core/modelcatalog/effort_variants.go`
- `internal/application/gateway/effort_variant.go`
- `internal/application/gateway/effort_variant_test.go`
- `docs/product/GOOGLE_SUBSCRIPTION.md`

## Technical Notes

- 上一轮修复失败的原因不是逻辑写错，而是**前置条件在真实数据里不存在**：折叠只在同名基础
  型号存在时发生，而 Google 这一代只发布变体。教训是这类判定必须拿生产目录验证，不能只测
  折叠后的假目录。
- 仍然只改档位、不改型号；`-xhigh` 的透传（TASK-20261113-001）保持不变。

---

# Verification

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
go test ./internal/application/gateway/ -count=1  → ok (52.2s)
go test ./internal/core/modelcatalog/ -count=1    → ok

隔离实例（生产数据副本、端口 3011、真实目录只有变体）：
POST /v1/chat/completions  model=gemini-4-hydrogen-xhigh  reasoning_effort=low  → 200
request_records 最新一行：input_model=gemini-4-hydrogen-xhigh  model=gemini-4-hydrogen-xhigh
                          reasoning_effort=xhigh  state=success
（同库更早的历史行仍是 low，与用户截图一致）

task --force verify → VERIFY_EXIT=0
```

---

# Documentation

- [x] `docs/product/GOOGLE_SUBSCRIPTION.md`：补充“只有变体、没有基础型号”时的判定方式
- [x] CURRENT

---

# Git

## Commit

见归档提交。未打标签、未推送。

---

# Result

`gemini-4-hydrogen-xhigh` 之类的变体型号在真实目录下也能以型号为准：记录与上游
`thinkingLevel` 都是 `xhigh`，不再被客户端同时带来的 `low` 覆盖。

---

# Completed

2026-11-13 11:33
