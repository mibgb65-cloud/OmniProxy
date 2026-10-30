# TASK-20261030-003: Dashboard 右侧卡片自适应填充

## Status

In Progress

## Created

2026-10-30 19:28

## Request

> 这个页面在放大时的显示的卡片感觉有点空，右侧的两个卡片，你看看能不能处理一下呢？给我一个优化方案呢

随后用户确认：「你看着改吧」。

## Goal

仪表盘在超宽窗口（≥1920 CSS px，实测 2560）下不再出现「右列两张卡片下方的大块空洞」，
两张右卡的内容密度与卡片尺寸匹配，左列图表随宽度长高而不是被拉成宽而扁。

## Background

窗口 2560px 时内容区约 2170px，右列 `(2170-20)/3 ≈ 716px`：

| 列 | 第一行 | 第二行 |
| --- | --- | --- |
| 左 (2fr) | 调用趋势 ~277px | 最近请求 ~370px |
| 右 (1fr) | 待处理问题 ~72px | 资源概况 ~155px |

用户截图（2560×1392）中右列两处垂直留白 ~200px / ~215px。

## Scope

- `DashboardView` 两行网格的垂直对齐与超宽比例。
- `DashboardIssues`：内容密度与底部槽位。
- `DashboardResources`：资源健康分布、按供应商状态、底部槽位。
- `DashboardTrend`：绘图高度随断点增长、柱宽上限。
- 配套 i18n（zh/en）、纯函数与回归测试。

## Out of Scope

- 不改任何后端接口、Wails 绑定、数据库含义。
- 不改统计口径（成功率、Token、耗时、覆盖率）与既有空态/失败态文案语义。
- 不引入新依赖、不改主题 token、不改其它页面布局。
- 不为首页增加任何自动官方额度请求（`enabled: false` 的共享缓存语义保持不变）。

## Existing Behavior

- `dashboard-view.tsx` 两个网格均为 `items-start`，右列卡片高度只由自身内容决定。
- `dashboard-issues.tsx` 空态是单个虚线框 + 一行文案；有 issue 时最多 5 条。
- `dashboard-resources.tsx` 内容是 3 个大数字 + 2 行 12px 灰字 + 1 个链接。
- `dashboard-trend.tsx` 绘图区固定 `h-52`，柱宽上限 22px。

## Expected Behavior

- ≥1280px：左右两列每一行的卡片底边对齐，右列不再出现行间空洞。
- 1280px 以上趋势绘图区 14.5rem，1680px 以上 16.5rem；柱宽上限 28px。
- 资源概况展示健康分布条 + 图例、按供应商状态点阵与账号数，底部信息与「查看配额」贴卡底。
- 待处理问题空态展示 3 行「体检清单」（订阅登录 / 额度与余额 / 路由配置），有 issue 时顶部按类型分组计数。
- 浅色/深色主题下文字对比度、状态色语义均保持可读，颜色不单独承载信息。
- 测试数据（`quota-test-helpers`）下渲染不崩溃、不产生 NaN / 除零。

---

# Analysis

## Related Modules

- `frontend/src/features/dashboard/*`（view / issues / resources / trend / model / use-dashboard）
- `frontend/src/features/quota/quota-model.ts`（复用 `groupQuotaEntries`、`quotaSummary`、`QuotaStatus`）
- `frontend/src/i18n/locales/dashboard-{zh,en}.ts`

## Dependencies

N/A（全部复用既有数据：`accounts` / `providers` / `routes` / `resources.entries`）。

## Risks

- 右列卡变高后若内容仍不动，空洞会转成卡内留白 → 必须同时补内容（已纳入计划）。
- 新增文案需要 zh/en 双份，`dashboardEn` 的类型要求键齐全，漏一个即类型错误。
- 排序/统计口径若与 `dashboardIssues` 不一致，会出现「清单说正常、列表有问题」→ 共用同一组过滤 helper。

---

# Implementation Plan

1. `dashboard-view.tsx`：`items-start` → `items-stretch`；`2xl` 档比例 2:1 → 1.75:1。
2. `dashboard-issues.tsx`：`h-full` + `flex-1` 主体 + `mt-auto` 底部槽位；空态改为体检清单；顶部按 kind 分组计数。
3. `dashboard-resources.tsx`：健康分布条（含图例）、按供应商状态点阵、数字放大、底部槽位收口。
4. `dashboard-trend.tsx` / `.css`：`--trend-h` 断点变量替换 `h-52`；柱宽上限 28px，`MIN_COLUMN_WIDTH` 26。
5. `dashboard-model.ts`：新增 `dashboardChecklist`，与 `dashboardIssues` 共用 enabled/live 过滤 helper。
6. `use-dashboard.ts`：暴露 `checklist`。
7. i18n zh/en 新增键；`dashboard-model.test.ts` 覆盖 checklist。

---

# Progress

- 定位根因：`dashboard-view.tsx` 两处 `items-start` 让右列卡片只占内容高度，行高由左列决定。
- 完成 P0/P1 实现：网格对齐、两张右卡的内容与底部槽位、趋势绘图高度断点变量。
- 隔离 Chromium 2560×1392 / 1440×900 截图核对：两列底边对齐，空态与单条问题态均无行间空洞。
- 首轮截图发现「居中后顶部空、底部空」的两头留白，改为加大行高（py-3 / py-2.5）后重新核对。

