# TASK-20261118-008: 家族名＋档位在选服务商前解析到官方变体

## Status

In Progress

## Created

2026-11-18 14:32

## Request

用户在看到 TASK-20261118-007 的诊断后要求：

> 不要让 workbuddy 去截获这个谷歌的请求，或者你设置不同服务之间的切换

选择“不要截获”这条：`gemini-4-carbon` 这类“家族名＋档位”的请求应当由发布该档位变体的官方订阅（Google／Antigravity）承接，而不是被只发布裸型号的 WorkBuddy 中转站截获。跨服务商自动切换（同一次请求里换服务商）本次不做。

## Goal

客户端按家族名 + 明确档位发送请求时，路由**先**解析到该档位在目录里真实存在的变体型号，再按变体选择服务商与账号：

- `gemini-4-carbon` + `reasoning_effort=high` → `gemini-4-carbon-high` → Google 订阅账号；
- 没有明确档位时不猜，保持精确匹配（今天的行为）；
- 家族只有一个档位变体时不改写（后缀可能只是型号的一部分）。

## Background

请求记录显示 2026-11-18 13:48 的三条 `gemini-4-carbon` 全部打到 WorkBuddy AI（`mibgb65@gmail.com`），而用户的 Google 订阅账号只发布 `gemini-4-carbon-high`／`-low`／`-medium`，从未收到过任何 carbon 请求。

## Scope

- `RouteRequest` 在 `resolvePreview` 之前增加一步“家族名＋明确档位 → 变体型号”的改写。
- 只对本机目录里确实存在同族 ≥2 个档位变体、且客户端明确给了档位的请求生效。
- 复用已有 `familyVariants`／`siblingEffortLevels` 判定，硬性要求档位后缀精确匹配（不做就近替换）。
- 测试：碳/氩家族名 + 档位走官方变体；无档位保持中转站；单变体家族不改写；精确变体型号不被改写。

## Out of Scope

- 跨服务商自动切换（同一次请求失败后换到另一个服务商）：候选池按 ADR-015 冻结在服务商内，改动面大，另行评估。
- 目录发布策略：`gemini-4-carbon`（WorkBuddy 裸型号）仍作为一个独立模型发布，客户端仍可看到它（不带档位时会走到它）。
- 历史请求记录回填。

## Existing Behavior

- 路由按“精确模型 ID”筛服务商（`evaluate` → `targetSupports`），裸 `gemini-4-carbon` 只有 WorkBuddy 发布 → 候选只有 WorkBuddy 两个账号。
- 家族名回退（`applyFamilyVariant`）只在“精确匹配失败”后触发，因此精确命中中转站裸型号时永远不会走官方变体。
- 请求期的档位变体改写（`applyEffortVariant`）发生在服务商与账号已选定之后，只能改型号，改不了服务商。

## Expected Behavior

- 带明确档位的 Gemini 3.8／4 家族请求命中官方订阅的档位变体，WorkBuddy 不再承接。
- 不带档位的同一家族名请求仍按精确匹配走原服务商。
- 请求明细记录的模型与档位与上游实际收到的一致（变体型号 + 档位字段被剥离）。

---

# Analysis

## Root Cause

档位变体的解析顺序错了：`RouteRequest` 先按裸名称选服务商，再在最后把档位换成变体（只改型号）。只要**任何**服务商发布了裸名称（这里是 WorkBuddy 中转站），官方订阅的变体就永远选不中。

本机目录里“裸型号 + 另一服务商的档位变体”只有三组，全部是 Gemini：`gemini-3.8-flash`、`gemini-4-argon`、`gemini-4-carbon`（其余同前缀型号是 `-sg`／`-flash`／`-free` 这类不同产品，不是档位）。因此影响面被限制在 Gemini 家族，符合用户意图。

## Related Modules

- `internal/application/gateway/routing.go`（RouteRequest）
- `internal/application/gateway/family_variant.go`
- `internal/core/modelcatalog`（EffortVariant 判定，只读）

## Dependencies

N/A

## Risks

- 官方订阅池全部冷却时，这类请求现在会返回 429，而不是静默走中转站。这是用户明确要的取舍：不再用中转站额度顶替自家订阅。
- 若某家族只有一个档位变体，或客户端没给档位，行为完全不变。

---

# Implementation Plan

