# TASK-20261113-015: 用量页增加图表与分析维度

## Status

In Progress

## Created

2026-11-13 13:20（Asia/Singapore）

## Request

> 这个用量页面的图表太少了，可以增加引入更多的图标，更多的分析维度，感觉体验会更好。

（“图标”按上下文理解为“图表”。）

## Goal

- 用量页从「累计数字 + 一张每日柱图 + 两张清单」扩展为完整的使用分析页：
  环比、构成、模型／提供商占比、模型每日结构、时段分布、请求质量、账号排行。
- 所有图表只使用已保存的本地统计或新增的只读聚合，不新增数据库迁移、不引入图表库。
- 保持现有口径：Token 十进制字符串、BigInt 精度、未知与真实零区分、5 秒自动刷新、无手动刷新按钮。

## Background

- 当前用量页（`frontend/src/features/activity/usage-view.tsx`）只有 3 个 KPI、
  近 30 天输入／输出柱图、提供商×模型表、账号与模型两张清单。
- 后端 `UsageInfo` 已提供累计、近 30 天按日、账号、模型、提供商×模型数据；
  另有未淘汰的 `dashboard_buckets`（15 分钟桶）可用于时段与成功率，
  `usage_accounts` 可用于缓存命中下界。
- 项目没有图表依赖，现有图表均为 CSS／div 自绘；继续沿用，不新增依赖。

## Scope

- 后端 `UsageInfo` 增加只读聚合：前 30 天对比、提供商汇总、Top 模型每日用量、
  本地时段分布、缓存命中下界、近 30 天请求质量。
- 前端新增可复用图表原语（环形图、占比条、紧凑坐标轴）与 6 个分析卡片；
  重做每日趋势（7／14／30 天、Token／请求数切换）。
- 中英文案、组件测试、Go 聚合测试、`docs/product/ACTIVITY.md` 与 CURRENT 更新。

## Out of Scope

- 不新增数据库迁移或后台任务；不改变记账口径。
- 不引入 recharts／echarts 等图表库。
- 不改仪表盘、请求页、配额页的布局；不改账号／模型的历史数据。
- 不做成本、导出、自定义时间范围（数据只有近 30 天日桶）。

## Existing Behavior

用量页展示累计输入／输出／合计、累计请求、完整用量覆盖、近 30 天柱图、
「按提供商与模型」表，以及账号成员／模型两张清单。没有占比、趋势结构、
时段、质量或环比视图。

## Expected Behavior

- 顶部 6 个 KPI：累计 Token、累计请求、近 30 天 Token（含环比）、平均每请求、
  缓存命中率（下界明确标注）、近 30 天成功率。
- 每日趋势可在 7／14／30 天和 Token／请求数之间切换，带纵轴刻度与提示。
- Token 构成环形图、模型占比、提供商占比三张结构卡片。
- Top 6 模型每日堆叠图，其余合并为「其他」。
- 24 小时时段分布与近 30 天请求质量（成功率、平均耗时、平均首字）。
- 账号成员排行（输入／输出拆分条 + 占比）。
- 保留提供商×模型明细表；空数据与未知口径继续显示 `—`／空状态。

---

# Analysis

## Related Modules

- `internal/application/activity/usage.go`、`types.go`、`provider_models.go`
- `internal/application/activity/dashboard.go`（复用桶结构与成功率算法）
- `frontend/src/features/activity/usage-*.tsx`、`provider-model-usage.tsx`
- `frontend/src/i18n/locales/activity-{zh,en}.ts`
- `frontend/bindings/omniproxy/internal/application/activity/models.ts`（生成）
- `docs/product/ACTIVITY.md`

## Dependencies

- 复用 `lucide-react`、现有 Card／Tooltip／Button 与 Tailwind 语义色。
- 复用 `dashboard_buckets` 的 15 分钟聚合（迁移 010，未随保留上限淘汰）。
- 复用 `usage_accounts` 的缓存已知计数（迁移 045）。
- 无新依赖、无迁移。

## Risks

- 图表变多后信息密度过高：用分区（概览／结构／趋势／时段／明细）控制层级，
  小屏纵向堆叠，占比列表限高滚动。
- `dashboard_buckets` 在迁移 010 之前的历史缺失：时段与质量卡片只承诺近 30 天，
  文案不把它当作终身累计。
- Top 模型每日查询使用动态占位符：模型名只作为参数，不拼接 SQL。

---

# Implementation Plan

1. 后端：为 `UsageInfo` 增加 `previous`、`providers`、`modelDays`、`hours`、
   `cache`、`reliability`，全部只读查询；补齐错误处理与单元测试。
2. 前端：抽出 `src/lib/chart.ts`（坐标轴、百分比、时长、紧凑格式），
   从 `dashboard-model.ts` 复用同一实现，不改变仪表盘行为。
3. 前端：新增 `usage-model.ts` 纯函数（本地日期序列、环比、Top 模型堆叠、求和），
   先写测试再接到组件。
4. 前端：新增 `usage-kpis`、`usage-daily`、`usage-composition`、`usage-share`、
   `usage-model-trend`、`usage-hours`、`usage-quality` 组件与共享原语，
   重排 `usage-view.tsx`，删除被替代的旧柱图／清单。
