# TASK-20261027-014: 请求页改为表格内部滚动

## Status

Completed（2026-10-27 18:12）

## Created

2026-10-27 18:05

## Request

用户提供「请求」页截图（1200×800 深色窗口）并指出：请求页面不要让列表的内容超出屏幕范围，不要让用户滚动外部内容区，可以滚动表格内容区的内容。

## Goal

- 请求页固定在可视区域内：标题、搜索／筛选、分页栏各占一行，只有表格内容区内部滚动。
- 表格纵向滚动时表头吸顶，横向滚动条沿用产品统一的细滑块。
- 列表条数、翻页、搜索、状态筛选、行详情与横向列宽均不变。

## Background

- 现状：`requests-view.tsx` 根节点是普通块级 `space-y-5` 段落，每页 50 行全部铺开，页面高度随行数增长，滚动的是根布局 `ScrollRegion`（外部内容区）。截图里第 9 行之后的行被窗口底边裁掉，整页需要外部滚动才能看到。
- 日志页（`features/logs/logs-view.tsx`）已经采用「页面占满可视区 + 列表内部 `ScrollRegion`」的布局：`h-[calc(100dvh-5.75rem)]`（标题栏 2.75rem + 根页面上下留白 3rem）+ `flex-1 min-h-0` 列表区。请求页没有跟上这套布局。
- 上一任务（TASK-20261027-013）给表格外层加了 `.app-scroll-x` 横向细滑块；改为内部滚动后需要竖向滚动条与吸顶表头。

## Scope

本任务预计包含：

- `frontend/src/features/activity/requests-view.tsx`：根节点改为占满可视区的纵向 flex，表格区 `flex-1 min-h-0`，其余区块 `shrink-0`。
- `frontend/src/features/activity/request-table.tsx`：外层改为可裁剪的 flex 容器，内层 `overflow-auto` 滚动视口 + `app-scroll` 细滑块，表头 `app-table-head` 吸顶，新增 `className` 由页面控制高度。
- `frontend/src/styles.css`：新增 `.app-page-fit`（占满可视区高度）与 `.app-table-head th`（不透明明暗一致的表头吸顶底色 + 下边线）；删除已被取代的 `.app-scroll-x`。
- `frontend/src/features/logs/logs-view.tsx`：改用同一个 `.app-page-fit`，去掉重复的 `h-[calc(100dvh-5.75rem)]` 魔法值。
- 回归测试：`request-metrics.test.tsx`、`requests-pagination.test.tsx`。
- 文档：本 Task Record、`docs/plans/CURRENT.md`。

## Out of Scope

- 不改列定义、列宽、截断、Tooltip、排序、搜索／筛选／分页查询参数与每页 50 条。
- 不改请求详情弹窗 `request-dialog.tsx`。
- 不改路由表、用量页、网关页等其它列表的滚动方式。
- 不改根布局 `__root.tsx`、`ScrollRegion` 与标题栏高度。

## Existing Behavior

- 请求页内容高度 = 表头 + 50 行 + 分页栏，超过窗口高度后由根布局的外部滚动区滚动；表格自身没有纵向滚动，表头随页面滚走。

## Expected Behavior

- 请求页占满内容区可视高度，不产生外部滚动：标题、搜索／状态筛选、分页栏与保留条数提示固定可见。
- 表格区域按剩余高度内部滚动（每页仍是 50 条），表头吸顶，横向仍可滚动且只有 4px 圆角滑块。
- 深浅主题下表头底色不透明，滚动行不会从表头后面透出。

---

# Analysis

## Related Modules

- 前端：`features/activity/{requests-view,request-table}.tsx`、`features/logs/logs-view.tsx`、`styles.css`、`components/common/scroll-region.tsx`（只读参考）。

## Dependencies

无新增依赖。

## Risks

- `100dvh - 5.75rem` 依赖标题栏 `h-11` 与根页面 `py-6`；抽成 `.app-page-fit` 后若将来改标题栏高度只需改一处，但日志页与请求页会同时受影响。
- 吸顶表头必须有**不透明**背景，否则滚动行会透出；用 `color-mix(var(--muted) 30%, var(--background))` 复现原 `bg-muted/30` 的观感并保持深浅主题一致。
- 表格外层加 `overflow-hidden` 后圆角才不会被吸顶表头与滚动条破坏；横向滚动容器改为内层 div，`.app-scroll-x` 不再需要。
- jsdom 不计算布局，测试只能断言结构类名，真实高度效果需静态复现核对。

