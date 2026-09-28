# TASK-20260928-007: 精简提供商页面说明文案

## Status

In Progress

## Created

2026-09-28 16:21

## Request

删除截图标出的四处说明：页面标题下的“九家内置提供商开箱即用…”，内置提供商默认启用的信息框、模型/测试边界提示，以及“内置提供商”标题下的说明。

## Goal

提供商页面直接呈现标题、内置卡片和自定义提供商区域，不显示上述重复说明；开关、模型与测试功能保持可用。

## Existing Behavior

页面标题与内置卡片之间连续显示三段说明，内置分区标题下另有一句说明，占据较多纵向空间。

## Expected Behavior

四处说明消失，页面主标题和“内置提供商”分区标题保留，卡片紧随其后。

---

# Analysis

## Related Modules

- `frontend/src/features/providers/providers-view.tsx`
- `frontend/src/features/providers/provider-catalog.tsx`
- `frontend/src/i18n/locales/{zh,en}.ts`

## Risks

移除说明后，网络测试和模型预览的边界文案不再常驻页面；测试按钮仍需通过结果状态明确可达性，后续账号功能再提供针对性的说明。

---

# Implementation Plan

1. 移除四处展示节点及其无用翻译键，保留页面层级。
2. 更新页面测试，检查说明不再显示且关键操作仍在。
3. 运行 `task verify`，检查实际窗口与差异并提交。

---

# Progress

- 已删除四处说明节点与中英文翻译键，保留页面标题、分区标题和卡片操作。
- 已在运行中的 Wails 窗口核对页面布局，并完成完整验证。

---

# Implementation Result

- 从提供商页面和内置提供商分区删除截图标出的四处重复说明。
- 删除对应的中英文翻译键；页面测试覆盖文案消失及关键操作保留。

---

# Verification

- `pnpm typecheck`、定向页面测试及 `pnpm format:check` 通过。
- `task verify` 通过：格式、Lint、类型检查、Go 测试、前端 15 个测试文件共 69 个测试及生产构建均通过。
- 在实际 Wails 窗口目视检查提供商页面；四处说明不再显示，标题、内置卡片和操作按钮正常呈现。
- `git diff --check` 通过。

---

# Documentation

已更新 `docs/plans/CURRENT.md`；产品与架构文档无需修改。

---

# Deviations

None.

---

# Git

## Commit

待完成。

---

# Result

提供商页面已精简，预设卡片与原有功能保留。
