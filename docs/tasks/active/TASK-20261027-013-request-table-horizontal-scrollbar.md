# TASK-20261027-013: 请求列表横向滚动条样式优化

## Status

In Progress

## Created

2026-10-27 17:30

## Request

用户提供「请求」页截图（1200×800 深色窗口）并指出：这个页面的左右滚动条 UI 样式需要优化。

## Goal

- 请求列表底部的横向滚动条不再呈现 WebView2 原生样式（灰色整幅轨道 + 两端箭头 + 方形端角）。
- 改为与产品其它滚动区一致的细圆角半透明滑块：轨道与箭头透明、滑块 4px 圆角、悬停加深，深浅主题都可辨认。
- 横向滚动本身、列宽、圆角边框、分页与数据契约全部保持不变。

## Background

- `frontend/src/features/activity/request-table.tsx` 的表格外层是 `<div className="overflow-x-auto rounded-xl border">`，横向滚动完全交给浏览器，滚出的是 WebView2 原生横向滚动条。
- `frontend/src/styles.css` 已有 `.app-scroll`（纵向，用于卡片内列表）：透明轨道 + 4px 圆角滑块 + 隐藏两端箭头。横向容器此前没有对应工具类，因此请求页是全应用里少数仍显示原生滚动条的位置。
- 截图现状问题：轨道横贯整个表格宽度、浅灰底在深色卡片里很跳；两端箭头与产品其它控件风格不符；滚动条紧贴表格圆角底边，端角看起来是方的。

## Scope

本任务预计包含：

- `frontend/src/styles.css`：新增横向滚动工具类 `.app-scroll-x`（只作用于横向，不预留 `scrollbar-gutter`，避免表格右侧出现空带）。
- `frontend/src/features/activity/request-table.tsx`：表格外层容器改用该工具类。
- 回归测试：`frontend/src/features/activity/request-metrics.test.tsx` 增加容器类名断言。
- 文档：本 Task Record、`docs/plans/CURRENT.md`。

## Out of Scope

- 不改 `routing-table.tsx`、`gateway-*`、`provider-model-usage.tsx` 等其它横向滚动容器（如需统一另建任务）。
- 不改请求列表的列定义、列宽、截断策略、Tooltip、排序或数据查询／分页逻辑。
- 不引入自绘滑块组件，不改动 `components/common/scroll-region.tsx` 与 `dashboard-trend.css` 的既有实现。
- 不改动纵向 `.app-scroll` 的既有行为。

## Existing Behavior

- 请求列表横向滚动条为原生样式：整幅浅色轨道 + 左右箭头按钮，高度约 14px，深浅主题下都与邻居界面对不上。

## Expected Behavior

- 请求列表横向滚动条只剩 4px 圆角半透明滑块，轨道、箭头与端角透明，悬停时滑块加深；滚动位置与拖动行为不变。
- 表格圆角边框、行列内容与分页栏位置不变（滚动条占位高度由原生 ~14px 降到 10px，表格可视高度略增）。

---

# Analysis

## Related Modules

- 前端：`styles.css`（既有 `.app-scroll` 约定）、`features/activity/request-table.tsx`、`features/activity/request-metrics.test.tsx`。

## Dependencies

无新增依赖。

## Risks

- `scrollbar-gutter: stable` 只适合纵向容器：横向复用会在表格右侧预留空带，因此横向工具类不设置该属性。
- WebView2 是 Chromium，`::-webkit-scrollbar` 生效；若同时设置标准 `scrollbar-width`／`scrollbar-color` 会在新版 Chromium 中覆盖自定义样式，因此沿用 `.app-scroll` 的纯 `::-webkit-scrollbar` 写法。
- 滑块可视高度 4px 依赖透明的 3px 边框形成内缩，指针命中区域仍为 10px；不影响键盘、Shift+滚轮与触控板横滑。

---

# Implementation Plan

1. `styles.css` 在 `.app-scroll` 旁新增 `.app-scroll-x`：10px 滚动条高度、透明轨道／端角、4px 圆角滑块（`border: 3px solid transparent` + `background-clip: content-box`）、悬停加深、隐藏两端箭头、`overscroll-behavior-x: contain`。
2. `request-table.tsx` 外层容器类名加 `app-scroll-x`。
3. `request-metrics.test.tsx` 增加「表格外层使用统一横向滑块工具类」断言。
4. 运行前端 format／lint／typecheck／test 与 `task verify`，回填结果。
5. 更新 `docs/plans/CURRENT.md`，归档 Task Record。

---

# Progress

- 2026-10-27 17:30 建立任务记录，确认请求页是全应用少数仍使用原生横向滚动条的位置，且 `.app-scroll` 已有可复用的滑块样式约定。
- 2026-10-27 17:32 新增 `.app-scroll-x` 并接入请求表；容器类名断言测试通过。
- 2026-10-27 17:35 用生产构建产物 + 无头 Edge 复现同一张表格，深色／浅色都只剩 4px 圆角滑块，原生轨道与箭头消失。
- 2026-10-27 17:52 完整 `task verify` exit 0（132 文件／810 项前端测试、Windows production 构建）。