---

# Implementation Result

## Changes

- `DashboardView`：两个网格 `items-start` → `items-stretch`；`2xl` 断点把 2fr:1fr 收到 1.75fr:1fr，右列在超宽窗口不再被压窄、左图不再被拉扁。
- `DashboardIssues`：
  - 卡片改为 `flex h-full flex-col`，主体 `flex-1`，待办入口 `mt-auto` 贴卡底；
  - 0 项空态从「一行文案 + 虚线框」改为“结论 + 三行体检清单”（订阅登录 / 额度与余额 / 路由配置，带正常数与待处理徽标）；
  - 有 issue 时顶部按类型（账号 / 额度 / 路由）分组计数，列表 `overflow-y-auto`，下方用一行压出「其余正常」的检查项；
  - 空态框由虚线改为 `bg-muted/30` 实心块。
- `DashboardResources`：
  - 三格数字 `text-2xl` → `text-3xl 2xl:text-4xl`；
  - 新增资源健康分布条（正常 / 耗尽 / 查询失败 / 待确认四段 + 图例 + `role="img"` 文案），用满卡片宽度；
  - 新增「按供应商」状态点阵（每账号一个状态点，最多 8 个）+ 账号数 + `sr-only` 计数说明，行间 `divide-y`；
  - 缓存进度、最近成功查询、检查结果与「查看配额」收进 `mt-auto border-t` 底部槽位。
- `DashboardTrend`：绘图高度改为 `--trend-h` 变量（13 / 14.5 / 16.5rem，1280 / 1680 断点），三处 `h-52` 同步替换；柱宽上限 22px → 28px，`MIN_COLUMN_WIDTH` 22 → 26。
- `dashboard-model.ts`：抽出 `enabledProviderIds` / `liveAccounts` / `liveResources` 供问题列表与新增的 `dashboardChecklist` 共用；后者返回三项 `{key, healthy, problems}`。
- `use-dashboard.ts`：暴露 `checklist`。
- i18n：新增 14 个 `dash.*` 键（zh + en 各一份）。

## Files Changed

- `frontend/src/features/dashboard/dashboard-view.tsx`
- `frontend/src/features/dashboard/dashboard-issues.tsx`
- `frontend/src/features/dashboard/dashboard-resources.tsx`
- `frontend/src/features/dashboard/dashboard-trend.tsx`
- `frontend/src/features/dashboard/dashboard-trend.css`
- `frontend/src/features/dashboard/dashboard-recent.tsx`
- `frontend/src/features/dashboard/dashboard-model.ts`
- `frontend/src/features/dashboard/dashboard-model.test.ts`
- `frontend/src/features/dashboard/use-dashboard.ts`
- `frontend/src/i18n/locales/dashboard-zh.ts`
- `frontend/src/i18n/locales/dashboard-en.ts`
- `docs/tasks/active/TASK-20261030-003-dashboard-cards-adaptive-fill.md`
- `docs/plans/CURRENT.md`

## Technical Notes

- 体检清单与问题列表共用同一过滤口径，并有单测断言「清单 problems 之和 = 问题列表条数」，避免出现「清单说正常、列表有问题」。
- 状态点阵纯色不承载信息：每行附 `sr-only` 的「{{provider}}：正常 N / M 个账号」。健康条用 `role="img"` 提供等价文本。
- 首页仍不发起任何官方额度请求：`enabled: false` 的共享缓存与 `canCheck` 判定未改动。

---

# Verification

## Tests

- [x] Unit Tests（`dashboard-model.test.ts` 新增 checklist 用例；`pnpm vitest run src/features/dashboard` 20 项通过）
- [ ] Integration Tests：N/A
- [ ] E2E Tests：N/A

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

- `format:check` / `lint`（eslint + go vet + staticcheck）/ `typecheck` 通过。
- `test`：Go 全量包通过；前端 145 文件 / 884 项测试通过（含新增 checklist 用例）。
- `build`：Windows amd64 production 构建产出 `bin/omniproxy.exe`（23,289,856 bytes）。
- 首轮 `task verify` 在 3 个无关文件（`accounts-view`、`stepfun`、`agent-update-action`）出现 4 项 5s 超时失败；
  单独重跑这 3 个文件 24 项全绿，第二轮完整 `verify` 亦全绿，判定为全量并发下的既有抖动（见 CURRENT.md
  记载的 Vitest worker 限制），与本次改动无关。
- 视觉核对：隔离 Chromium（headless Chrome，DSF=1）2560×1392 与 1440×900 截图，覆盖 0 项空态与 1 条问题态；
  两列底边对齐，右列无行间空洞。截图留在 `tmp/dashboard-shots/`（已被 `.gitignore` 忽略）。

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

仪表盘视觉调整不涉及产品需求、架构或技术栈变更，只更新 `docs/plans/CURRENT.md` 状态看板。

---

# Deviations

原计划把体检清单与供应商点阵在卡片剩余高度里居中，截图核对后发现上下两头都会留白（内容被推到卡片中部），改为加大行高（清单 `py-3`、供应商行 `py-2.5`）并让内容从顶部自然排布。

---

# Git

---

# Result

---

# Completed
