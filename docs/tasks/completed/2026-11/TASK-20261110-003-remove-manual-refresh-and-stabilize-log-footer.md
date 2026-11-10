# TASK-20261110-003: 移除手动刷新按钮并稳定日志页脚高度

## Status

Completed

## Created

2026-11-10 09:05

## Request

用户提供三张界面截图，要求修改两个点：

1. 不需要仪表盘（首页）右上角的「刷新概况」按钮，页面会自动刷新，这个按钮没有必要；「用量」页面和「请求」页面的刷新按钮也不需要。
2. 日志页底部条（左侧「当前运行记录 · 采集级别 INFO · 清空显示保留文件日志」，右侧「回到最新」按钮或「自动跟随最新」文字）中，按钮与文字描述的占用高度不一致，导致页面跳动，影响体验，需要优化显示效果。

## Goal

- 仪表盘、用量、请求三个页面不再提供手动刷新按钮，只保留既有自动刷新机制；随之失效的 hook 字段、图标导入与 i18n 键一并清理。
- 日志页脚在「回到最新」按钮与「自动跟随最新」文字之间切换时高度保持不变，日志列表不再上下跳动。

## Background

- 仪表盘统计由 `activity:updated` 事件与窗口重新可见触发重新读取，另有 30 秒本地时钟只用于日期边界。
- 用量页与请求页的 TanStack Query 使用 `refetchInterval: 5000`，本就每 5 秒自动刷新。
- 日志页脚右侧原为 `size="sm"` 按钮（32px）与 `text-xs` 纯文本（16px）二选一，页脚行高随状态变化；页脚是 `app-page-fit` 纵向布局的最后一项，`flex-1` 的日志列表被动伸缩，切换状态即整页跳动。

## Scope

- 仪表盘视图与 `useDashboard`
- 用量／请求视图
- 日志页脚
- 相关 i18n 文案与前端测试
- `docs/product/DASHBOARD.md`、`ACTIVITY.md`、`LOGS.md`、`docs/plans/CURRENT.md`

## Out of Scope

- 配额、账号、网关、路由、存储等其它页面的刷新入口（执行真实官方查询或非周期读取，仍需手动触发）。
- 修改任何自动刷新周期、事件协议或后端接口。
- 仪表盘「检查资源」入口（用户主动检查官方额度，不是刷新按钮）。

## Existing Behavior

- 仪表盘标题右侧为「今日／近 7 天／近 30 天」与「刷新概况」按钮，按钮同时重读统计、账号、提供商、路由和客户端。
- 用量页、请求页标题右侧各有「刷新」按钮，`refetch` 当前查询；读取中禁用并旋转图标。
- 日志页脚右侧在 32px 按钮与 16px 文本之间切换，切换时页脚高度随之变化。

## Expected Behavior

- 三个页面标题区只保留时段／筛选等必要控件，无手动刷新按钮；数据仍按既有事件与 5 秒轮询自动更新。
- 日志页脚无论显示「回到最新」按钮还是「自动跟随最新」文字，右侧槽位恒为 32px，切换时日志列表高度不变。
- 刷新相关的失效 hook 字段、图标导入与 i18n 键被删除，不留死代码。

---

# Analysis

## Related Modules

- `frontend/src/features/dashboard/`：视图、`useDashboard`、统计订阅
- `frontend/src/features/activity/`：`usage-view.tsx`、`requests-view.tsx`
- `frontend/src/features/logs/log-footer.tsx`
- `frontend/src/i18n/locales/`：`activity-*.ts`、`dashboard-*.ts`
- 现有前端测试与产品文档

## Dependencies

无新增依赖。

## Risks

- Low：删除按钮后，仪表盘统计失败时只能等待下一次事件／窗口可见；页面本就会自动重试，错误文案同步改为「稍后自动重试」。
- 日志页脚固定 32px 后窄窗口仍允许换行，但右侧槽位恒为 32px，两种状态高度一致。

## Root Cause

图二／图三的跳动来自页脚右侧两个分支的固有高度差：`Button size="sm"` 为 `h-8`（32px），跟随态只是 `text-xs` 文本（约 16px）。页脚高度变化 16px，而上方的日志卡片是 `flex-1`，只能被动收缩／拉伸，浏览器在每次「暂停／继续／向上翻阅」时重算列表高度，视觉上就是整页跳动。按钮与文案本身没有对齐或行高问题，是两条分支各占各的高度。

## Fix

把右侧分支收进统一的 `div[data-slot="log-follow-slot"]`，固定 `h-8` 并垂直居中；两种状态只替换槽位内部的内容，页脚高度不再随状态变化。

## Regression Prevention

`logs-view.test.tsx` 新增「自动跟随提示与回到最新按钮共用固定行高，切换不改变页脚高度」：断言槽位存在 `h-8`，跟随态显示文案、暂停态在同一槽位显示按钮，且 class 不变。

---

# Implementation Plan

1. 删除仪表盘、用量页、请求页的刷新按钮，并清理只为按钮存在的 `refresh`／`refreshing` 字段与图标导入。
2. 日志页脚右侧改为固定 `h-8` 的槽位，按钮与文字共用同一容器。
3. 删除失效 i18n 键 `dash.refresh`、`activity.refresh`，把 `dash.failed` 改为自动重试语义（中英同步）。
4. 更新受影响测试，并新增日志页脚高度稳定的回归断言。
5. 更新产品文档与 `CURRENT.md`，运行质量门，提交并归档任务记录。

---

# Progress

- 定位页脚跳动根因：按钮 32px 与文字 16px 的固有高度差，页脚伸缩挤动 `flex-1` 日志列表。
- 三个页面的数据本就自动刷新（事件 + 5 秒轮询），删除按钮不影响数据更新。
- 完成代码、文案、测试与文档修改；全量前端测试（限 4 worker）146 文件／893 项通过。

