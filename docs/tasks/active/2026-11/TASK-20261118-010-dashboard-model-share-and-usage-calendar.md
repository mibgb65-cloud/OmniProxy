# TASK-20261118-010: 仪表盘模型 Token 占比与用量页年度热图

## Status

In Progress

## Created

2026-11-18 15:50

## Request

> 这个仪表盘页也可以显示一些更多的数据哇，例如什么今天用的token的模型的占比啥的，还有就是用量页面可以弄一个类似图二这种，你看看呢

图二为 GitHub 贡献图形态：按周分列、周一／周三／周五为行的整年小方格热图，右下角「少 → 多」色阶图例。

## Goal

- 仪表盘在所选时段（默认今日）内增加「模型 Token 占比」卡片：按已报告合计 Token 拆分模型，前若干名之外合并为「其他」，与指标、趋势、最近请求同一时段口径。
- 用量页增加近一年按本地日期的活跃度热图（GitHub 贡献图形态），支持 Token／请求数切换、悬浮查看单日数据、色阶图例与「少／多」说明。
- 两者只读现有 `usage_buckets` 累计表：无迁移、无新依赖、不改变现有记账与统计口径。

## Background

- `usage_buckets`（迁移 008）主键 `(day, subject, model)`，`day` 为请求开始时间的本地日期，含
  `requests`／`completed_usage`／`input_tokens`／`output_tokens`，明细淘汰后仍保留，可支撑按日、按模型聚合。
- 仪表盘现有 `DashboardInfo` 只有 15 分钟桶派生的小时／日期趋势、最近请求与累计提供商请求数；
  没有任何「按时段 × 模型」的 Token 维度（`dashboard_buckets` 无 model 列）。
- 用量页现有分析卡片（TASK-20261113-015）只覆盖近 30 天与终身累计，没有跨年的日历视图。

## Scope

- 后端 `internal/application/activity`：`DashboardInfo` 增加所选时段的模型 Token 汇总；
  `UsageInfo` 增加近 365 天按本地日期的用量序列。
- 前端仪表盘：新增「模型 Token 占比」卡片并与「客户端接入」同排，窄屏单列。
- 前端用量页：新增「年度活跃度」热图卡片（自绘 CSS grid，无图表库）。
- i18n 中英文案、单元测试、`task verify`、文档（PRD／DASHBOARD／ACTIVITY）。

## Out of Scope

- 新增数据库迁移、新增聚合表或改变记账事务。
- 按小时 × 模型的占比（`usage_buckets` 只有日粒度；小时粒度无模型维度）。
- 图表库、外部请求、缓存命中／成本维度进入新卡片。
- GitHub 式的连续天数、徽章或分享图导出。

## Existing Behavior

- 仪表盘：四项指标、趋势（结果分类／Token）、待处理问题、最近请求、资源概况、客户端接入。
- 用量页：6 项 KPI、每日用量（7／14／30 天）、Token 构成、模型／提供商／账号占比、模型每日堆叠、
  24 小时时段分布、请求质量、按提供商与模型明细。

## Expected Behavior

- 仪表盘新卡片标题「模型 Token 占比」，按所选时段列出前 6 个模型及其合计 Token 占比、请求数，
  余量合并为「其他」；没有已报告 Token 时显示空状态而不是 0%。
- 点击今日／近 7 天／近 30 天，卡片与其它模块同步变化；切换时不显示上一时段数字。
- 用量页热图覆盖近 365 个本地日期，周一为首列起点；未来日期留空不可见；每格悬浮显示日期、
  输入／输出／合计 Token 与请求数；色阶按非零日中位数分四档，全零时整图中性色。
- 热图提供 Token／请求数切换，图例与悬浮口径跟随切换；读不到数据时显示已有空状态文案。

---

# Analysis

## Related Modules

