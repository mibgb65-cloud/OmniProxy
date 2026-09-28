# TASK-20260928-001: 修复主界面滚动布局与滚动条交互

## Status

In Progress

## Created

2026-09-28 10:06

## Request

修复顶部操作条随页面滚动、左侧操作列与右侧内容一起滚动、主界面出现横向滚动条的问题。所有页面的滚动条保留占位，默认和滚动时隐藏；鼠标移到滚动条区域时渐显，移出后渐隐。

## Goal

外层应用视口固定；侧栏与主内容各自滚动；仅在各自滚动区域需要时提供可交互的悬停滚动条，且主界面没有横向滚动。

## Existing Behavior

截图显示顶部标题栏和侧栏顶部随文档滚动离开视口，窗口底部出现横向滚动条。主内容和侧栏使用原生纵向滚动条，滚动时始终可见。

## Expected Behavior

标题栏始终位于顶部；侧栏滚动不影响主内容，主内容滚动不影响侧栏；文档无横向或纵向滚动条；滚动条固定占位且只在指针悬停其区域时淡入。

---

# Analysis

## Related Modules

- `frontend/src/app/routes/__root.tsx`
- `frontend/src/components/common/app-sidebar.tsx`
- `frontend/src/styles.css`

## Root Cause

根布局使用 `h-screen w-screen`，但 `html`、`body` 和 `#root` 未显式限制文档级滚动；截图中的标题栏、侧栏一起上移和窗口级横向滚动条，说明外层文档成为了滚动容器。`w-screen` 还会按视口宽度计算，在有窗口级纵向滚动条时进一步造成横向溢出。主内容和侧栏分别直接展示原生滚动条，没有统一的悬停交互。

## Risks

隐藏原生滚动条后必须保留滚轮、触控板、键盘和拖拽滚动能力；深色模式及低动态效果设置也应可用。

---

# Implementation Plan

1. 约束文档和根布局的尺寸及溢出，保持标题栏、侧栏和主内容的独立布局。
2. 为两处纵向滚动容器增加共用的固定占位滚动条，悬停轨道时淡入，离开时淡出。
3. 用布局交互检查和 `task verify` 验证，更新任务记录与当前任务看板。

---

# Progress

- 已检查截图和根布局：需要让文档不可滚动，并把纵向滚动限定在主内容与导航区。
- 已实现独立滚动区域和固定占位滚动条，并增加指针拖拽及键盘滚动支持。
- Headless Chrome 在 1200px 宽、500px 高的窗口中确认：文档宽度等于视口宽度；主内容滚动 100px 后标题栏位置为 0，侧栏滚动位置仍为 0；滚动时滑块透明度为 0，鼠标进入轨道后为 0.7，移出后回到 0。
- `task verify` 全部通过。

---

# Implementation Result

## Changes

- 对 `html`、`body`、`#root` 固定尺寸并禁止文档级滚动；根布局改为继承容器宽高，防止视口宽度计算造成横向溢出。
- 主内容和侧栏导航分别使用独立滚动容器；标题栏保持在滚动区域之外。
- 共享滚动组件始终保留 12px 轨道，占位不改变布局；仅鼠标悬停轨道或拖拽时渐显滑块，滚动时保持隐藏；支持点击轨道、拖拽滑块、键盘和滚轮滚动。
- 为 `prefers-reduced-motion` 关闭透明度动画，并为键盘焦点保留可见轮廓。

## Files Changed

- `frontend/src/app/routes/__root.tsx`
- `frontend/src/components/common/app-sidebar.tsx`
- `frontend/src/components/common/scroll-region.tsx`、`scroll-region.css`、`scroll-region.test.tsx`
- `frontend/src/styles.css`、`frontend/src/test/setup.ts`
- `docs/plans/CURRENT.md`、本任务记录

## Technical Notes

滚动条轨道在主内容与侧栏各自的布局内占位。滚动尺寸由 `ResizeObserver` 更新，滑块位置在滚动时同步；原生滚动容器保留滚轮、触控板和键盘行为。

---

# Verification

## Tests

- [x] 滚动区域回归测试：主内容与侧栏位置独立、轨道点击与拖拽可滚动。
- [x] 现有侧栏交互测试。
- [x] 浏览器布局与悬停交互检查。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

`task verify` PASS：Go 测试通过；前端 13 个测试文件、52 个测试通过；Windows 生产构建成功。`git diff --check` PASS。浏览器预览未接入 Wails 服务，截图中设置数据加载报错属于预览环境限制，与滚动布局验证无关。

---

# Documentation

已更新 `docs/plans/CURRENT.md`。PRD、架构、技术栈和构建文档无需修改；这是现有界面的滚动行为修复。

---

# Deviations

None.

---

# Git

## Commit

待提交。

---

# Result

主界面标题栏与侧栏位置固定，主内容和侧栏独立滚动，文档无横向滚动条；两处滚动条仅在鼠标悬停轨道时渐显。
