# TASK-20261110-004: 构建本地 v2.2.3-beta.6 生产安装包

## Status

In Progress

## Created

2026-11-10 09:45（Asia/Singapore）

## Request

用户要求提交代码后，构建新的 Beta 生产版本供其本地更新，并完成任务 Markdown 更新。

## Goal

将已完成的界面调整（TASK-20261110-003，`d383ccc7`）纳入 `v2.2.3-beta.6`，同步版本与双语发行说明，提交后构建 Windows amd64 生产安装包并记录实际产物。

## Background

- `v2.2.3-beta.5` 已于 2026-11-10 本地构建（TASK-20261110-002），当前源码版本为 `2.2.3-beta.5`。
- 本版本包含：移除仪表盘／用量页／请求页的手动刷新按钮；日志页脚「回到最新」按钮与「自动跟随最新」提示共用固定 32px 行高，消除页面跳动。
- 该改动的全量前端测试（146 文件／893 项，限 4 worker）、Go 全部包测试与生产构建已通过，见 TASK-20261110-003。

## Scope

- 应用／前端／Windows 显示版本由 `2.2.3-beta.5` 提升为 `2.2.3-beta.6`。
- Windows 数字版本保持 `2.2.3.0`，NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明，提交版本源码后执行生产打包。
- 核对生产产物的版本资源、SHA256 和发布目录，更新任务记录及 CURRENT。

## Out of Scope

- 不重跑完整质量门：本任务只做发布版本映射与打包；实现任务的全量前端测试、Go 测试和生产构建已通过。
- 不推送、不打标签、不公开发布、不签名。
- 不安装、不启动或停止用户实例，不修改真实用户数据及客户端配置。

## Existing Behavior

界面调整已提交为 `d383ccc7`，任务归档提交为 `45c75ccf`；当前版本 `2.2.3-beta.5`，`release-assets/v2.2.3-beta.6` 尚不存在。

## Expected Behavior

新安装器包含上述界面调整，用户可从托盘完全退出后本地覆盖更新；版本资源与摘要与实际产物一致。

# Analysis

## Related Modules

- `internal/version/version.go`
- `frontend/package.json`
- `build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/releases/`、`docs/plans/CURRENT.md`

## Dependencies

沿用 Wails v3.1.0、Task 3.53.1、Go、pnpm、NSIS 及现有版本映射／产物校验脚本。

## Risks

- 本地安装器未签名，可能触发未知发布者或 SmartScreen 提示。
- 版本与资源映射不一致或复用旧产物会导致错误交付；使用 `--force` 生产构建并核对原生版本资源与 SHA256。

# Implementation Plan

1. 创建本记录；确认当前版本与发布目录状态。
2. 更新源码版本、发行说明与 CURRENT，运行 `scripts/set-release-version.ps1` 映射 Windows 资源。
3. 审阅并提交版本变更，排除无关文件。
4. 执行 `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.6/bin`。
5. 核对产物版本、大小和摘要，归档任务并提交最终结果。

# Progress

- 已确认当前版本 `2.2.3-beta.5`、`release-assets/v2.2.3-beta.6` 不存在，NSIS 位于 `C:\Program Files (x86)\NSIS`。