- `internal/application/activity/dashboard.go`、`dashboard_types.go`（新增 `Models` 字段与只读查询）
- `internal/application/activity/usage.go`、`types.go`（新增 `Calendar` 字段与只读查询）
- `internal/transport/wails/dashboard.go`、`activity.go`（绑定不变，仅结构体字段）
- `frontend/src/features/dashboard/*`（新卡片、视图排布、`dashboard-clients` 窄列适配）
- `frontend/src/features/activity/*`（新热图组件、`usage-model.ts` 网格与色阶计算）
- `frontend/src/i18n/locales/dashboard-*.ts`、`activity-*.ts`
- 生成物 `frontend/bindings/**`（由 `task build` 自动重新生成）

## Dependencies

N/A（复用 `database/sql`、React、现有 Tooltip 与卡片组件）

## Risks

- 时段到本地日期的换算必须与前端边界一致：边界由前端按本地日历生成，后端只把首／末边界折回
  本地 `time.DateOnly`；若用 UTC 日期会漏掉/多算本地凌晨的用量。测试需固定本地午夜边界。
- 热图 365 格若每格都设 `tabIndex`，键盘用户要按 365 次 Tab；不做逐格焦点，保留 `aria-label`
  与悬浮提示。
- 年度查询随用量页 5 秒轮询重复执行；按 `day` 过滤后只扫一年，行数受「日期 × 模型数」约束。

---

# Implementation Plan

1. 后端：`DashboardInfo.Models`（前 8 名，`LIMIT` 常量）+ `readDashboardModels` 于同一只读事务；
   `UsageInfo.Calendar`（近 365 天按日）+ `calendarDays`。
2. Go 测试：模型占比随时段收窄、日历覆盖一年且排除窗口外。
3. 重新生成绑定；前端服务类型透传新字段。
4. 前端：`dashboard-model.ts` 增加占比计算（纯函数 + 单测）；`dashboard-model-share.tsx` 卡片；
   `dashboard-view.tsx` 排布调整；`dashboard-clients.tsx` 适配窄列竖排。
5. 前端：`usage-model.ts` 增加日历网格／色阶／月份标签纯函数 + 单测；
   `usage-calendar.tsx` 组件 + 单测；插入用量页。
6. i18n：zh／en 同步新增键。
7. `task verify`；更新 PRD／DASHBOARD／ACTIVITY；记录 Implementation Result 与 Verification；提交。

---

# Progress

- 2026-11-18：确认现状（`usage_buckets` 为唯一含 model 的日粒度累计表，`dashboard_buckets` 无 model 列），
  确定新卡片随仪表盘时段、热图复用现有累计表，无迁移。

---

# Implementation Result

## Changes

实际完成：

- `DashboardInfo.Models`：所选时段各模型已报告的输入／输出／合计 Token 与请求数，在首页的只读事务内
  与指标、趋势一起读取；时段边界按本地日期折算（末边界为开区间，回退一秒取最后一个被包含的日期）。
- 首页新增「模型 Token 占比」卡片，与「客户端接入」同排（2:1），窄屏单列；卡片按同一批数据取前六名，
  其余模型合并成精确的「其他 N 个模型」（余量 Token 与请求数都由剩余行相加，不依赖指标卡）。
- `UsageInfo.Calendar`：近 365 个本地日期按日的输入／输出／合计 Token 与请求数；没有记录的日期不返回，
  由前端补零。
- 用量页新增「年度活跃度」热图：53 列 × 7 行、周一起始，支持 Token／请求数切换、单日悬浮明细、
  四分位四档色阶与「少／多」图例；今天之后的格子占位不显示，也不参与色阶。
- 抽出共用色序 `CHART_SERIES_COLORS` 与 `chartPercentLabel` 到 `lib/chart.ts`，`usage-model.ts` 改为复用，
  避免首页与用量页各写一份配色／百分比格式。
- 首页「客户端接入」改为卡片内竖排（标题 + 入口、状态胶囊换行），适配 1fr 窄列。