---

# Implementation Plan

1. `styles.css`：新增 `.app-page-fit`、`.app-table-head th`（`position: sticky; top: 0; z-index: 10;` + 不透明底色 + `inset 0 -1px` 下边线）；删除 `.app-scroll-x` 整段。
2. `request-table.tsx`：新增可选 `className`；外层 `flex min-h-0 flex-col overflow-hidden rounded-xl border` + 内层 `app-scroll min-h-0 flex-1 overflow-auto`；`thead` 改用 `app-table-head`。
3. `requests-view.tsx`：根节点 `app-page-fit flex min-h-0 w-full flex-col gap-5`，header／筛选／提示／footer 加 `shrink-0`，表格加 `min-h-0 flex-1`。
4. `logs-view.tsx`：`h-[calc(100dvh-5.75rem)]` 换成 `app-page-fit`。
5. 回归测试：表格结构（`app-scroll` + `app-table-head`）与页面布局（`app-page-fit`、表格区 `flex-1`）断言。
6. `task verify`，生产 CSS + 无头 Edge 在 1200×800 复现核对，回填文档并归档。

---

# Progress

- 2026-10-27 18:05 建立任务记录，确认根因是请求页沿用普通块级布局、由根 `ScrollRegion` 外部滚动，而日志页已有「页面占满 + 列表内部滚动」的既有做法可直接对齐。
- 2026-10-27 18:00 页面与表格布局改完，activity／logs 定向测试 37 项通过。
- 2026-10-27 18:07 生产 CSS + 无头 Edge 复现 1200×800：`outer scroll: false`、`inner scroll: true`（scrollH 1706 / clientH 456），滚动后表头吸顶、横向滑块仍在。
- 2026-10-27 18:09 完整 `task verify` exit 0（132 文件／811 项前端测试、Windows production 构建）。

---

# Implementation Result

## Changes

已实现：

- **页面占满可视区**（`requests-view.tsx`）：根节点由 `space-y-5 pb-6` 块级段落改为 `app-page-fit flex min-h-0 flex-col gap-5`；标题、搜索／状态筛选、错误与加载提示、分页栏全部 `shrink-0`，只有表格容器 `min-h-0 flex-1`。空列表提示改为撑满表格区并居中，不再在长页面上留一大片空白。
- **表格内部滚动**（`request-table.tsx`）：新增可选 `className`；外层只做圆角裁剪与高度分配（`flex min-h-0 flex-col overflow-hidden rounded-xl border`），内层 `app-scroll min-h-0 flex-1 overflow-auto` 承担纵横两个方向的滚动；`thead` 改用 `.app-table-head` 吸顶。
- **样式工具类**（`styles.css`）：新增 `.app-page-fit`（`calc(100dvh - 5.75rem)`，= 标题栏 2.75rem + 根页面 `py-6`）与 `.app-table-head th`（`position: sticky; top: 0; z-index: 10`，`color-mix(var(--muted) 30%, var(--background))` 不透明底色 + `inset 0 -1px` 下边线）；删除上一任务引入、本任务已取代的 `.app-scroll-x`。
- **去重**（`logs-view.tsx`）：日志页原来手写的 `h-[calc(100dvh-5.75rem)]` 改为同一个 `.app-page-fit`，两页共用一处高度定义。
- **回归测试**：表格结构（内层 `app-scroll` + `overflow-auto`、外层 `overflow-hidden`、`thead` 带 `app-table-head`）与页面布局（根 `section` 带 `app-page-fit`／`flex-col`、不再有 `space-y-5`）各 1 项。

## Files Changed

- `frontend/src/features/activity/requests-view.tsx`
- `frontend/src/features/activity/request-table.tsx`
- `frontend/src/features/logs/logs-view.tsx`
- `frontend/src/styles.css`
- `frontend/src/features/activity/request-metrics.test.tsx`
- `frontend/src/features/activity/requests-pagination.test.tsx`
- `docs/tasks/completed/2026-10/TASK-20261027-014-requests-view-internal-scroll.md`（实现期间位于 `docs/tasks/active/`）
- `docs/plans/CURRENT.md`

## Technical Notes

