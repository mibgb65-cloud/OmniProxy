# TASK-20261025-009: 一体式通顶侧边栏与单一品牌标识

## Status

Completed

## Created

2026-10-25 12:00（Asia/Singapore）

## Request

用户指出左侧上部分的双图标重复问题（截图红框：自绘标题栏左侧 `OmniProxy` 小图标＋标题文字，正下方侧栏顶部又有同一图标＋`OmniProxy`），选择方案：

> 一体式通顶侧边栏（Linear / Raycast 风格）：取消横跨全屏的顶部标题栏左侧小 Logo，让左侧边栏直接通顶（占满 100% 高度），全窗口只在左上角保留一个主品牌标识，并将原本底部的「折叠」按钮提炼到品牌栏右侧。

## Goal

窗口只在左上角出现一次品牌标识（图标＋文字），侧栏占满窗口高度，标题栏只覆盖内容区，折叠／展开入口位于品牌栏内。

## Background

- 修改前根布局 `frontend/src/app/routes/__root.tsx` 为「整宽标题栏 + 下方 [侧栏 | 内容] 两列」，标题栏左侧品牌与侧栏顶部品牌纵向紧邻，形成重复。
- 侧栏顶部品牌行 `h-14`（56px），标题栏 `h-11`（44px），两者高度不同，层次不一致。
- 折叠按钮位于侧栏底部，与「设置与关于」同组，与品牌入口分离。

## Scope

- 根布局改为「侧栏通顶 + 右侧 [标题栏 / 内容] 两行」。
- 侧栏品牌栏：`h-11`（与标题栏同高）内放主图标、`OmniProxy` 文字与收展按钮。
- 收起态品牌栏整行即展开按钮。
- 标题栏移除左侧图标与标题文字，只保留可拖动区与语言／主题／窗口控制。
- 更新侧栏、标题栏测试与外观文档。

## Out of Scope

- 不改 64/224px 收展宽度、导航项、网关状态卡片、逐字动画与滚动条实现。
- 不改 Wails 窗口配置、`--wails-draggable` 拖动实现与窗口控制行为。
- 不改后端；i18n 词典中的 `window.title` 保留（不再被标题栏使用）。

## Existing Behavior

- 品牌图标与 `OmniProxy` 文字在标题栏和侧栏顶部各出现一次；标题栏文字取 `window.title`，侧栏文字为硬编码 `OmniProxy`。
- 折叠按钮在侧栏底部 `border-t` 容器内，位于「设置与关于」上方，带文字标签。
- 标题栏左侧区域（图标＋标题）为可拖动区，双击最大化。
- 侧栏不参与窗口拖动。

## Expected Behavior

- 全窗口仅左上角一处品牌（图标＋文字）；标题栏不再有品牌元素。
- 侧栏从窗口顶端贯穿到底端，品牌栏与标题栏同为 44px，横向分隔线在同一 y 坐标连续。
- 品牌栏右侧为折叠按钮（展开态）；收起态品牌栏整行是展开按钮，图标悬停／键盘聚焦切换为展开图标并带提示。
- 标题栏剩余空间仍为拖动区并支持双击最大化；语言、主题与窗口控制行为不变。
- 侧栏底部仅剩「设置与关于」。

---

# Analysis

## Related Modules

- `frontend/src/app/routes/__root.tsx`（根布局）
- `frontend/src/components/common/app-sidebar.tsx`（品牌栏、收展按钮）
- `frontend/src/components/common/app-titlebar.tsx`（移除品牌）
- `frontend/src/components/common/app-sidebar.test.tsx`、`app-titlebar.test.tsx`
- `docs/product/APPEARANCE.md`、`docs/product/PRD.md`、`docs/plans/CURRENT.md`

## Dependencies

- 复用已有 `SidebarLabel`、`Tooltip`、lucide `PanelLeftClose`／`PanelLeftOpen`；无新依赖。

## Risks

- Low：纯前端布局调整。需保证 44px 对齐、收起态 64px 内按钮可点击、逐字动画与滚动行为不变、现有标题栏双击与拖动行为不受影响。

---

# Implementation Plan

1. `__root.tsx`：外层横向 flex，`AppSidebar` 通顶，右侧列内放 `AppTitlebar` 与主内容 `ScrollRegion`。
2. `app-sidebar.tsx`：品牌栏改为 `h-11`；展开态 `[图标][OmniProxy][折叠按钮]`，收起态整行是展开按钮（品牌图标悬停／聚焦切换为 `PanelLeftOpen`）；移除底部折叠按钮。
3. `app-titlebar.tsx`：删除图标与 `window.title` 文字，保留撑满高度的拖动区（含双击最大化）。
4. 测试：标题栏测试改用稳定拖动区定位，并新增「标题栏不含品牌」回归；侧栏测试新增「品牌唯一 + 收展入口在品牌栏」回归。
5. 文档：更新 APPEARANCE、PRD 与 CURRENT。

---

# Progress

- 布局、品牌栏、标题栏、测试与文档改动完成；前端定向测试与全量测试通过。
- 隔离数据目录启动 Windows production 构建，原生窗口展开／收起／悬停三种状态截图核对通过。

---

# Implementation Result

## Changes

