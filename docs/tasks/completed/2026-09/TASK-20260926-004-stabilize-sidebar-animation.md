# TASK-20260926-004: 修复侧栏收展时图标位移与垂直跳动

## Status

Completed

## Created

2026-09-26 09:49

## Request

左侧操作栏收起时，图标先向右移动再跳回左侧；收起与展开时图标上下间距不一致，导致上下闪动。希望优化修复收展动画。

## Goal

侧栏收展期间图标位置连续、行高和间距保持一致，保留导航、展开按钮与低高度滚动能力。

## Scope

- 检查并修复侧栏布局和动画。
- 验证收起、展开、连续切换及减少动态效果偏好。
- 更新任务记录和当前状态看板。

## Out of Scope

导航内容、业务功能、全局视觉风格及技术栈变更。

# Analysis

## Root Cause

- 收起状态立即应用 `justify-center`，但侧栏仍处于从 224px 到 64px 的宽度过渡，图标先跳到宽栏中心再随宽度向左移动。
- 导航项高度由内容决定：文字行高 20px，图标高 16px；移除文字后每行从 36px 降为 32px，后续条目逐行上移。
- 底部语言控件展开时为 26px 高、收起时为 32px 高，额外改变了导航区可用高度。

## Approach

使用固定左侧图标位置及显式行高，让侧栏宽度独立过渡。比起增加动画库或根据过渡事件切换布局，这种方案无需额外状态，也能自然响应连续反向切换。

导航、品牌与底部图标统一水平中心；语言控件两种状态统一为 32px 高。保留原有裁切和纵向滚动，支持减少动态效果偏好。

# Implementation Plan

1. 定位动画期间布局变化，选择最小修复方案。
2. 统一两种状态的图标定位和纵向尺寸。
3. 执行针对性验证与 `task verify`，审查差异并记录结果。

# Implementation Result

## Changes

- 导航和底部按钮始终左对齐，固定内边距使图标中心始终位于侧栏左侧 32px；品牌标识使用匹配的中心位置。
- 导航项固定为 36px 高、行间距 4px；底部语言控件在两种状态下均为 32px 高。
- 侧栏宽度使用 200ms 缓入缓出过渡，并遵循 `prefers-reduced-motion`；底部按钮只过渡颜色，避免通用按钮的 `transition-all` 影响布局。
- 导航在图标模式下保留可访问名称，收展按钮通过 `aria-expanded` 暴露状态；现有交互测试覆盖这些行为。
- 保留横向裁切和导航纵向滚动，不增加依赖或动画状态。

## Files Changed

- `frontend/src/components/common/app-sidebar.tsx`
- `frontend/src/components/common/language-switcher.tsx`
- `frontend/src/components/common/app-sidebar.test.tsx`
- `docs/plans/CURRENT.md`、本任务记录。

# Verification

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

- `task verify` PASS：Go 测试通过，前端 11 个测试文件 / 41 个测试通过，Windows 生产构建成功。
- Playwright 使用本机已有 Python 包与 Chromium，对运行中的 Vite 前端进行真实布局测量；通过 `requestAnimationFrame` 采样每一帧的 14 个图标中心、11 个导航项行高和间距、滚动区域与底部按钮边界。
- 16 组场景全部通过：中文浅色 1280×800、英文深色 1280×800、中文深色 1024×600、英文浅色 900×400，分别检查收起、展开、60/110/165ms 内连续反向切换及减少动态效果。
- 所有采样中，图标横向和纵向位移均为 0px；每行高度始终为 36px、相邻行顶边间距始终为 40px，导航滚动高度和底部区域高度不变。
- 800px 高度无纵向滚动条；600px 和 400px 高度下可滚动到最后一项，底部收展按钮保持在窗口内；所有场景禁止横向滚动条。
- 图标模式下语言切换和导航跳转通过；截图人工检查中英文、明暗主题的展开和收起布局，文字和底部按钮完整可见。
- `git diff --check` PASS。

## Limits

- 动画几何验证在 Chromium 中执行，未直接录制原生 Wails WebView2 窗口；Windows 构建已验证。
- 构建仍提示现有 JS 包超过 500kB。Task 初始化提示找不到 `uname` / `tail`，但全部质量检查及 Windows 构建成功，进程退出码为 0；这些已有环境/打包提示不属于本次侧栏修复。
- 布局检查使用本机已有 Playwright，未提前向项目引入 E2E 依赖。脚本、截图与 JSON 测量结果保存在本次会话的本地可视化目录中；jsdom 单测不承担真实几何布局断言。

# Documentation

- `docs/plans/CURRENT.md` 同步任务状态。
- 无需变更产品、架构、技术栈或 ADR：只修复已有交互行为。

# Deviations

None.

# Git

```text
2562acfe8e8836f1e6051eefc825892291d379b0 fix(frontend): 修复侧栏收展时图标位移与垂直跳动
```

提交 hash 由随后的任务归档提交回填。

# Result

侧栏收展时图标保持原位，垂直间距一致，快速切换不产生位移或高度跳变；底部操作和低高度导航滚动能力保留。

# Completed

2026-09-26
