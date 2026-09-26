# TASK-20260926-002: 修复侧边栏展开时短暂出现滚动条

## Status

In Progress

## Created

2026-09-26 09:31

## Request

“左侧的操作列在从折叠态展开时会出现右侧的上下滚动条和下侧的左右滚动条，你看看是什么情况？”

## Goal

折叠侧边栏展开期间不再出现横向或不必要的纵向滚动条；窗口空间不足时导航项仍可纵向滚动。

## Existing Behavior

- 侧边栏宽度从 `w-16` 动画到 `w-56`，标签和底部语言控件在动画开始时立刻显示。
- 导航容器仅设置 `overflow-y-auto`，横向溢出在动画中也会形成滚动区；文字换行会增加导航高度。

## Expected Behavior

- 展开动画中标签不换行，超出当前宽度的内容被侧边栏裁切。
- 导航区域无横向滚动条；正常高度下不出现纵向滚动条，窗口过矮时仍能滚动导航。

---

# Analysis

## Related Modules

- `frontend/src/components/common/app-sidebar.tsx`
- `frontend/src/components/common/language-switcher.tsx`
- `frontend/src/components/common/app-sidebar.test.tsx`

## Root Cause

宽度动画与内容立即展开不同步，导航区默认横向溢出行为及可换行内容共同造成临时滚动条。

## Risks

- 裁切必须只影响横向动画过渡，不能让矮窗口下的导航项不可访问。

---

# Implementation Plan

1. 约束侧边栏宽度和裁切；让导航链接与底部控件在宽度过渡时保持单行。
2. 导航区仅在纵向空间不足时滚动，禁止横向滚动条。
3. 用实际窗口折叠/展开及矮窗口验证；运行 `task verify` 并提交。

---

# Progress

- 已复现折叠/展开流程，并定位为宽度过渡期间的内容溢出与换行。
- 添加侧栏裁切、导航横向隐藏和单行约束；实际窗口验证常规高度无临时滚动条，最小高度保留纵向滚动。
- `task verify` 全部通过。

---

# Implementation Result

## Changes

- 侧边栏固定自身宽度并裁切宽度动画中的超出内容。
- 标题、导航链接和底部控件保持单行，避免过渡期间内容换行撑高导航区。
- 导航容器显式隐藏横向溢出并允许纵向溢出滚动；增加 `min-h-0`，确保矮窗口内导航区可以收缩。

## Files Changed

- `frontend/src/components/common/app-sidebar.tsx`
- `docs/plans/CURRENT.md`、本任务记录

## Technical Notes

- 使用现有 CSS 能力修复，不引入依赖或改变侧边栏展开动画。

---

# Verification

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

- `task verify` PASS：Go 测试全通过；前端 11 个测试文件、41 个测试全通过；Windows 构建成功。
- 实际开发窗口折叠/展开截图验证：动画开始、过渡中和完成后没有横向滚动条，也未在正常窗口高度出现纵向滚动条。
- 将窗口缩至最低高度 600px：导航区可纵向滚动，底部操作区仍显示；窗口已恢复原尺寸。
- `git diff --check` PASS。
- 本问题依赖真实浏览器布局；现有 jsdom 测试不计算滚动尺寸，因此采用实际 Wails 窗口验证，未添加仅断言 CSS 类名的测试。

---

# Documentation

- [x] `docs/plans/CURRENT.md`：同步任务状态。
- PRD、架构、技术栈文档无需更新：仅修复现有界面的过渡布局。

---

# Deviations

None.

---

# Git

## Commit

待提交并回填。

---

# Result

侧边栏展开时的临时滚动条已消除，矮窗口下必要的纵向滚动仍可用。

---

# Completed

待填写。
