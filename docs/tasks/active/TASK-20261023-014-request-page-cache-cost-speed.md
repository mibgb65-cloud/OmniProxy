# TASK-20261023-014: 请求页增加缓存命中、成本、速度与首字字段

## Status

In Progress（实现与质量门通过，原生窗口视觉确认待人工复核）

## Created

2026-10-23 00:00（Asia/Singapore）

## Request

用户给出参考截图（表头「速度 ?」，行值为 `≈23 tok/s`）并说明：

> 我想在请求页面增加缓存命中字段，成本，速度 tok/s，首字这几个数据，帮我实现

## Goal

在请求列表（`/requests`）中以列的形式展示每条请求的缓存命中、估算成本、生成速度和首字耗时；详情弹窗同步可读，缺失上游明细时保持未知而不是伪造零。

## Background

- 迁移 037 已为 `request_records` 增加可空 `cached_input_tokens`，`activity.Update` 只在输入已知时写入，`core/proxy` 已按 Chat／Responses／Messages 三种协议解析缓存明细。
- 迁移 032 已保存 `first_token_ms`，详情弹窗已有「首字耗时」，列表页没有。
- `internal/core/tokenprice` 提供基础参考价与 `Price.Cost`（缓存未知时返回区间），目前只被配额估算使用，请求明细未暴露成本。
- 列表页当前列：时间、模型、Agent、账号、结果、输入、输出、耗时。

## Scope

- 后端 `RequestItem` 暴露缓存明细（数量 + 已知标志）与按参考价估算的成本区间（含已知标志）。
- 请求列表新增「缓存命中」「成本」「速度」「首字」四列，表头带说明提示。
- 请求详情弹窗补齐缓存、成本、速度三项。
- 中英文文案与对应测试、产品文档。

## Out of Scope

- 不新增迁移：所需数据列已存在。
- 不改网关记账、不改价格表取值、不引入官方账单金额。
- 不给仪表盘、账号本机用量、日志页新增这些字段。
- 不按模型名或前缀猜测价格，不为未知型号回退某个 GPT 价格。

## Existing Behavior

请求列表只显示输入／输出 Token 与总耗时；缓存与成本只能通过配额估算间接感知；速度无法在界面上查看。

## Expected Behavior

- 缓存命中：显示缓存输入占输入 Token 的百分比，提示框给出「缓存 / 输入」原始数量；上游未报告时显示 `—`。
- 成本：按基础参考价估算的美元金额；缓存未知时显示区间；用量不完整或型号无价格时显示 `—`。
- 速度：输出 Token ÷ 生成耗时（流式扣除首字等待），形如 `≈23 tok/s`；输出未知或进行中显示 `—`。
- 首字：首字毫秒；未采集（非流式／历史记录）显示 `—`。

---

# Analysis

## Related Modules

- `internal/application/activity`：`types.go`（`RequestItem`）、`queries.go`（列与扫描）、新增成本计算文件。
- `internal/core/tokenprice`：只调用既有 `Lookup`／`Cost`，不修改价格。
- `frontend/src/features/activity`：`request-table.tsx`、`request-dialog.tsx`、`activity-format.ts`。
- `frontend/src/i18n/locales/activity-zh.ts`、`activity-en.ts`。
- `frontend/bindings/omniproxy/internal/application/activity/models.ts`（生成）。

## Dependencies

- 既有迁移 032／037 的列，无新依赖。

## Risks

- 成本为估算而非账单，需在提示文案中明确，避免被当作实际扣费。
- 速度依赖已报告的输出 Token 与耗时，失败／取消请求可能给出误导值：只在已结束且有输出 Token 时计算。
- 列表列数从 8 增至 12，窄窗口依赖既有横向滚动容器；需确认表头提示不破坏可访问名称。

---

# Implementation Plan

1. `RequestItem` 增加 `cachedInputTokens`／`cachedInputKnown`、`costMin`／`costMax`／`costKnown`。
2. `requestColumns` 追加 `r.cached_input_tokens`，`scanRequest` 读取 `sql.NullInt64` 并调用成本计算。
3. 新增 `request_cost.go`：仅在用量完整且型号有参考价时给值，缓存未知时保留区间。
4. 前端格式化函数：`formatCachedHit`、`formatCost`、`formatSpeed`、`formatFirstToken`。
5. 请求列表新增四列（含帮助提示），详情弹窗补齐缓存／成本／速度。
6. 中英文文案；Go 与前端测试；`wails3 generate bindings` 更新提交的绑定。
7. `task verify`。

---

# Progress