---

# Implementation Result

## Changes

已实现：

- **新增横向滚动工具类**（`frontend/src/styles.css`）：`.app-scroll-x` 把请求表的 WebView2 原生横向滚动条替换为与 `.app-scroll` 同一套滑块——滚动条高度 10px、轨道与端角透明、滑块用 3px 透明边框内缩成 4px 圆角条（`background-clip: content-box`）、悬停由 50% 加深到 80%、两端箭头 `display: none`；另加 `overscroll-behavior-x: contain` 避免横向滚动链接到外层页面。刻意不设 `scrollbar-gutter`，否则表格右侧会多出一条空带。
- **请求列表接入**（`frontend/src/features/activity/request-table.tsx`）：表格外层容器由 `overflow-x-auto rounded-xl border` 改为 `app-scroll-x overflow-x-auto rounded-xl border`，其余结构、列与列宽不变。
- **回归测试**（`frontend/src/features/activity/request-metrics.test.tsx`）：新增断言，确认表格外层同时带 `app-scroll-x` 与 `overflow-x-auto`，防止后续改回原生滚动条。

## Files Changed

- `frontend/src/styles.css`
- `frontend/src/features/activity/request-table.tsx`
- `frontend/src/features/activity/request-metrics.test.tsx`
- `docs/tasks/active/TASK-20261027-013-request-table-horizontal-scrollbar.md`
- `docs/plans/CURRENT.md`

## Technical Notes

- WebView2 是 Chromium，样式只走 `::-webkit-scrollbar`：同时声明标准 `scrollbar-width`／`scrollbar-color` 会在新版 Chromium 覆盖自定义滑块，因此沿用 `.app-scroll` 的写法。
- 滑块指针命中区域仍是 10px（含透明边框），4px 只是可视高度；键盘、Shift+滚轮、触控板横滑行为不变。
- 视觉核对：用生产 `vite build` 产物 CSS + 无头 Edge 在 1200×800 复现同一张表格（`tmp/scrollbar-preview/`，未入库），前后对比确认原生轨道与箭头消失、滑块圆角内缩；浅色主题同样可辨。

---

# Verification

## Tests

- [x] Unit Tests（`request-metrics.test.tsx` 7 项通过，含新增横向滑块断言）
- [x] Integration Tests（N/A：仅样式工具类，无跨模块行为变化）
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

- 首次运行 `task verify`：格式／lint／类型检查／809 项前端测试与 Go 测试全部通过，但在最后的 Windows 原生构建收尾（`powershell Remove-item *.syso` 之后）以 exit 1 结束，日志未留下可复现错误行；单独重跑 `task build` 与随后完整重跑 `task verify` 均为 exit 0，判断为收尾步骤的偶发失败，非本次改动引起。
- 最终 `task verify`：exit 0，132 个前端测试文件／810 项测试通过（含新增 1 项），Windows production 原生构建产出 `bin/omniproxy.exe`（23,131,136 bytes）。
- 定向测试：`vitest run src/features/activity/request-metrics.test.tsx` 7 项通过。
- 视觉核对：`pnpm run build` 产物 CSS + 无头 Edge，1200×800 深色／浅色各一张，表格底部只剩 4px 圆角滑块，原生轨道与箭头消失；140% 字号与真实原生窗口未单独验收。

---

# Documentation

- [x] CURRENT

其余文档无需更新：产品行为、架构边界、技术栈与测试规范均未改变。

---

# Deviations

1. 原计划：只提「新增横向滚动工具类」。实际方案：在同一条任务里一并补了容器类名回归测试与深浅主题的静态视觉复现（`tmp/scrollbar-preview/`，未入库）。改变原因：仅改 CSS 时没有任何自动检查能防止后续把容器改回 `overflow-x-auto` 裸原生滚动条。
2. 原计划未定滑块透明度。实际采用与 `.app-scroll` 相同的 50%（悬停 80%），以保持全应用滚动条一致，而不是为表格单独调一套颜色。

---

# Git

## Commit

```text
<commit hash> feat(activity): 请求列表横向滚动条改为统一细滑块
```

---

# Result

请求页列表底部的 WebView2 原生横向滚动条（整幅浅色轨道 + 两端箭头）已替换为产品统一的 4px 圆角半透明滑块，悬停加深、箭头隐藏、横向滚动不再链接外层；表格结构、列宽、分页与数据契约保持不变。新增 1 项容器类名回归测试，完整 `task verify` exit 0（132 文件／810 项前端测试、Windows production 构建）。浅色／深色以无头 Edge 静态复现核对，真实原生窗口与 140% 字号待用户视觉验收。