## Files Changed

- `internal/application/activity/dashboard.go`、`dashboard_types.go`
- `internal/application/activity/usage.go`、`types.go`
- `internal/application/activity/dashboard_test.go`、`usage_insights_test.go`
- `frontend/src/features/dashboard/dashboard-model.ts`、`dashboard-model-share.tsx`、`dashboard-view.tsx`、
  `dashboard-clients.tsx`、`dashboard-model.test.ts`、`dashboard-model-share.test.tsx`、
  `dashboard-test-helpers.tsx`、`dashboard-view.test.tsx`
- `frontend/src/features/activity/usage-calendar.tsx`、`usage-calendar-model.ts`、`usage-view.tsx`、
  `usage-model.ts`、`usage-calendar.test.tsx`、`usage-calendar-model.test.ts`、`activity.test.tsx`
- `frontend/src/lib/chart.ts`、`frontend/src/services/activity.ts`、`frontend/src/services/dashboard.ts`
- `frontend/bindings/**`（由 `wails3 generate bindings` 重新生成）
- i18n：`activity-zh/en`、`dashboard-zh/en`

## Technical Notes

- `usage_buckets` 只有「本地日期 × 身份 × 模型」粒度，`dashboard_buckets` 没有模型列，因此模型占比只能按
  本地日期折算时段；前端生成的边界是本地午夜的 UTC 秒，若直接截 UTC 日期会错开一天，`dashboardModelDayRange`
  统一按 `time.Local` 折算。
- 年度热图用 `aspect-ratio: 53 / 7` 的网格容器给每格定尺寸，行高由容器高度均分，与左侧星期标签列同高；
  不依赖 `aspect-square` 在 auto 行里的尺寸推导，避免窄窗口下格子变成矩形。
- 热图整体作为 `role="img"` + 摘要 `aria-label`，365 个格子不设逐格焦点（避免键盘用户按 365 次 Tab），
  悬浮提示仍可查看单日数据。

---

# Verification

## Tests

- [x] Unit Tests（Go：`TestDashboardModelShareFollowsSelectedPeriod`、`TestUsageCalendarCoversLastYear`；
      前端：`dashboard-model.test.ts`、`dashboard-model-share.test.tsx`、`usage-calendar-model.test.ts`、
      `usage-calendar.test.tsx`）
- [ ] Integration Tests（不适用：只读现有累计表，无新接口）
- [ ] E2E Tests（不适用：桌面应用无 E2E 框架）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify
EXIT=0
```

- `format:check`：gofmt 无待格式化文件，Prettier 全部通过。
- `lint`：`go vet` / `staticcheck` 通过；ESLint 0 error（2 个既有 warning 来自
  `features/accounts/reset-credit-list.tsx`，与本任务无关）。
- `typecheck`：`tsc --noEmit` 通过（含重新生成的绑定）。
- `test`：全部 Go 包通过；前端 162 文件／990 项测试通过（本任务新增 13 项）。
- `build`：`bin/omniproxy.exe` 生产构建成功（Windows amd64，`-tags production`）。

未做：桌面应用内的实机目视检查（热图与卡片在真实窗口下的排布）。布局按固定网格与
`aspect-ratio` 推导，已由单元测试锁定结构与合计口径，未在 GUI 中截图确认。

---

# Documentation

- [x] PRD（仪表盘与用量页的能力描述）
- [x] Architecture（无变更：只新增两个只读聚合字段，记账事务、表结构与模块边界不变）
- [x] ADR（无：不改变架构决策，复用既有累计表与只读事务）
- [x] CURRENT

同时更新 `docs/product/DASHBOARD.md`（模型 Token 占比口径）与 `docs/product/ACTIVITY.md`
（年度活跃度热图与近一年查询）。

---

# Git

## Commit

（未开始）

---

# Result

（未开始）

---

# Completed

（未完成）
