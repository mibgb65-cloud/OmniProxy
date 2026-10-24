# TASK-20261024-010: 桌面行为开关颜色与尺寸统一

## Status

Completed

## Created

2026-10-24 13:42

## Request

用户在「设置与关于 → 通用设置 → 桌面行为」截图反馈：同一张卡片里的两个开关颜色不一致（「开机自启动」开启态是近白色，「启动软件时自动启动网关」开启态是蓝色），看起来很奇怪。

## Goal

同一张卡片内的两个开关在开启态、关闭态、尺寸、圆角、滑块和聚焦环上完全一致，并复用同一个共享组件，避免再次出现两套实现。

## Background

- 「开机自启动」由 `TASK-20260926-008` 引入，当时在 `desktop-behavior.tsx` 内手写 `<input type="checkbox" role="switch">` + 自绘 `<span>` 轨道，开启色使用 `bg-primary`。
- 「启动软件时自动启动网关」由 beta.14 引入，使用共享组件 `components/ui/switch.tsx`（Radix Switch），开启色写死 `bg-blue-600 dark:bg-blue-400`。
- 深色主题下 `--primary` 为 `oklch(0.922 0 0)`（≈ `#e5e5e5`），因此手写开关的开启态渲染为近白色。

## Scope

- `frontend/src/features/settings/desktop-behavior.tsx`：开机自启动开关改用共享 `Switch`，删除手写轨道。
- `frontend/src/features/settings/desktop-behavior.test.tsx`：增加两个开关视觉一致性的回归测试。

## Out of Scope

- 不修改 `components/ui/switch.tsx` 的蓝色取值（供应商卡片 `ProviderSwitch`、更新偏好开关均已使用蓝色滑动开关，蓝色是既有约定）。
- 不合并 `features/providers/provider-switch.tsx` 与共享 `Switch`（跨页面重构，另立任务）。
- 不改动任何后端绑定、配置或文案。

## Existing Behavior

`DesktopBehavior` 卡片内两个开关：`h-6 w-11` / 滑块 `size-5` / 开启态 `bg-primary`（深色近白 `#e5e5e5`）；`h-5 w-9` / 滑块 `size-4` / 开启态蓝色 `#60a5fa`。

## Expected Behavior

两个开关都由共享 `Switch` 渲染：`h-5 w-9`、`border-2`、滑块 `size-4`、开启态蓝色、相同的聚焦环与禁用态。整行仍可点击切换，无障碍名称仍为「开机自启动」。

---

# Analysis

## Related Modules

- `frontend/src/features/settings/desktop-behavior.tsx`
- `frontend/src/components/ui/switch.tsx`（只引用，不修改）

## Root Cause

两个开关由两套独立实现生成，且颜色来源不同：手写实现用语义令牌 `bg-primary`，共享实现写死 `bg-blue-600/dark:bg-blue-400`。深色主题下 `--primary` 是近白色，于是同一卡片出现「白色开关 + 蓝色开关」，尺寸（44×24 对 36×20）也不同。

截图采样确认：`#e5e5e5`（`bg-primary`）对 `#60a5fa`（`dark:bg-blue-400`），两者都是开启态（滑块均在右侧）。

## Fix

删掉手写的 checkbox + 轨道，改用 `<Switch id=... checked=... onCheckedChange=...>`，与 `GatewayStartupSetting` 完全同构；颜色、尺寸、滑块、聚焦环、过渡动画因此只有一处来源。

## Regression Prevention

新增测试断言两个开关节点都带 `data-slot="switch"` 且 `className` 完全相等，任何一方再手写或改样式都会失败。

## Dependencies

N/A

## Risks

低。整行点击依赖 `<label htmlFor>` 关联到 Radix 的 `<button role="switch">`，与既有 `GatewayStartupSetting` 采用同一模式；现有 6 个行为测试覆盖勾选、禁用、失败回滚。

---

# Implementation Plan

1. 创建 Task Record。
2. `desktop-behavior.tsx` 改用共享 `Switch`，删除手写轨道，保留 `autoStartAvailable` 禁用逻辑与错误提示。
3. 补充视觉一致性回归测试。
4. 运行前端测试与 `task verify`，写回结果。

---

# Progress

- 定位根因：同一卡片内两套开关实现，开启态颜色分别取自 `bg-primary` 与 `bg-blue-600/dark:bg-blue-400`；截图像素采样为 `#e5e5e5` 与 `#60a5fa`。
- 改为共享组件后，定向 7 项测试与前端全量 751 项测试通过。

---

# Implementation Result

## Changes

- 「开机自启动」开关由手写 `input[role=switch]` + 自绘轨道改为共享 `Switch`（Radix），与「启动软件时自动启动网关」使用同一个组件、同一套类名。
- 删除 `bg-primary` / `h-6 w-11` / `size-5` 滑块的手写实现，两个开关现在同为 `h-5 w-9`、`bg-blue-600 dark:bg-blue-400`、滑块 `size-4`。
- 保留原有行为：`autoStartAvailable` 为假或保存中时禁用、错误就地显示、失败后回滚到原值、整行可点击。

## Files Changed

- `frontend/src/features/settings/desktop-behavior.tsx`
- `frontend/src/features/settings/desktop-behavior.test.tsx`

## Technical Notes

共享 `Switch` 的开启色仍是写死的 `bg-blue-600 dark:bg-blue-400`，未改成主题令牌——`ProviderSwitch`、更新偏好开关以及本次改动都依赖这抹蓝色作为「开启」的视觉约定。若将来要把开关颜色纳入主题令牌（例如 `--switch-checked`），应作为独立任务统一处理三个实现，避免只改一处又产生新的不一致。

---

# Verification

## Tests

- [x] Unit Tests：`desktop-behavior.test.tsx` 7 项通过（含新增一致性用例）。
- [x] Integration Tests：前端全量 126 文件 / 751 项通过。
- [ ] E2E Tests：N/A（无 E2E 套件；原生窗口视觉由用户确认）。

## Quality Checks

- [x] Format：`prettier --check` 通过。
- [x] Lint：`eslint` 通过。
- [x] Type Check：`tsc --noEmit` 通过。
- [x] Test：通过。
- [x] Build：`task verify` 内 Windows production 构建通过。
- [x] Verify：`task verify` PASS。

## Verification Result

```text
task verify
VERIFY_EXIT=0
Test Files  126 passed (126)
Tests  751 passed (751)
```

首次 `task verify` 运行中 `src/features/accounts/mimo.test.tsx` 有一条 5s 超时（与本任务无关）；单独重跑该文件 8/8 通过，随后重跑完整门禁全部通过，判定为并发负载下的偶发超时。

新增用例：

```text
renders both desktop toggles through the shared switch component
```

---

# Documentation

No documentation changes required.（设计约定仍以代码为准；`docs/plans/CURRENT.md` 看板已追加本任务一行，该文件同时存在其他在途改动，未随本任务提交。）

---

# Deviations

None.

---

# Git

## Commit

```text
43bff26 fix(settings): 桌面行为开关统一使用共享 Switch 组件
```

---

# Result

同一张卡片里的两个开关现在由同一个组件渲染，颜色、尺寸、圆角、滑块和聚焦环完全一致；开关外观只有一处来源，不会再出现白色/蓝色并存。

---

# Completed

2026-10-24 14:00
