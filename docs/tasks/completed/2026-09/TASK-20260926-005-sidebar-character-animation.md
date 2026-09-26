# TASK-20260926-005: 为侧栏文字增加逐字收展动画

## Status

Completed

## Created

2026-09-26 10:00

## Request

收起展开时的文字消失、出现，希望做成一个字一个字消失、出现的动画。

## Goal

展开时文字从左往右逐字出现，收起时从右往左逐字消失；保持图标位置、导航行高、底部操作和滚动能力稳定。

## Scope

侧栏品牌、导航与底部文字动效，以及相关交互、布局和减少动态效果验证。

## Out of Scope

业务功能、全局样式重设计、引入动画依赖。

# Analysis

当前文字依赖 `collapsed` 条件挂载、卸载，因此直接出现或消失。单纯宽度裁切可能截断半个字，也无法明确控制逐字顺序；采用常驻文字、按字素分段与 CSS opacity/visibility 错峰过渡。

文字只改变可见性，不改变布局。入场从左到右、离场从右到左，最长错峰时长限制在侧栏宽度过渡内。无需定时器、动画库或过渡完成事件，快速切换由浏览器直接反向过渡。辅助技术读取完整文本，隐藏的语言选项立即退出键盘焦点顺序。

# Implementation Plan

1. 使用与侧栏宽度配合的逐字过渡，保证文字动画不参与图标和行高定位。
2. 让品牌、导航和底部文字采用一致的动效。
3. 验证逐字顺序、连续切换、减少动态效果及原有布局回归，运行 `task verify`。
4. 更新任务记录、审查差异并提交。

# Implementation Result

## Changes

- 新增 `SidebarLabel` 与局部 CSS，按字素拆分文字，展开时从左往右逐字淡入、收起时从右往左逐字淡出。每字过渡 60ms、错峰间隔最高 32ms、总错峰上限 150ms，避免长英文拖慢动画。
- 品牌、11 个导航项、语言控件及折叠按钮共 16 处文字应用同一动效。侧栏宽度过渡调整为 250ms，文字入场最迟在 245ms 完成。
- 文字始终占位，仅改变透明度和可见性；保持固定图标中心、36px 行高、4px 间隔和底部高度。
- 语言控件保留退出动画需要的 DOM；通过 `inert` 与 `aria-hidden` 立即关闭隐藏选项的交互和可访问性，图标模式的语言切换按钮继续可用。
- 辅助技术读取完整文本，逐字视觉片段不重复朗读；减少动态效果偏好下直接切换终态，无需等待或监听动画完成。
- 未引入新依赖、计时器或动画状态。

## Files Changed

- `frontend/src/components/common/sidebar-label.tsx`、`sidebar-label.css`
- `frontend/src/components/common/app-sidebar.tsx`、`app-sidebar.test.tsx`
- `frontend/src/components/common/language-switcher.tsx`、`language-switcher.test.tsx`
- `docs/plans/CURRENT.md`、本任务记录。

# Verification

## Verification Result

- Chromium / Playwright 逐帧检查：中英文各 16 处文字的透明度顺序正确，确实存在不同字符依次变化的中间帧；收起、展开、55/105ms 快速反向切换全部到达正确终态，无文字残留。
- 检查初始收起页面无文字闪现、收起时切换语言后展开无旧文本残留、减少动态效果下立即达到终态。
- 键盘检查：收起时保留的语言选项不能获得焦点；从最后一个导航项 Tab，依次进入语言按钮和展开按钮。
- 复用上一任务的 16 组布局回归：中英文、明暗主题、800/600/400px 高度及减少动态效果全部通过。14 个图标的横向和纵向位移仍均为 0px，行高与间距恒定；矮窗口可滚动到最后一项，底部操作始终可见。
- 人工检查中英文展开态截图，品牌、导航、语言选项和底部按钮无裁切。
- 初次测试发现旧断言依赖文字卸载，已改为验证文字的可访问性隐藏；初次类型检查发现内联 CSS 自定义属性的上下文类型冲突，已改为显式类型的局部变量。
- 最终 `task verify` PASS：格式、Lint、类型检查、Go 测试、前端 11 文件 / 41 测试及 Windows 生产构建全部通过。
- `git diff --check` PASS。

## Limits

真实动画验证使用 Chromium，未直接录制原生 Wails WebView2；浏览器检查脚本、截图和 JSON 结果保存在本次会话的本地可视化目录中，未向项目提前引入 E2E 依赖。质量门仍输出既有的 `uname` / `tail` 环境提示与 JS 包超过 500kB 的构建警告，最终退出码为 0。

# Documentation

已同步 `docs/plans/CURRENT.md`；产品、架构和技术栈无变更。

# Deviations

None.

# Git

```text
d8ec641e9c1f34ab59b28f1edc75ea2353c916f8 feat(frontend): 为侧栏文字增加逐字收展动画
```

提交 hash 由随后的任务归档提交回填。

# Result

侧栏文字逐字收展，快速切换可恢复到正确终态；图标、行高、底部操作和滚动行为保持稳定。

# Completed

2026-09-26
