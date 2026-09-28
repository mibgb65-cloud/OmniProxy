# TASK-20260928-006: 测试结果悬浮提示

## Status

In Progress

## Created

2026-09-28 16:06

## Request

将“地址可达 · 914 ms · HTTP 404”等提供商测试结果移动到“测试”按钮左侧，改成小型悬浮提示，并提供出现和消失动画。

## Goal

测试完成后，结果在按钮左侧以不挤压布局的小提示显示，短暂保留后淡出；再次测试会更新提示。键盘和屏幕阅读器仍能获知结果，低动态效果设置下避免动画。

## Existing Behavior

测试结果作为卡片底部独立的一行文字长期显示，改变卡片高度。

## Expected Behavior

结果浮层锚定在测试按钮左侧，淡入并轻微位移，约四秒后淡出，不改变卡片尺寸。失败与成功状态使用现有文案和颜色。

---

# Analysis

## Related Modules

- `frontend/src/features/providers/provider-card.tsx`
- `frontend/src/features/providers/providers-view.tsx`
- `frontend/src/features/providers/providers-view.test.tsx`

## Risks

- 浮层不得遮挡测试按钮或导致水平滚动；自定义项的编辑/删除按钮也要保持可用。
- 自动消失不能让屏幕阅读器错过结果；需保留 `role=status` 的即时公告。
- 重复点击和组件卸载时要清理计时器，避免旧结果重新出现。

---

# Implementation Plan

1. 将结果移动到测试按钮左侧的绝对定位浮层，使用可逆的透明度/位移过渡和低动态效果覆盖。
2. 为结果增加短暂展示与重复测试重置逻辑，清理卸载计时器。
3. 增加交互回归测试，运行 `task verify`，检查实际窗口与差异后提交。

---

# Progress

- 测试结果已移至按钮左侧的绝对定位浮层；首次出现及自动隐藏分别使用短关键帧动画，低动态效果下关闭动画。
- 单次结果展示约四秒；重复测试、配置编辑/删除和组件卸载会清理旧结果或计时器。
- 定向组件测试、类型检查和 lint 已通过。实际 Wails 窗口中浮层位于按钮左侧，不挤压卡片；五秒后从可访问窗口视图消失。
- `task verify` 全部通过。

---

# Implementation Result

## Changes

- 成功、失败、异常三类测试结果统一放在按钮左侧的小型浮层，不再占用卡片新行；自定义卡片的编辑/删除操作仍可点击。
- 使用 CSS 关键帧完成 180 ms 的淡入/轻微滑入和淡出/滑出。低动态效果偏好下不播放动画。
- 结果保留四秒后进入退出状态；重新测试会清理旧计时器，编辑、删除配置及页面卸载也会清理过期结果。退出后从可访问性树中隐藏。

## Files Changed

- `frontend/src/features/providers/{provider-card.tsx,provider-probe-tooltip.css,providers-view.tsx,providers-view.test.tsx}`
- `frontend/src/features/providers/{provider-catalog.tsx,provider-list.tsx}`
- `docs/plans/CURRENT.md`、本任务记录

## Technical Notes

浮层首次插入 DOM 时就处于 open 状态，因此用关键帧保证首次出现也有动画；退出时保持 DOM 节点只改变状态，待动画结束后视觉上隐藏，避免布局跳动。

---

# Verification

## Tests

- [x] 点击测试后结果出现在按钮前方，四秒后进入隐藏状态并设置 `aria-hidden`。
- [x] 实际 Wails 窗口显示浮层在按钮左侧，五秒后不再可见。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

`task verify` PASS：Go 测试通过；前端 15 个测试文件、69 个测试通过；Windows 生产构建成功。Wails 窗口中结果“地址可达 · 743 ms · HTTP 404”位于测试按钮左侧且不改变卡片高度，五秒后可访问窗口视图中该提示数为 0。`git diff --check` PASS。

---

# Documentation

已更新 CURRENT 状态看板。产品与架构行为未改变，无需修改其他文档。

---

# Deviations

初次渲染需要出现动画，因此实际采用 CSS 关键帧而非仅依赖首次挂载时可能不触发的 CSS transition。

---

# Git

## Commit

待完成。

---

# Result

测试结果以按钮左侧的短时悬浮提示呈现，包含进入和退出动画，不改变卡片布局。