- 根布局改为「侧栏通顶 + 右侧 [标题栏 / 内容]」：侧栏占据窗口左侧整列高度，标题栏只覆盖内容区宽度。
- 侧栏品牌栏改为 `h-11`（与标题栏同高、`border-b` 连续）：展开态依次为主图标、`OmniProxy` 文字、右对齐折叠按钮；收起态整行即展开按钮，默认显示主图标，悬停／键盘聚焦交叉淡入 `PanelLeftOpen` 并显示「展开」提示。
- 移除侧栏底部折叠按钮，底部只保留「设置与关于」。
- 标题栏移除左侧图标与 `window.title` 文字，只保留可拖动区（双击最大化）和语言／主题／最小化／最大化／关闭按钮。

## Files Changed

- `frontend/src/app/routes/__root.tsx`
- `frontend/src/components/common/app-sidebar.tsx`
- `frontend/src/components/common/app-titlebar.tsx`
- `frontend/src/components/common/app-sidebar.test.tsx`
- `frontend/src/components/common/app-titlebar.test.tsx`
- `docs/product/APPEARANCE.md`
- `docs/product/PRD.md`
- `docs/plans/CURRENT.md`

## Technical Notes

- 品牌栏与标题栏均为 `h-11` 且各自 `border-b`，两侧分隔线在同一 y 坐标连续，通顶侧栏与标题栏之间没有高度错位。
- 收起态使用整行按钮（`h-11 w-full`）而不是「图标 + 小按钮」：64px 内两个元素会互相挤压，整行按钮同时提供更大的点击／聚焦区域。
- 品牌文字仍走 `SidebarLabel` 的逐字动画；收起后品牌文字不再挂载（不再保留占位），避免文字宽度把按钮挤出 64px。
- 新增稳定测试钩子 `data-testid="sidebar-brand"`、`data-testid="titlebar-drag-region"`，替代此前依赖「标题栏内 OmniProxy 文字」的定位方式。
- 标题栏不再承载品牌，但保留 `window-titlebar`（`--wails-draggable: drag`）与双击最大化；侧栏品牌栏不参与窗口拖动。

---

# Verification

## Tests

- [x] Unit Tests（`app-sidebar.test.tsx`、`app-titlebar.test.tsx`）
- [ ] Integration Tests（不适用：纯布局组件）
- [ ] E2E Tests（不适用；以原生窗口截图核对替代）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

- PASS：`task --force verify BIN_DIR=tmp/ui-verify-20261025/verify-bin` exit 0
  （日志 `tmp/ui-verify-20261025-verify2.log`；格式检查、`go vet`／`staticcheck`、ESLint、`tsc --noEmit`、
  Go 全部包测试、前端 **129 文件／782 项**测试、Windows production 构建输出
  `tmp/ui-verify-20261025/verify-bin/omniproxy.exe`）。
  注：首次直跑时控制台因 Taskfile 的格式检查片段打印 `uname`／`tail` 缺失的 stderr 被 PowerShell 判为
  NativeCommandError 而报 exit 1；各阶段实际全部通过，重跑并把 stderr 单独重定向后 exit 0（该项为既有环境问题）。
- PASS：定向测试 `app-sidebar.test.tsx` + `app-titlebar.test.tsx` 共 2 文件／14 项（新增 2 项回归：
  标题栏不含品牌、品牌唯一且收展入口在品牌栏）。
- PASS：隔离数据目录（`OMNIPROXY_DATA_DIR=tmp/ui-verify-20261025/data`）启动 production 构建，
  1200×800 原生窗口截图核对三种状态：
  - 展开：品牌图标＋`OmniProxy` 只在左上角出现一次，品牌栏右侧为折叠按钮，标题栏只覆盖内容区；
  - 收起：64px 轨道内品牌图标仍在左上角，整行可点，底部只剩「设置与关于」；
  - 悬停：品牌图标换成展开图标并显示「展开」提示。
  证据：`tmp/ui-verify-20261025/shell-expanded.png`、`shell-collapsed.png`、`shell-collapsed-hover.png`。
  说明：该隔离实例的网关端口 3000 被本机生产实例占用，卡片显示「代理端口不可用」，与布局无关。
- 未执行：浅色主题与 80%／140% 字号的逐像素核对、真实鼠标拖动窗口（拖动实现未改）。

---

# Documentation

- [x] APPEARANCE（一体式通顶侧栏、单一品牌、收展入口位置、拖动区）
- [x] PRD（侧栏底部「折叠」入口描述改为品牌栏收展 + 底部设置入口）
- [ ] Architecture
- [ ] Tech Stack
- [ ] Coding Rules
- [ ] Testing
- [ ] Build
- [ ] ADR（无长期架构决策变更）
- [x] CURRENT

---

# Deviations

- 原计划：收起态「整行展开按钮（图标悬停／聚焦切换为 PanelLeftOpen）」；实际：一致，额外把品牌文字改为收起后不挂载
  （原侧栏用 `SidebarLabel` 保持占位，宽度会把按钮挤出 64px）。
- 原计划：测试改用稳定定位；实际：新增两个 `data-testid`（`sidebar-brand`、`titlebar-drag-region`），
  替代标题栏测试原先依赖「`OmniProxy` 文字的父元素」的脆弱定位。
- 计划外清理：`app-sidebar.tsx` 中不再使用的 `Button` 导入。

---

# Git

## Commit

```text
7ae0f22 feat(frontend): 侧栏通顶并合并为单一品牌标识
```

---

# Result

窗口只在左上角保留一处品牌标识，侧栏通顶占满窗口高度，标题栏仅覆盖内容区；收展按钮移入品牌栏右侧，收起态整行即展开按钮，底部只剩「设置与关于」。完整质量门与原生窗口截图核对通过。

---

# Completed

2026-10-25 12:20
