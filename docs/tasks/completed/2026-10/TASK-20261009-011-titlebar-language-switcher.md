# TASK-20261009-011: 语言切换移至标题栏

## Status

Completed

## Created

2026-10-09

## Request

把侧栏底部的语言切换按钮放到亮暗色模式按钮左侧。

## Goal

在标题栏主题切换按钮左侧显示语言选择，保留中英文切换和偏好持久化，侧栏不再重复显示入口。

## Analysis

标题栏主题按钮位于 `app-titlebar.tsx`；语言选择组件已接入 Zustand 和 i18n。移动后无需依赖侧栏收起态，可以精简旧的侧栏专用布局。标题栏已有按钮不可拖动规则，语言选择复用它，保留主标题双击最大化。

## Implementation Plan

1. 精简语言选择布局，移入主题按钮左侧，移除侧栏入口。
2. 调整已有导航／标题栏验证，检查切换、持久化、按钮顺序及窗口交互。
3. 执行质量检查，确认原生窗口更新，记录结果并同步状态看板。

## Implementation Result

语言选择组件移到标题栏亮暗模式按钮左侧，移除侧栏原入口；精简侧栏专用收起态布局，保留图标、标签、中文／English 分段选择、选中语义和偏好持久化，并补充键盘焦点样式。标题栏复用已有按钮不可拖动规则。调整既有语言／侧栏测试，增加标题栏集成验证。

## Verification

- `task verify BIN_DIR=<临时 QA 目录>/quality-bin` 通过：格式、Go 静态检查、ESLint、TypeScript、Go 测试、47 个前端文件共 240 项测试及 Windows 生产构建。
- 浏览器验证 12 种中英文／亮暗主题／80、100、140 字号组合（应用最小宽度 960）：语言按钮位于主题按钮左侧，标题栏无溢出；键盘选择、持久化、按钮不可拖动及双击语言按钮不触发最大化均通过，无页面错误。
- 原生 Wails 标题栏截图确认开发窗口已通过 HMR 更新，开发进程持续运行。截图及验证 JSON 位于临时目录 `omniproxy-titlebar-language-qa-20261009`。
- `git diff --check` 通过。保留已有构建包体及 Task 环境探测提示，没有新增依赖或后端行为。

## Documentation

已同步 `docs/product/APPEARANCE.md` 和 `docs/plans/CURRENT.md`。

## Deviations

None.

## Git

工作区含前序未提交改动，保留并不混合提交。

## Result

已完成语言切换入口迁移，验证通过。

## Completed

2026-10-09
