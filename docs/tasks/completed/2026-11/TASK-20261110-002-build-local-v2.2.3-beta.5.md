# TASK-20261110-002: 构建本地 v2.2.3-beta.5 生产安装包

## Status

Completed

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
- 版本与文档先提交为 `c6e1d167`，再执行生产打包。
- `task --force package` exit 0，完成绑定／图标生成、前端生产构建、Go `production` 编译和 NSIS 打包。
- Windows 原生版本资源、发布目录及 SHA256 核对通过；未签名，未启动、安装或公开发布。

# Implementation Result

## Changes

- 应用、前端和 Windows 显示版本统一为 `2.2.3-beta.5`；Windows 数字资源 `2.2.3.0`、NSIS 基础版本 `2.2.3` 保持不变。
- 新增双语发行说明，说明三个 API Key 接入修复、固定按量账号路由、本地覆盖更新及既有质量门限制。
- 包含修复提交 `a175b8ca`；没有新增迁移、依赖、业务逻辑或生成绑定差异。
- 已保存完整构建日志至 `tmp/local-v2.2.3-beta.5-package.log`；前置开发 Task 构建的未完成问题本次通过强制生产打包完成，未修改 Taskfile。

## Build and Artifacts

构建命令（当前进程 PATH 临时加入 NSIS）：

```text
task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.5/bin
```

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.5/OmniProxy-Setup-v2.2.3-beta.5-windows-amd64.exe` | 20,597,316 | `b25030280c702cdf03db8c7a3bb21f40ee3b612996be56212f9e087106dfd483` |
| `tmp/local-v2.2.3-beta.5/bin/omniproxy.exe` | 23,641,088 | `bde79ec99f7dff14af5157e6536d98120d6c68b53dd55a01fe08a277d5e30af5` |

安装器摘要侧车与安装器同目录，后缀 `.exe.sha256`。原始 NSIS 安装器位于 `bin/OmniProxy-amd64-installer.exe`。产物目录不入 Git。

# Verification

## Verification Result

- 版本映射脚本：通过。
- `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.5/bin`：exit 0。
- 前端生产构建（含 TypeScript 编译）：通过。
- Go 构建信息：`-tags=production`、`GOOS=windows`、`GOARCH=amd64`、`CGO_ENABLED=0`、Wails `v3.1.0`。
- `scripts/test-windows-metadata.ps1`：应用与安装器原生版本资源、数字版本及产品身份核对通过。
- `scripts/verify-release-assets.ps1`：发布目录只包含安装器及 SHA256 侧车，摘要匹配。
- 应用与安装器 Authenticode 状态均为 `NotSigned`。
- `git diff --check`：通过，打包没有产生额外受跟踪文件差异。
- 按用户要求未重跑单测、集成测试、真实 API 调用或 `task verify`，不将前置质量门未完成描述为本轮已通过。

# Documentation

- 新增 `docs/releases/v2.2.3-beta.5.md`。
- 更新 `docs/plans/CURRENT.md` 的本地构建状态。
- 更新 TASK-20261110-001 的交付后续。
- 本记录已填写实际构建结果并归档。

# Git

版本提交：`c6e1d167a3b52f36ccc2cca25e6cfca763154abd`（`chore(release): 准备本地 v2.2.3-beta.5 生产版本`）。

代码修复提交：`a175b8ca4693bc60f99791cd651b4f3c16120d3c`。

先提交版本再打包，本记录与 CURRENT 的最终归档单独提交。无关未跟踪的 `TASK-20261029-010-subq-provider.md` 保持原样，未纳入本次提交。未打标签、未推送、未公开发布。

# Result

`v2.2.3-beta.5` Windows amd64 本地 Beta 生产安装器已生成，包含 OpenAI API Key Codex 接入修复，版本资源与摘要核对通过。安装器未签名、未自动安装、未启动；用户可先从托盘完全退出并备份数据，再手动覆盖更新。真实 Codex 调用及安装后行为由用户本地验收。

## Completed

2026-11-10（Asia/Singapore）
