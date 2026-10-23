# TASK-20261023-016: 仪表盘调用趋势卡片与横向滚动条优化

## Status

In Progress

## Created

2026-10-23 21:30

## Request

用户提供仪表盘「调用趋势」卡片截图并指出：这个调用趋势的 UI 需要优化一下，包含这个滚动条。

## Goal

- 调用趋势卡片在窄卡片宽度下仍然可读：坐标轴数值不压在图里、横轴末尾时间不被裁掉、稀疏柱形不显得单薄。
- 图表底部的 Windows 原生横向滚动条替换为与产品一致的细圆角自绘滑块，出现时不改变卡片结构、可拖动、可在深浅主题下辨识。
- 保持既有数据契约：请求结果分类／Token 两种指标、按本地小时或本地日期分桶、悬浮与键盘焦点查看精确值、空态与「用量未上报」提示不变。

## Background

- `frontend/src/features/dashboard/dashboard-trend.tsx` 使用 `overflow-x-auto` 内部滚动，滚出的是 WebView2 原生滚动条（灰色轨道 + 两端箭头），与侧栏、主内容区自绘 `scroll-region` 细滑块的视觉语言不一致。
- 现状问题（截图 671×383 深色窗口）：
  1. 纵轴刻度用 `absolute right-0` 画在图内右侧，直接压在柱形与网格线上，且刻度值是 `maximum × n / 4`，会出现 1.3K 这类不整的数字。
  2. 横轴标签复用柱形的网格列并 `whitespace-nowrap`，最后一列标签被滚动容器右边缘裁掉，截图里只显示「2」而不是「23」。
  3. 柱形 `w-full` 铺满 26px 网格列，列间只有 4px 间距，柱形与图区没有基线强调，视觉上像孤立的细高条。
  4. 图区高度 `h-52` 内还要容纳横轴标签与滚动条，悬浮反馈只有 Tooltip，没有列高亮。
- `docs/product/DASHBOARD.md` 已说明「图表在窄屏内部滚动」，本次只改呈现方式，不改变分桶与统计口径（ADR-007 不变）。

## Scope

本任务预计包含：

- `frontend/src/features/dashboard/dashboard-model.ts`：新增坐标轴上限取整、刻度值、横轴标签密度的纯函数。
- 新增 `frontend/src/features/dashboard/dashboard-trend.css`：横向滚动区域与自绘滑块的样式。
- 重写 `frontend/src/features/dashboard/dashboard-trend.tsx` 的图表区：左侧纵轴刻度、网格与基线、柱形宽度与列高亮、独立横轴标签行、自绘横向滑块、图例显示本时段合计。
- `frontend/src/i18n/locales/dashboard-{zh,en}.ts`：新增 Tooltip 合计文案。
- 回归测试：`dashboard-model.test.ts` 增加坐标轴用例，新增 `dashboard-trend.test.tsx`。
- 文档：`docs/product/DASHBOARD.md`、`docs/plans/CURRENT.md`。

## Out of Scope

- 不改后端统计、分桶（迁移 010／ADR-007）、`use-dashboard` 查询与刷新逻辑。
- 不改网关页 `gateway-request-trend.tsx` 等其它图表的滚动条（如需统一另行建任务）。
- 不改 `components/common/scroll-region.tsx`（纵向主滚动区）。
- 不改指标卡、待处理问题、最近请求、资源概况、客户端接入等其它仪表盘模块。

## Existing Behavior

- 图区纵轴刻度画在图内右上角；横轴最后一个标签会被裁切；柱形铺满网格列。
- 底部使用浏览器原生横向滚动条（带轨道与箭头），始终横跨整个卡片内容宽度。

## Expected Behavior

- 纵轴刻度在柱形左侧独立栏内，数值为整刻度（如 0／400／800／1.2K／1.6K），不覆盖绘图区；横轴标签行独立成行，首尾标签对齐到边界不被裁切，标签按密度自动抽稀。
- 柱形居中且有最大宽度，稀疏时段不过分单薄；悬浮与键盘焦点时整列背景高亮，Tooltip 显示精确值与合计。
- 需要横向滚动时，图区下方出现 8px 细圆角滑块：默认半透明、悬停加深、可拖动、可点击轨道跳转；不需要滚动时滑块与轨道一起收起，不占高度。
- 深浅主题、140% 字号下刻度与标签不重叠；空态与「Tokend 用量未上报」文案保持原样。

---

# Analysis

## Related Modules

- 前端：`features/dashboard`（趋势卡片、模型纯函数）、`components/ui/{card,button,tooltip}`、`i18n/locales/dashboard-*`、`styles.css` 既有 `.app-scroll` 约定。
- 只读参考：`components/common/scroll-region.tsx`（纵向自绘滑块实现）、`services/dashboard.ts`（`DashboardInfo` / `DashboardSummary` 契约）。

## Dependencies

无新增依赖。

## Risks