---

# Implementation Result

## Changes

- 仪表盘：移除「刷新概况」按钮；`useDashboard` 删除 `refresh`／`refreshing`（只被该按钮使用），头部只保留时段切换。
- 用量页、请求页：移除「刷新」按钮与相关图标／`isFetching` 使用；两页仍按 `refetchInterval: 5000` 自动重读。
- 日志页脚：右侧改为固定 `h-8` 的 `log-follow-slot`，按钮与「自动跟随最新」文案共用槽位；页脚高度恒定。
- i18n：删除 `dash.refresh`、`activity.refresh`；`dash.failed` 改为「运行统计读取失败，稍后自动重试。」／“Could not read request statistics. Retrying automatically.”。
- 测试：仪表盘失败场景改为通过 `activity:updated` 事件自动恢复并断言不存在「刷新概况」；请求分页不再依赖刷新按钮判定加载完成，并断言无「刷新」按钮；用量页断言无「刷新」按钮；日志新增固定行高回归测试。
- 文档：更新 `DASHBOARD.md`（刷新章节）、`ACTIVITY.md`（自动重读与无手动刷新）、`LOGS.md`（页脚固定行高）。

## Files Changed

- `frontend/src/features/dashboard/dashboard-view.tsx`
- `frontend/src/features/dashboard/use-dashboard.ts`
- `frontend/src/features/activity/usage-view.tsx`
- `frontend/src/features/activity/requests-view.tsx`
- `frontend/src/features/logs/log-footer.tsx`
- `frontend/src/i18n/locales/activity-zh.ts`、`activity-en.ts`、`dashboard-zh.ts`、`dashboard-en.ts`
- `frontend/src/features/dashboard/dashboard-view.test.tsx`
- `frontend/src/features/activity/activity.test.tsx`
- `frontend/src/features/activity/requests-pagination.test.tsx`
- `frontend/src/features/logs/logs-view.test.tsx`
- `docs/product/DASHBOARD.md`、`ACTIVITY.md`、`LOGS.md`
- `docs/plans/CURRENT.md`

## Technical Notes

- 槽位使用 `data-slot="log-follow-slot"` 仅用于结构与回归断言，不参与业务逻辑；`h-8` 与 `Button size="sm"` 的高度同源。
- 请求页的上翻／下翻按钮仍按原有 `data.isFetching` 判定禁用，行为未变。
- 仪表盘统计失败后的恢复依赖既有 `activity:updated` 与 `visibilitychange` 订阅，不新增轮询。

---

# Verification

## Tests

- [x] Unit Tests：新增／更新 4 个前端测试文件，针对仪表盘、请求分页、用量页与日志页脚。
- [ ] Integration Tests：N/A
- [ ] E2E Tests：N/A

## Quality Checks

- [x] Format（`prettier --check` 与 `gofmt` 检查通过）
- [x] Lint（Go `vet`／`staticcheck` 与 ESLint 通过；ESLint 仅 2 条既有 `react-refresh` warning，位于未改动的 `reset-credit-list.tsx`）
- [x] Type Check（`tsc --noEmit`）
- [x] Test（Go 全部包通过；前端 146 文件／893 项在 `--maxWorkers=4` 下全部通过）
- [x] Build（`task build`：`tsc && vite build --mode production`、Wails 绑定生成与 `bin/omniproxy.exe` 生产构建 exit 0）
- [x] Verify（`task verify` 已运行；结果见下）

## Verification Result

```text
task verify
exit 1（前端测试阶段）
```

`task verify` 的 format／lint／typecheck／Go 测试均通过；`vitest run` 在默认高并发下，3 个与本次改动无关的既有文件各出现超时／输入竞态失败（`accounts/stepfun.test.tsx`、`accounts/kimi-code.test.tsx`、`gateway/agent-installation.test.tsx`，5 秒超时）。这 3 个文件单独重跑 19/19 通过；完整前端套件加 `--maxWorkers=4` 后 146 文件／893 项全部通过（413s）。失败用例与本任务改动的页面、组件无调用关系。

```text
task build
exit 0
```

生产构建生成 `bin/omniproxy.exe`（Windows amd64，`-tags production`），前端 `tsc && vite build --mode production` 与 Wails 绑定生成均通过。

---

# Documentation

- [x] PRD：无需修改（未改变产品范围）
- [x] Architecture：无需修改
- [x] Tech Stack：无需修改
- [x] Coding Rules：无需修改
- [x] Testing：无需修改
- [x] Build：无需修改
- [x] ADR：无需修改
- [x] CURRENT：已更新

---

# Deviations

- 原计划仅删除按钮；实施时同步把 `dash.failed` 文案从「请刷新重试」改为「稍后自动重试」，因为按钮移除后该提示指向了不存在的操作。
- 原计划运行 `task verify` 一次通过；实际在其前端测试阶段遇到与本次改动无关的并发超时，改用限 4 worker 的全量前端测试并单独重跑失败文件佐证。

---

# Git

## Commit

```text
d383ccc7 fix(ui): 移除手动刷新按钮并稳定日志页脚高度
```

本次提交包含代码、测试、失效 i18n 键与产品文档；任务记录、`CURRENT.md` 更新及归档单独提交。工作区中原有的未跟踪文件 `docs/tasks/completed/2026-10/TASK-20261029-010-subq-provider.md` 与本任务无关，保持原样未提交。未推送远端。

---

# Result

三个页面的手动刷新按钮已移除，数据仍按原机制自动更新；日志页脚两种状态共用 32px 行高，切换不再引起页面跳动。全量前端测试（限 4 worker）与 Go 测试通过。

---

# Completed

2026-11-10 09:35
