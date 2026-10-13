# TASK-20261010-005: 每个页面独立记录和恢复滚动位置

## Status

Completed

## Created

2026-10-10 15:37（Asia/Singapore）

## Request

“每个页面的滚动进度应该单独记录，而不是共用一个滚动进度。”

## Goal

首次进入页面从顶部开始；页面之间切换后回到各自之前的位置，异步内容加载和自绘滚动条正常配合。

## Analysis

根布局一直复用同一个主内容滚动容器，路由变化没有保存／恢复逻辑，所以页面共用 scrollTop。侧栏和账号卡片内部使用独立容器。需按页面地址区分主内容位置，并避免异步内容暂时变短时覆盖已保存的位置。

选择在现有 ScrollRegion 增加可选页面键，根布局传入已提交路由匹配的 pathname；记录保存在当前窗口内存中。路由内置恢复面向历史记录／整个文档，本需求只需要持久容器的页面级恢复，还要等待异步内容高度，因此使用局部 hook，不新增依赖。查询／筛选属于同一页，动态账号详情按实际地址区分。网关页签已有切换回顶规则保留；页面重新进入时由父层恢复该页位置。

真实浏览器验证发现 useLocation 的地址更新早于 Outlet 渲染，可能用上一页高度提前完成恢复，改用 useMatches 的已提交页面键，回归场景通过。

## Implementation Plan

1. 检查路由恢复能力和网关页签的现有滚动重置，选择最小兼容方案。
2. 在主内容导航边界记录／恢复各页面位置，保留内部容器行为。
3. 验证页面来回切换、首次访问、动态详情和异步内容；运行必要回归及 `task verify`。
4. 更新外观说明、当前看板和实际结果，检查本任务增量并归档。

## Implementation Result

- 根布局以已提交的路由 pathname 作为页面键；不同账号详情分别记忆，同页查询参数不重置位置。
- ScrollRegion 的可选页面键启用窗口内存记录；首次访问归零，重新进入和前进／后退恢复各自的位置。
- 提取 57 行 useScrollRestoration hook，通过 ResizeObserver 等待短加载态变高，不保存临时裁剪后的数值；用户滚动／交互后停止等待，过期回调不能覆盖新页面。
- 自绘滑块与恢复同步；侧栏和卡片内部容器保持原逻辑，网关页签回顶行为保留。无新增依赖，无后端改动。
- 主要文件：`frontend/src/app/routes/__root.tsx`、`frontend/src/components/common/scroll-region.tsx`、`use-scroll-restoration.ts` 及两个就近回归测试；同步 `docs/product/APPEARANCE.md` 和 `docs/plans/CURRENT.md`。

## Limitations

位置只保留在当前窗口会话，重新加载／关闭后重置；恢复目标遇到永久缩短的内容时先显示当前可达位置，用户交互后以新位置为准。

## Verification

- 新增 10 条回归：独立页面和滑块、异步短内容、四种主动操作、过期回调、动态详情与历史导航、同页查询、延迟路由提交；组件测试在 StrictMode 下运行。
- 隔离 Chromium 使用真实根布局／路由／ScrollRegion，模拟业务内容与侧栏，360×500、1200×800、1920×900（140% 字号）均通过：来回切换／前进后退、动态详情、异步加载、滚轮取消、轨道点击后恢复、侧栏独立、无横向溢出，pageErrors 为 0。
- TypeScript 单独检查通过；`git diff --check` 通过。
- 最终 `task verify BIN_DIR=C:/Users/mimanchi/.codex/tmp/omniproxy-page-scroll-20261010/bin` PASS：格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、58 文件／297 前端测试、Vite 生产构建及 Windows 原生构建均通过。构建写入临时目录，未覆盖运行中的开发程序；既有前端包体积提示未阻断构建。
- 产物、截图、基线和增量 patch：`C:/Users/mimanchi/.codex/tmp/omniproxy-page-scroll-20261010/`。
- 已确认开发窗口继续运行并热更新。真实客户端配置和当前网关状态未作测试操作。

## Git

Not committed by Agent.

工作区已有大量未提交修改，ScrollRegion 等文件包含此前尚未提交的改动，相关文档和模块也处于共享新增状态；仅提交整文件会混入其他任务，不能独立形成可构建增量。本任务已按开始前基线保存 `implementation.patch`、检查增量与 Git 状态，保留所有既有修改。

## Result

各页面滚动位置独立，首次访问从顶部开始，返回原页恢复位置，异步内容和用户主动操作兼容；完整质量门和浏览器验收通过，开发窗口可直接查看。

## Completed

2026-10-10 15:56（Asia/Singapore）
