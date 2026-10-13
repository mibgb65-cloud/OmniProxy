# TASK-20261009-013: 语言切换仅保留图标

## Status

Completed

## Created

2026-10-09

## Request

不需要“语言”两个文字，只保留图标。

## Goal

标题栏语言入口仅显示图标，与相邻主题按钮尺寸一致，保留现有菜单及可访问名称。

## Implementation Plan

移除按钮可见文字，改为居中的 44px 方形按钮；复用已有语言／标题栏验证，检查实际窗口并更新文档。

## Implementation Result

移除语言触发按钮的可见文字，仅保留 Lucide 语言图标，按钮使用与主题按钮一致的 44px 方形居中布局。保留 aria-label、现有语言单选菜单与持久化行为。同步外观说明和状态看板。

## Verification

- 完整 `task verify BIN_DIR=<临时 QA 目录>/quality-bin` 通过：格式、Go 静态检查、ESLint、TypeScript、Go 测试、47 个前端文件共 241 项测试和 Windows 生产构建。
- 浏览器复用已有验证流程检查 12 种语言／主题／字号组合，确认触发按钮可见文字为空、浮层布局无溢出、选中标记及键盘／取消／保存行为正常，无页面错误。
- 原生 Wails UIAutomation 确认按钮可访问名称与展开菜单仍可用，随后 Escape 关闭，未改变用户语言。开发窗口通过 HMR 更新。
- `git diff --check` 通过。纯显示调整，复用既有测试，无新增测试或依赖。

## Git

保留前序未提交改动，不混合提交。

## Result

已完成纯图标语言入口，验证通过。

## Completed

2026-10-09