5. i18n：新增中英文案；未知值、下界、空状态都要有明确措辞。
6. 生成 bindings，运行 `task verify`，用临时预览页做桌面宽度视觉核对后删除预览文件。
7. 更新 `docs/product/ACTIVITY.md`、`docs/plans/CURRENT.md` 与 Task Record，
   提交并记录 commit。

---

# Progress

- 2026-11-13 13:20：创建记录，完成现状与数据源分析。

---

# Implementation Result

用量页从「3 个 KPI + 一张每日柱图 + 两张清单」扩展为 6 个 KPI 与 7 个分析区块，
全部由现有累计表与桶聚合只读得出，无迁移、无新依赖。

## Changes

- 后端 `UsageInfo` 增加 `previous`／`providers`／`modelDays`／`hours`／`reliability`／`cache`：
  - `previous`：最近 30 天之前的 30 天窗口合计，用于环比；上一窗口为 0 时不计算百分比。
  - `providers`：`usage_provider_models` 按 `provider_id` 聚合，名称优先取当前 `providers` 表，
    历史归属仍以 `legacy:` 前缀保留。
  - `modelDays`：先按窗口内合计选出前 6 个模型，再取这些模型逐日合计；模型名只作为参数。
  - `hours`／`reliability`：读 `dashboard_buckets`（迁移 010，15 分钟桶）按本地时区归入 24 小时，
    质量指标复用 `dashboardCounts.summary()`（成功率只计已结束成功／失败，耗时与首字只取成功样本）。
  - `cache`：汇总 `usage_accounts` 的 `cached_input_tokens`／`cached_known`，供前端判定下界。
- 前端抽出 `src/lib/chart.ts`（坐标轴上限／刻度／标签抽稀／百分比／紧凑数／时长），
  仪表盘 `dashboard-model.ts` 改为复用同一实现并保留原有导出名，行为不变。
- 新增 `usage-model.ts` 纯函数（本地日期序列与补零、BigInt 求和、环比、占比、Top 模型堆叠、峰值时段）。
- 新组件：`usage-kpis`（6 张 KPI 卡）、`usage-daily`（7／14／30 天 × Token／请求数）、
  `usage-composition`（环形构成 + 每请求输入／输出 + 输入输出比）、`usage-share`
  （模型／提供商／账号三张占比卡）、`usage-model-trend`（Top 6 每日堆叠）、
  `usage-hours`（24 小时分布）、`usage-quality`（成功率／平均耗时／平均首字）；
  `usage-charts.tsx` 提供卡片、空态、图例、环形图与占比条原语。
- `usage-view.tsx` 重排为「概览 → 每日趋势 → 结构占比 → 模型每日 → 时段／质量 → 账号排行 →
  提供商×模型明细」，删除被替代的 `usage-breakdown.tsx`。
- i18n 中英文案补齐；空数据与未知值继续显示 `—`／空状态，缓存下界标「≥」。
- `wails3 generate bindings -ts -i` 重新生成 TypeScript 绑定（仅新增接口／字段）。

## Files Changed

- 后端：`internal/application/activity/usage.go`、`types.go`、新增 `usage_insights_test.go`
- 前端新增：`src/lib/chart.ts`、`src/features/activity/usage-{model,charts,kpis,daily,composition,share,model-trend,hours,quality}.tsx|ts`、
  `usage-model.test.ts`、`usage-insights.test.tsx`
- 前端修改：`usage-view.tsx`、`activity.test.tsx`、`services/activity.ts`、`dashboard-model.ts`、
  `i18n/locales/activity-{zh,en}.ts`、`bindings/.../activity/{models,index}.ts`
- 删除：`frontend/src/features/activity/usage-breakdown.tsx`
- 文档：`docs/product/ACTIVITY.md`、`docs/plans/CURRENT.md`、本记录

## Technical Notes

- 所有 Token 仍以十进制字符串返回，前端用 BigInt 求和与比较；百分比用整数运算保留两位，
  避免累计值超过 `Number.MAX_SAFE_INTEGER` 后失真。
- 图表全部用 div／SVG 手绘，未引入 recharts／echarts。
- 时段与质量来自 `dashboard_buckets`：迁移 010 之前的历史缺失，因此文案只承诺近 30 天，
  不代表终身累计；账号排行与占比仍读不淘汰的 `usage_buckets`。
- 缓存命中率的分母是全部累计输入，分子只含已确认缓存；`knownRequests < requests` 时显示
  `≥` 下界，没有任何缓存报告时显示 `—`，不把未知当 0。
- 模型占比与 Top 6 趋势使用同一「前 6 + 其他」口径，避免两处排名不一致。

---

# Verification

## Tests

- [x] Unit Tests
- [ ] Integration Tests
- [ ] E2E Tests

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
go test ./internal/application/activity
ok  omniproxy/internal/application/activity

task verify
format check → lint → typecheck → test → build 全部通过，EXIT=0
Test Files  151 passed (151)
     Tests  923 passed (923)
```

- 新增 Go 测试 3 项（聚合维度、Top 6 模型、空 profile），新增前端测试 19 项
  （`usage-model.test.ts` 8 项、`usage-insights.test.tsx` 11 项）。
- 临时 Vite 预览页在 1440×1000 下按浅色／深色逐屏截图核对：KPI、每日趋势、
  构成／占比、Top 6 堆叠、时段与质量、账号排行、提供商×模型表均正常；
  预览文件已在提交前删除，不进入提交。

---

# Git

- Commit: TBD