- 根布局的 `ScrollRegion` 视口本身是 `flex-1`，但其内容包裹层没有确定高度，子级 `h-full` 无法解析，所以继续沿用日志页的 `calc(100dvh - 标题栏 - py)` 写法；现在抽成 `.app-page-fit` 后只保留一处魔法值。
- 吸顶表头必须是**不透明**背景：原来的 `bg-muted/30` 是半透明，滚动行会从表头下面透出；`color-mix(in oklab, var(--muted) 30%, var(--background))` 复现原观感且不透明，深浅主题一致。
- 表格滚动视口沿用 `.app-scroll`：纵横同为 10px 槽位、透明轨道、4px 圆角滑块、隐藏箭头；`scrollbar-gutter: stable` 顺带避免筛选后行数变化导致的列宽抖动。
- 视觉核对（`tmp/requests-fit/`，未入库）：1200×800 复现整页（标题栏 + 根留白 + 页面），诊断值 `main scrollH=664 / clientH=664`（外部不滚动）、`table scrollH=1706 / clientH=456`（内部滚动）、`scrollW=1123 / clientW=867`（横向可滚）；滚动到中间后表头仍在顶部，横向滑块保持 4px 圆角，浅色主题同样成立。

---

# Verification

## Tests

- [x] Unit Tests（activity／logs 定向 37 项、全量 811 项通过，含新增 1 项布局断言）
- [x] Integration Tests（N/A：纯布局与样式变化，无跨模块契约变化）
- [x] E2E Tests（N/A：原生窗口视觉以无头 Edge 静态复现核对）

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
PASS
```

- `task verify` exit 0：132 个前端测试文件／811 项测试通过，Go 测试、格式、lint、类型检查全部通过，Windows production 原生构建产出 `bin/omniproxy.exe`（23,130,624 bytes）。
- 定向测试：`vitest run src/features/activity src/features/logs` 8 个文件／37 项通过。
- 视觉核对：生产 `vite build` 产物 CSS + 无头 Edge 1200×800，深色／浅色各一张；外部滚动区 `scrollHeight === clientHeight`，表格内部可纵横向滚动，滚动后表头吸顶。真实原生窗口、140% 字号与窄窗口高度未单独验收。

---

# Documentation

- [x] CURRENT

其余文档无需更新：产品行为、架构边界与技术栈未变；`docs/plans/CURRENT.md` 的「原生窗口视觉」待验收项在归档时同步。

---

# Deviations

1. 原计划只改请求页与表格。实际方案：顺带把日志页的 `h-[calc(100dvh-5.75rem)]` 换成同一个 `.app-page-fit`。改变原因：该高度依赖标题栏与根页面留白，若只在请求页复制一份，将来改标题栏高度会漏改其中一页。
2. 原计划保留上一任务的 `.app-scroll-x`。实际方案：表格视口改用 `.app-scroll`（纵横都有细滑块）并删除 `.app-scroll-x`。改变原因：内部滚动后同一视口既要纵向也要横向滑块，保留两个只差一个轴的类会变成死代码。
3. 原计划未提空态。实际方案：空列表提示改为撑满表格区并居中。改变原因：页面固定高度后，若不撑满，空态下方会留出一大块空白。

---

# Git

## Commit

```text
82c28ed3 feat(activity): 请求页改为表格内部滚动
0ebfca15 docs(tasks): 归档 TASK-20261027-014 并记录提交 hash
```

## Related Tasks

- 前序：[TASK-20261027-013](TASK-20261027-013-request-table-horizontal-scrollbar.md) 引入的 `.app-scroll-x` 在本任务被 `.app-scroll` 取代（横向细滑块的观感不变）。

---

# Result

请求页已固定在内容区可视高度内：标题、搜索／状态筛选、保留条数与分页常驻，50 行只在表格内部纵横向滚动，表头吸顶且底色不透明；页面不再把整页顶出窗口，根滚动区不再需要滚动。高度定义抽成 `.app-page-fit` 由请求页与日志页共用，上一任务的 `.app-scroll-x` 被 `.app-scroll` 取代。新增 1 项布局回归测试，完整 `task verify` exit 0（132 文件／811 项前端测试、Windows production 构建）。深色／浅色已用无头 Edge 复现核对，真实原生窗口、140% 字号与窄窗口高度待用户验收。

---

# Completed

2026-10-27 18:12
