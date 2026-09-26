# TASK-20260926-007: 设置页参考稿布局

## Status

In Progress

## Created

2026-09-26 10:35

## Request

用户提供 `D:/googledown/gemini-code-1790389787885.html`，希望把当前设置页面的前端优化成参考稿的样式。

## Goal

让设置页的信息层级、占用图、路径列表、维护操作和迁移表单接近参考稿，同时保持现有真实数据、双主题、多语言与功能行为。

## Existing Behavior

占用以环形图呈现，路径为简单文字行，迁移表单为普通横向表单。整体间距和分组与参考稿差异较大。

## Expected Behavior

- 顶部为存储设置标题和说明；概览卡片提供总占用强调块、横向分段占用条、四类图例。
- 路径以独立行显示，可复制真实路径；维护操作位于概览卡片底部。
- 迁移表单使用独立卡片、浏览与保存按钮及清晰提示；不改变“退出迁移、重启生效”的后端语义。
- 浅色和深色主题、窄窗口、键盘操作与中英文案可用。

# Analysis

## Related Modules

- `frontend/src/app/routes/settings.tsx`
- `frontend/src/features/settings/storage-overview.tsx`、`storage-usage-chart.tsx`、`data-dir-form.tsx`
- `frontend/src/i18n/locales`、对应组件测试

## Dependencies

使用已有 Tailwind、Lucide、Wails Clipboard API，不新增依赖。

## Risks

参考稿中的 45 KB 和“正常状态”是静态示意，不能当成真实后端数据。参考稿的迁移描述与现有实现时序不同，需保留真实时序。

# Implementation Plan

1. 改造概览组件为参考稿的总量块、横向分段条和图例，并调整卡片标题、路径行和操作区。
2. 调整迁移卡片布局、信息提示和文案，路径复制提供可见成功/失败反馈。
3. 更新交互测试与中英文案，运行 `task verify`，审查最终差异。

# Progress

- 已核对 HTML 的视觉参考与现有数据接口：占用图改为真实比例的横向分段条；参考稿中没有真实状态来源的状态标记不显示。
- 使用构建后的 CSS 和真实 React 组件生成深色、浅色静态预览，修正图标按钮的原生绘制和浅色提示区配色。

# Implementation Result

## Changes

- 设置页按参考稿重排：总占用强调块、四色横向分段条、四列图例、独立路径行、底部维护操作区和迁移提示卡片。
- 路径行使用 Wails Clipboard API 复制真实路径，并提供成功/失败反馈；文件夹选择失败也在表单中显示。
- 调整中英文案与按钮视觉，沿用后端现有清理、退出迁移和重启生效行为；不显示无数据来源的“正常状态”。
- 通过 `appearance-none` 避免原生按钮背景覆盖深色主题。

## Files Changed

- `frontend/src/app/routes/settings.tsx`
- `frontend/src/features/settings/storage-overview.tsx`、`storage-usage-chart.tsx`、`storage-path-list.tsx`、`data-dir-form.tsx`
- `frontend/src/features/settings/storage-overview.test.tsx`、`data-dir-form.test.tsx`
- `frontend/src/i18n/locales/zh.ts`、`en.ts`

## Technical Notes

参考 HTML 仅作为视觉稿；图表数值取自 `StorageInfo`，预览截图中的 45 KB 为测试示意数据，不写入产品代码。参考稿的目录打开图标需要新增系统集成，本次提供可用的复制操作，没有放置无行为按钮。

# Verification

## Tests

- [x] 前端 46 个测试通过，覆盖占用分布、路径复制成功/失败、刷新、清理、迁移表单及目录选择失败。
- [x] Go 测试通过（本任务未修改后端）。

## Quality Checks

- [x] `task verify`：格式、Go vet、staticcheck、ESLint、TypeScript、测试和 Windows 生产构建全部通过。
- [x] 深色、浅色静态预览人工核对。
- [x] `git diff --check`。

## Verification Result

2026-09-26：`task verify` PASS。Vite 仍提示既有 500 KB 包体积提醒，本任务未新增依赖。

# Documentation

- [x] Task Record 记录设计参考、实现与验证。
- [x] 完成时更新 `docs/plans/CURRENT.md`。
- PRD、架构与技术栈无变化。

# Deviations

参考稿的固定容量和“正常状态”为静态演示，最终界面使用真实容量，不显示虚构健康状态；迁移文案依当前后端“退出迁移、下次启动生效”编写。

# Git

## Commit

待填写。

# Result

设置页已按参考稿建立清晰的存储概览、路径列表和迁移卡片，现有功能保持可用。
