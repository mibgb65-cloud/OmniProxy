# TASK-20260926-003: 恢复收缩侧栏的展开按钮

## Status

Completed

## Created

2026-09-26 09:40

## Request

“你这操作列在收缩的情况下，直接没有了展开按钮了呀。”

后续补充：“还有一个点，这个收缩时的图标没有居中啊，感觉有些偏左。”

## Goal

收缩状态下，侧栏底部始终显示且可以点击“展开”按钮，所有图标在侧栏内居中，同时保留 TASK-20260926-002 的无临时滚动条修复。

## Existing Behavior

上一任务在底部容器添加 `whitespace-nowrap`。两个 `inline-flex` 按钮因此处于同一行，第二个“展开”按钮被侧栏的 `overflow-hidden` 横向裁掉。

## Expected Behavior

底部语言按钮和展开/折叠按钮始终纵向排列；收缩态顶部标识、导航图标、底部图标均居中；两种侧栏状态及最小窗口高度下均可见可操作。

---

# Analysis

## Root Cause

- 底部 `space-y-2` 不改变行内元素的排版流；新增 `whitespace-nowrap` 禁止两个 `inline-flex` 按钮换行。
- 实际窗口 UI Automation 显示“语言”和“展开”按钮同处一个 y 坐标，后者位于 64px 侧栏之外。
- 导航链接与顶部标识仍沿用展开态的左内边距，收缩态图标中心偏左约 4px；展开按钮同样沿用左对齐。

## Related Modules

- `frontend/src/components/common/app-sidebar.tsx`
- `frontend/src/components/common/app-sidebar.test.tsx`

## Risk

底部布局调整不能重新引入展开动画的滚动条；导航区在矮窗口中仍应可纵向滚动。

---

# Implementation Plan

1. 将底部操作区改为明确的纵向 Flex 排列，保留过渡时的横向裁切。
2. 收缩态单独设置顶部标识、导航链接和展开按钮的水平居中样式。
3. 用真实窗口 UI Automation 与截图检查两个底部按钮的纵向坐标、可见范围、点击展开行为及各图标居中；现有组件测试继续覆盖按钮调用。
4. 验证折叠/展开及最小高度下无回归；运行 `task verify`，提交并更新记录。

---

# Progress

- 已用实际窗口的 UI Automation 确认：收缩态两个底部按钮同处一行，“展开”按钮被横向裁切。
- 已改为纵向排列并在实际窗口验证展开按钮可见、可点击；用户补充要求统一居中收缩态图标，继续实施。
- 收缩态顶部、导航和底部图标已居中；实际窗口普通高度与 600px 最小高度均通过可见范围检查。
- `task verify` 全部通过。

---

# Implementation Result

## Changes

- 底部容器使用纵向 Flex 布局与固定间距，语言按钮和展开/折叠按钮不再被 `whitespace-nowrap` 排到同一行。
- 收缩态顶部标识、导航链接和展开按钮单独使用水平居中样式；展开态继续沿用原有左对齐。
- 保留侧栏横向裁切，未撤销上次的滚动条修复。

## Files Changed

- `frontend/src/components/common/app-sidebar.tsx`
- `docs/plans/CURRENT.md`、本任务记录

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
- 实际 Wails 窗口 UI Automation：收缩态语言按钮与展开按钮纵向排列，展开按钮完整位于 64px 侧栏与窗口边界内，点击可展开。
- 将窗口缩到最低高度 600px：展开按钮仍完整可见；测试后恢复窗口尺寸。
- 收缩态截图检查：顶部标识、导航图标及底部图标沿同一水平中心排列；`git diff --check` PASS。
- 现有 jsdom 组件测试覆盖按钮存在和点击，但不计算真实布局；本回归以 Windows 窗口边界测量和截图验证。

---

# Documentation

- [x] `docs/plans/CURRENT.md`：同步任务状态。
- 其他文档无需更新：仅修复既有侧栏布局。

---

# Deviations

None.

---

# Git

## Commit

```text
9f56ad42ab78b9a31c9b38ad4082471f473e78c9 fix(frontend): restore collapsed sidebar controls and alignment
```

（本任务的 commit hash 由紧随其后的记录提交回填。）

---

# Result

收缩态展开按钮恢复可见并可操作，图标完成居中，上一任务的无临时滚动条修复继续保留。

---

# Completed

2026-09-26 09:47