1. `applyRequestedFamilyVariant`：在 `resolvePreview` 之前，用正文里的明确档位 + 目录里同族 ≥2 档位变体判定，把型号改写成 `模型-档位`。
2. `RouteRequest` 接入该步骤。
3. 测试：新增 4 个断言场景。
4. ADR-022 增补“解析时机”，`docs/product/FAILOVER.md` 或路由文档补充口径。
5. `task verify`。

---

# Progress

- 2026-11-18 14:32 确认影响面：只有 Gemini 3.8／4 三个家族存在“裸型号在 A、档位变体在 B”的冲突。
- 2026-11-18 14:38 实现并接入 `RouteRequest`，新增 5 项断言；既有家族回退测试不变。
- 2026-11-18 14:41 `task verify` EXIT=0。

---

# Implementation Result

## Changes

1. `applyRequestedFamilyVariant`：在 `resolvePreview` 之前把“家族名 + 明确档位”改写成真实变体型号。触发条件全部命中才改写：
   - 正文里存在明确档位（Chat `reasoning_effort`／Responses `reasoning.effort`／Messages `output_config.effort`）；
   - 同族已确认至少两个档位后缀（沿用 `siblingEffortLevels` 判定）；
   - 该档位的变体在目录里**精确存在**（`exactFamilyVariant`，不做就近替换）。
2. `RouteRequest` 在选服务商前调用该步骤；改写后路由看到的就是变体型号，发布裸型号的中转站不再参与。
3. 改写后沿既有链路剥离正文里的档位字段并把档位写进请求明细（`applyEffortVariant`），记录与上游实际收到的一致。

效果（本机目录）：`gemini-4-carbon`／`gemini-4-argon`／`gemini-3.8-flash` + 明确档位 → `…-high`／`-low`／`-medium` → Google 订阅账号；不带档位的同名请求仍按精确匹配走 WorkBuddy。

## Files Changed

- `internal/application/gateway/family_variant.go`
- `internal/application/gateway/routing.go`
- `internal/application/gateway/family_variant_test.go`
- `docs/decisions/ADR-022-effort-vocabulary-and-variants.md`
- `docs/product/ROUTING.md`
- `docs/plans/CURRENT.md`

## Technical Notes

- 只改路由入口，不改目录发布：WorkBuddy 的裸型号仍在客户端模型列表里，未指定档位时可用。
- 判定沿用 `familyVariants`（同提供商前缀）与 `siblingEffortLevels`（同前缀至少两个档位），避免把 `-sg`／`-flash`／`-free` 这类不同产品当成档位；本机目录里满足条件的只有 Gemini 3.8／4 三组。
- 官方订阅池整体冷却时这类请求返回 429（既有池冷却语义），不再回落到中转站：用哪家额度由用户明确指定，不由目录顺序决定。

---

# Verification

## Tests

- [x] Unit Tests：家族名 + 四档位矩阵（explicit／medium／无档位／无对应档位）、单变体家族不改写
- [x] Integration Tests：`RouteRequest` 端到端（路由首次解析即看到变体，正文档位已剥离）
- [ ] E2E Tests：N/A（需要真实 Antigravity 桌面端与 Google 账号）

## Quality Checks

- [x] Format／Lint／Type Check／Test／Build／Verify

## Verification Result

```text
task verify
EXIT=0
Go：internal/application/gateway 53s、routing 11s、modelcatalog 全绿
Frontend：159 files / 977 tests passed（本任务未改前端）
wails3 production build + vite build 通过
```

---

# Documentation

- [x] ADR：ADR-022 增补「变体解析必须在选服务商之前」
- [x] CURRENT
- [x] 产品文档：`docs/product/ROUTING.md`
- [ ] 其它：No documentation changes required.

---

# Deviations

- 计划里只写“家族名 + 档位”，实现时补了“同名请求在目录不可用时完全不改写”的兜底（读取目录失败直接返回原请求，让后续路由给出明确错误码）。
- 未实现跨服务商自动切换（原需求里的另一种选择）：候选池按 ADR-015 冻结在服务商内，属于另一项改动。

---

# Git

## Commit

```text
Planned
```

---

# Result

Gemini 家族的“家族名 + 档位”请求改由官方订阅承接，WorkBuddy 只在客户端没有指定档位时才会被选中。