- 刻度上限必须用 BigInt 运算：Token 合计会超过 `Number.MAX_SAFE_INTEGER`（现有 `dashboardTokenRatio` 已按 BigInt 处理，不能回退到浮点）。
- 自绘滑块依赖 `scrollWidth` / `clientWidth` 与 `ResizeObserver`；jsdom 不计算布局，测试需模拟尺寸，且组件在零尺寸下必须安全退化。
- Tooltip 触发元素从整列改为列内元素后，键盘焦点顺序与可访问名称不能退化（`role="group"` + 每列 `aria-label` 保持不变）。
- 图表高度与横轴标签行高度必须与左侧刻度栏一致，否则刻度会与网格线错位。

---

# Implementation Plan

1. `dashboard-model.ts`：新增 `dashboardAxisMaximum`（按 5% 余量 + 一位有效数字取整，BigInt）、`dashboardAxisTicks`（上限／3⁄4／1⁄2／1⁄4／0）、`dashboardLabelStep`（按列数抽稀横轴标签）。
2. 新增 `dashboard-trend.css`：`.trend-scroll` 隐藏原生滚动条、`.trend-rail` / `.trend-rail__thumb` 自绘 8px 滑块（`data-scrollable` 控制显隐，`prefers-reduced-motion` 关闭过渡）。
3. 重写 `dashboard-trend.tsx` 图表区：纵轴独立刻度栏 + 网格／基线 + 居中柱形（最大宽度、列悬浮高亮）+ 独立横轴标签行 + 局部 `TrendScroller`（滑块拖动／轨道跳转／尺寸同步）。
4. 图例显示本时段各分类合计（紧凑记数，Tooltip 保留精确值）；Token 模式 Tooltip 增加合计行。
5. 补中英文案（`dash.trendTotal`）。
6. 补回归测试并在非原生渲染下做视觉核对（真实构建 CSS + jsdom DOM 复现截图场景）。
7. `task verify`，更新 `docs/product/DASHBOARD.md` 与 `docs/plans/CURRENT.md`。

---

# Progress

- 2026-10-23 21:30 建立任务记录，定位四个现状问题（刻度压图、横轴末尾裁切、柱形铺满、原生滚动条）。
- 2026-10-23 21:35 完成坐标轴纯函数与自绘滑块，首次以真实 Vite／Tailwind 产物 + 无头 Chromium 截图核对：纵轴刻度回到左侧独立栏、末位小时标签不再被裁切。
- 2026-10-23 21:40 按截图复核调整：网格顶／基线改为实线、删除零值列的基线短刻度（读成噪点）、列宽 26→22px 使今日 24 桶在 620–640px 卡片内不再需要横向滚动。
- 2026-10-23 21:45 核对 Token 模式（万级刻度）、浅色主题、140% 字号与 Tooltip 键盘焦点状态，均无重叠或裁切；新增 3 项坐标轴与 4 项图表回归测试。
- 2026-10-23 22:05 完整质量门 `task verify` exit 0（126 文件 / 747 项前端测试）。

---

# Implementation Result

## Changes

已实现：

- **坐标轴纯函数**（`dashboard-model.ts`）：`dashboardAxisMaximum` 先加 5% 余量再按一位有效数字向上取整（全程 BigInt，Token 合计超过 `Number.MAX_SAFE_INTEGER` 时不失真），`dashboardAxisTicks` 给出上限／3⁄4／1⁄2／1⁄4／0 五个刻度，`dashboardLabelStep` 把横轴标签压到最多 6 个且末位始终显示（24 桶→每 5 个、7 天→每 2 天、30 天→每 6 天）。
- **纵轴改为左侧独立刻度栏**：`w-9 sm:w-10` 的固定栏在滚动容器之外，只滚动绘图区时刻度保持可见；刻度按整数值用紧凑记数（`800`／`1.2K`／`800万`），首项贴顶、末项贴基线，不再压在图内。
- **网格与柱形**：三条虚线中间刻度 + 顶／基线实线；柱形居中（列宽 58%，最小 4px、最大 22px）不再铺满网格列，零值列不再画基线短刻度；悬浮或键盘焦点时整列背景高亮，Tooltip 内每项带色点与精确值，多序列时附「合计」行。
- **横轴标签独立成行**：标签行在滚动内容内与柱形同列对齐，标签按密度抽稀，首／末标签分别贴左／右边界，末位小时（如 `23`）不再被右边缘裁掉。
- **自绘横向滑块**（`dashboard-trend.css` + `TrendScroller`）：隐藏 WebView2 原生滚动条，改为 8px 轨道内的 4px 细圆角滑块；`ResizeObserver` 同步尺寸，可拖动、可点击轨道跳转，`data-scrollable="false"` 时滑块与轨道一起收起不占高度，`prefers-reduced-motion` 关闭过渡。
- **图例**：每个分类旁显示本时段合计（紧凑记数，`tabular-nums`），色点改为圆形与 Tooltip 行一致。
- **列宽**：每列最小 22px，使今日 24 桶在常见 620–640px 卡片内不需要横向滚动，仅在窄卡片或 30 天时段出现滑块。
- **文案**：中英文各新增 `dash.trendTotal`（合计／Total）。