- 已确认数据基础（迁移 032／037）与价格来源，无需新迁移。
- 后端暴露缓存明细与成本区间后，前端列表／详情接入并补齐中英文文案与表头说明。
- 新增测试先覆盖格式化边界（未报告 vs 真实零、区间成本、进行中、首字未知），再补列表渲染断言。

---

# Implementation Result

## Changes

- 后端 `RequestItem` 新增 `cachedInputTokens`／`cachedInputKnown`、`costMin`／`costMax`／`costKnown`；列表、详情与仪表盘最近记录共用同一扫描，因此三处一致返回。
- `requestColumns` 追加 `r.cached_input_tokens`，`scanRequest` 以 `sql.NullInt64` 读取：未报告保持空串 + `known=false`，真实零命中为 `"0"` + `known=true`。
- 新增 `request_cost.go`：只在用量完整且型号有参考价时给值；缓存未知时用 `tokenprice.Price.Cost` 的上下界，未知型号与不完整用量保持未知（不前移为 0）。
- 前端 `activity-format.ts` 新增 `formatCachedHit`／`formatCost`／`formatSpeed`／`formatFirstToken`；Token 字符串经 `BigInt` 转换后再计算，避免安全整数边界问题。
- 请求列表新增「缓存命中」「成本」「速度」「首字耗时」四列（表头带圆形说明按钮和提示），缓存列悬停显示「缓存 / 输入」原始数量；详情弹窗按同样顺序补齐这四项。
- 中英文文案（含四条列说明与缓存明细）同步；`wails3 generate bindings` 只更新了 activity 的 `models.ts`。

## Files Changed

- `internal/application/activity/types.go`、`queries.go`、`request_cost.go`（新增）
- `internal/application/activity/request_cost_test.go`（新增）
- `frontend/src/features/activity/activity-format.ts`、`request-table.tsx`、`request-dialog.tsx`
- `frontend/src/features/activity/request-metrics.test.tsx`（新增）、`activity.test.tsx`、`request-agent.test.tsx`
- `frontend/src/features/dashboard/dashboard-test-helpers.tsx`、`frontend/src/features/gateway/gateway-overview-test-data.ts`（补齐固定夹具的新增字段）
- `frontend/src/i18n/locales/activity-zh.ts`、`activity-en.ts`
- `frontend/bindings/omniproxy/internal/application/activity/models.ts`（生成）
- `docs/product/ACTIVITY.md`、`docs/plans/CURRENT.md`

## Technical Notes

- 成本为 `internal/core/tokenprice` 的基础参考价估算，不是账单；缓存未知时 `min != max`，前端显示区间。
- 速度 = 输出 Token ÷ 生成耗时；`firstTokenMs` 有效且小于总耗时时扣除首字等待，结果按四舍五入加 `≈` 前缀。
- 未新增迁移；历史记录没有缓存列时保持未知，不补零。

---

# Verification

## Tests

- [x] Unit Tests（`go test ./internal/application/activity/ -count=1`；`request-metrics.test.tsx`）
- [x] Integration Tests（既有 activity 事务／用量回归在同包测试内）
- [ ] E2E Tests（N/A：项目尚无桌面 E2E）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
go test ./internal/application/activity/ -count=1
ok  	omniproxy/internal/application/activity	10.751s

task verify
exit=0
format:check → lint → typecheck → test → build 全部通过
前端 125 个测试文件 / 740 项测试通过（新增 6 项）
Go 全部包 ok；Windows production 构建 bin/omniproxy.exe 成功
```

原生窗口视觉验收（`task dev` 中核对 12 列宽度、表头提示与深浅主题）尚未由 Agent 执行，等待人工确认。

---

# Documentation

- [ ] PRD
- [ ] Architecture
- [ ] Tech Stack
- [ ] Coding Rules
- [ ] Testing
- [ ] Build
- [ ] ADR
- [x] CURRENT

`docs/product/ACTIVITY.md` 的「页面关联」补充四列的来源、未知语义与「不参与聚合」边界；`docs/plans/CURRENT.md` 增加 In Progress 条目。PRD／架构／技术栈不变。

---

# Deviations

原计划未规定列顺序与表头提示形式。实际按用户列出的顺序（缓存命中 → 成本 → 速度 → 首字 → 耗时）排列，并给这四列各加一个可聚焦的说明按钮（参考截图中表头的问号），说明文案写入 i18n。

---

# Git

## Commit

```text
待填写
```

---

# Result

请求列表和详情现在会显示缓存命中率、参考价估算成本、`≈N tok/s` 生成速度和首字耗时；未知上游明细统一显示 `—`，不再需要到配额估算里间接推断。完整质量门通过，原生窗口视觉确认待人工复核。

---

# Completed

Pending.
