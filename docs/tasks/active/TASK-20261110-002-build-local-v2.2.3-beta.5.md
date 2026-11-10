# TASK-20261110-002: 构建本地 v2.2.3-beta.5 生产安装包

## Status

In Progress

## Created

2026-11-10（Asia/Singapore）

## Request

用户要求无需再测试，提交代码，然后构建新的 Beta 生产版本供本地更新，并完成任务 Markdown 更新。

## Goal

将已提交的 OpenAI API Key Codex 接入修复（TASK-20261110-001，`a175b8ca`）纳入 `v2.2.3-beta.5`，同步版本和发行说明，提交后构建 Windows amd64 生产安装包并记录实际产物。

## Scope

- 应用／前端／Windows 显示版本由 `2.2.3-beta.4` 提升为 `2.2.3-beta.5`。
- Windows 数字版本保持 `2.2.3.0`，NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明，提交版本源码后执行生产打包。
- 核对生产产物的版本资源、SHA256 和发布目录，更新任务记录及 CURRENT。

## Out of Scope

- 按用户明确要求，不再运行单测、集成测试、真实 API 调用或 `task verify`。
- 不推送、不打标签、不公开发布、不签名。
- 不安装、不启动或停止用户实例，不修改真实用户数据及 Codex 配置。

## Existing Behavior

修复代码已提交为 `a175b8ca`，任务归档提交为 `9ff69d1e`。当前版本为 `2.2.3-beta.4`，`v2.2.3-beta.5` 本地发布目录尚未创建。

## Expected Behavior

新安装器包含 API Key 默认上游地址、后台目录同步、新建账号同步通知修复，用户可自行退出托盘后覆盖更新。

# Analysis

## Related Modules

- `internal/version/version.go`
- `frontend/package.json`
- `build/windows/info.json`
- `docs/releases/`
- `docs/plans/CURRENT.md`

## Dependencies

沿用 Wails v3.1.0、Task、Go、pnpm、NSIS 及现有版本映射／产物校验脚本。

## Risks

- 前置任务的完整质量门未通过／未完成，不能把本次打包描述为重新测试通过。
- 未签名的本地安装器可能触发未知发布者／SmartScreen 提示。
- 版本和资源不一致、复用旧产物会导致错误交付，使用强制生产构建并核对元数据和摘要。

# Implementation Plan

1. 创建本记录；确认版本与打包规范。
2. 更新版本、发行说明、CURRENT 和前置任务交付说明，置为 In Progress。
3. 运行版本映射脚本，审阅并提交本次变更，排除无关文件。
4. 执行 `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.5/bin`。
5. 核对产物版本、大小和摘要，归档任务并提交最终结果。

# Progress

- 修复代码已提交，工作区仅有一份原有无关未跟踪任务文件，保持原样。
- 用户已明确授权生产 Beta 构建并要求不再测试。
- 已选择下一本地版本 `v2.2.3-beta.5`；沿用 `task --force package`，不调用会强制执行质量门的 `scripts/build-release.ps1`。
- 版本映射脚本核对通过，应用／前端／Windows 显示版本为 `2.2.3-beta.5`，数字版本为 `2.2.3.0`；已确认 NSIS 可用、目标发布目录不存在。

# Implementation Result

待填写。

# Verification

不重跑测试或 `task verify`；只执行打包所需生产编译及产物完整性／版本核对。实际结果待填写。

# Documentation

待填写。

# Git

待填写。

# Result

待填写。