## Files Changed

- `frontend/src/features/dashboard/dashboard-trend.tsx`（图表区与 `TrendScroller` 重写）
- `frontend/src/features/dashboard/dashboard-trend.css`（新增）
- `frontend/src/features/dashboard/dashboard-trend.test.tsx`（新增）
- `frontend/src/features/dashboard/dashboard-model.ts`、`dashboard-model.test.ts`
- `frontend/src/i18n/locales/dashboard-zh.ts`、`dashboard-en.ts`
- `docs/product/DASHBOARD.md`、`docs/plans/CURRENT.md`

## Technical Notes

- 纵向主滚动区 `scroll-region` 只处理纵轴并有页面滚动恢复，趋势图需要的是纯横向、无页面语义的滑块，因此新增局部 `TrendScroller`，复用同一套「隐藏原生滚动条 + 自绘滑块 + `ResizeObserver` 同步」做法，未改动全局滚动组件。
- 滑块几何按 `scrollWidth / clientWidth` 比例计算，拖动与点击轨道都按 `rail.clientWidth - thumb.offsetWidth` 的行程换算；零尺寸（jsdom、隐藏容器）时直接退化不报错。
- 柱形高度仍走既有 `dashboardTokenRatio`（BigInt 千分比），坐标轴上限同样用 BigInt，绘制时再转百分比字符串，避免 9e20 级 Token 合计精度丢失。
- 堆叠顺序保持「成功／失败／已取消／进行中」自下而上（Token 为输入／输出），极小占比的分类只留 1px 最小高度，不会因取整消失。
- 140% 字号下 `text-[10px]`、`h-52`、`w-9` 均为 rem 基准，只有柱形最大宽度是固定 px，因此放大字号只让柱形相对变细，不产生重叠。

---

# Verification

## Tests

- [x] Unit Tests（`dashboard-model.test.ts` 新增 3 项：整刻度取整与刻度值、超出安全整数的 Token 轴、标签抽稀规则；`dashboard-trend.test.tsx` 新增 4 项：刻度与末位小时、周标签、滑块宽度／位移／拖动跳转、空态）
- [x] Integration Tests（前端 126 个文件 / 747 项测试全部通过）
- [ ] E2E Tests（N/A，无 E2E 框架）

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
PASS（format:check → go vet → staticcheck → eslint → tsc → go test ./internal/... → vitest → wails3 production build，exit 0）
前端 126 文件 / 747 项测试通过；production 构建产物 dist/assets/index-*.css 95.59 kB、index-*.js 1,450.34 kB。
```

视觉核对（真实 Vite／Tailwind 产物 + 无头 Chromium 截图，非原生 Wails 窗口）：

- 深色 640px 卡片（今日 24 桶）：刻度 800／600／400／200／0 在左侧栏，横轴 00／05／10／15／20／23 全部可见、末位不裁切，柱形居中，无滚动条。
- 深色 1000px 与 620px（近 7 天）、380px（近 30 天）：宽卡片无滑块，380px 卡片出现 8px 细圆角滑块且可拖动；380px 今日视图列宽变窄后仍无横向溢出。
- Token 模式：纵轴为 0／200万／400万／600万／800万，图例为「输入 Token／输出 Token」及其合计。
- 浅色主题：刻度、虚线网格与基线与深色主题对比一致，滑块为半透明浅灰。
- 140% 根字号：图例换行、刻度与横轴标签不重叠，滑块随图区缩放。
- 键盘焦点：聚焦某一列后弹出 Tooltip（时间、四项数值、合计行）并显示整列焦点环。

原生 Wails 窗口内的实际视觉与真实数据未在本任务内验收，待用户确认。

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

更新：`docs/product/DASHBOARD.md`（趋势呈现与横向滚动条说明）、`docs/plans/CURRENT.md`（本任务条目）。
ADR-007 的分桶与统计口径未变，无需新增 ADR。

---

# Deviations

- 原计划只写「按一位有效数字取整」；实际在取整前加 5% 余量，避免最高柱贴到坐标轴顶线。
- 原计划未定列宽；实际把每列最小宽度从 26px 降到 22px，使今日 24 桶在常见 620–640px 卡片内完全可见，只有更窄卡片或 30 天时段才需要滑块。
- 原计划在零值列保留基线短刻度以维持节奏；实现后截图显示会被读成断续虚线噪点，改为只保留整条基线。
- 原计划把滑块写成一次性内联逻辑；实际抽成组件内 `TrendScroller`（含 `ResizeObserver` 同步与指针拖动），与既有 `scroll-region` 保持同一套交互习惯。

---

# Git

## Commit

（待填写）

---

# Result

（待填写）

---

# Completed

（待完成）
