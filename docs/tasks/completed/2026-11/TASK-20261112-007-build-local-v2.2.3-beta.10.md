# TASK-20261112-007: 构建本地 v2.2.3-beta.10 生产安装包

## Status

Completed

## Created

2026-11-12 16:45（Asia/Singapore）

## Request

用户在确认 River 工具数量修复后要求：

> 打包吧

即：把 [TASK-20261112-006](TASK-20261112-006-river-tool-count-limit.md)（`4a62be80`）纳入
`v2.2.3-beta.10`，产出本地 Windows amd64 生产安装包，供用户从托盘退出后覆盖更新；完成后
恢复 DSH 的 OpenPencil MCP 并实测 176 个工具可用。

## Goal

- 版本由 `2.2.3-beta.9` 提升为 `2.2.3-beta.10`，Windows 数字版本保持 `2.2.3.0`、NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.10.md`。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。
- 更新任务记录与 `docs/plans/CURRENT.md`。

## Background

- `v2.2.3-beta.9` 已于 2026-11-12 构建（TASK-20261112-004，版本提交 `9f2d825d`），
  当前源码版本仍为 `2.2.3-beta.9`。
- 自 beta.9 起的用户可见改动只有 TASK-20261112-006（`4a62be80` River 工具数量修复、
  `10adc6f5` 提交号回填）。
- 用户当前运行的仍是 beta.9 旧二进制（生产端口 3000），其中 River 适配器仍按 128 个工具拒绝。
- 工作区另有一条未跟踪、与本次无关的旧记录
  `docs/tasks/completed/2026-10/TASK-20261029-010-subq-provider.md`，保持原样，不纳入构建或提交。

## Scope

- `internal/version/version.go`、`frontend/package.json` 与 `build/windows/info.json` 的版本映射。
- `docs/releases/v2.2.3-beta.10.md`。
- 生产打包、产物复制到 `release-assets/v2.2.3-beta.10/`、SHA256 侧车与两项校验脚本。
- 任务记录与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不重跑 `task verify`：实现任务已运行并 exit 0，本任务只做版本映射与打包。
- 不推送、不打标签、不公开发布、不签名。
- 不自动安装、不启动或停止用户实例，不修改真实用户数据。
- 恢复 DSH 的 OpenPencil MCP 属于用户在仓库外的环境操作，不在本次提交范围内；
  待用户装好新版后再执行。

## Existing Behavior

`4a62be80` 已移除 River 的 128 工具计数上限；版本仍为 `2.2.3-beta.9`，
`release-assets/v2.2.3-beta.10` 不存在。

## Expected Behavior

新安装器中的 River 适配器接受 176 个工具；用户覆盖安装后，恢复 OpenPencil MCP 的 DSH
在 `DeepSeek-V4.1-Flash-1017` 上不再收到本地 `400 invalid_request`。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`
- `build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/releases/`、`docs/plans/CURRENT.md`

## Dependencies

沿用 Wails v3.1.0、Task 3.53.1、Go 1.26.2、pnpm 12.6.0、NSIS 及既有版本映射／构建校验脚本。

## Risks

- 本地安装器未签名，可能触发未知发布者或 SmartScreen 提示。
- 用户当前实例仍在运行 beta.9（端口 3000），未完全退出时覆盖安装会被拒绝；发布说明要求先从托盘退出。
- 版本与资源映射不一致或复用旧产物会导致错误交付；用 `--force` 生产构建并核对原生版本资源与 SHA256。
- 工作区存在他人未提交的旧任务记录，提交时只暂存本任务文件。

---

# Implementation Plan

1. 创建本记录；确认当前版本与发布目录状态。
2. 更新 `internal/version/version.go` 与 `frontend/package.json`，运行
   `scripts/set-release-version.ps1 -Version v2.2.3-beta.10`。
3. 新增双语发行说明并审阅，仅提交本任务文件。
4. 执行 `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.10/bin`。
5. 复制安装器到 `release-assets/v2.2.3-beta.10/`，生成 SHA256 侧车，运行
   `scripts/test-windows-metadata.ps1` 与 `scripts/verify-release-assets.ps1`。
6. 填写实际结果，归档任务并提交，更新 CURRENT。

---

# Progress

- 已创建记录，确认当前版本为 `2.2.3-beta.9`、`release-assets/v2.2.3-beta.10` 不存在。
- 版本映射通过：应用／前端／Windows 显示版本 `2.2.3-beta.10`，数字版本 `2.2.3.0`，
  NSIS 基础版本 `2.2.3`（只有 `info.json` 的 `ProductVersion` 变化；脚本写入的 CRLF 已还原为 LF）。
- 版本与发行说明提交为 `2a51e6ab`，随后执行生产打包。
- `task --force package` exit 0（20,683,630 bytes 安装器）；发布资产、SHA256 与原生版本资源校验通过。

---

# Implementation Result

## Changes

- 应用、前端与 Windows 显示版本统一为 `2.2.3-beta.10`；Windows 数字资源 `2.2.3.0`、NSIS 基础版本 `2.2.3` 不变。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.10.md`：说明 River 不再按 128 个工具拒绝、上游核对依据、
  保留的形状校验、其他提供商维持原样，以及安装后如何恢复 DSH 的 OpenPencil MCP。
- 版本中包含 TASK-20261112-006（`4a62be80`）与提交号回填（`10adc6f5`）；无新增迁移、依赖或绑定差异。

## Build and Artifacts

构建命令（当前进程 PATH 临时加入 NSIS）：

```text
task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.10/bin
```

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.10/OmniProxy-Setup-v2.2.3-beta.10-windows-amd64.exe` | 20,683,630 | `1bf3ab2e6164625afd0bd8a0b038387a48fa51ba69963762e1b3aa4b928c088f` |
| `tmp/local-v2.2.3-beta.10/bin/omniproxy.exe` | 23,745,024 | `80fd04160914e62171ad7e782a5518e81611a26be9a11897a1742fb296857e53` |

摘要侧车 `OmniProxy-Setup-v2.2.3-beta.10-windows-amd64.exe.sha256` 与安装器同目录，内容为
`<hash>  <文件名>` 加结尾换行、无 BOM。原始 NSIS 安装器位于 `bin/OmniProxy-amd64-installer.exe`。
产物目录与 `tmp/` 不入 Git。

## Technical Notes

- 版本映射由 `scripts/set-release-version.ps1` 完成；脚本把 `build/windows/info.json` 写成 CRLF，
  已还原为 LF 后再提交，保持仓库无 CRLF。
- 本机 `pwsh` 不在 PATH，脚本用 `powershell -NoProfile -ExecutionPolicy Bypass -File` 执行；
  NSIS 通过临时追加 `C:\Program Files (x86)\NSIS` 到 PATH 找到。
- 打包日志中的 `uname`／`tail` 缺失提示来自 Taskfile 的 Windows 分支探测，构建仍 exit 0。
- 工作区未跟踪的 `TASK-20261029-010-subq-provider.md` 未纳入提交。

---

# Verification

## Tests

- [x] Unit Tests：不重跑；见 TASK-20261112-006（`task --force verify` exit 0，全部 Go 包与 147 文件／895 项前端）。
- [ ] Integration Tests：N/A
- [ ] E2E Tests：N/A（真实 DSH 176 工具端到端由用户装后验收）

## Quality Checks

- [x] Format：版本提交只改版本字面量、`info.json` 与发行说明。
- [x] Lint：未改业务代码，不重跑（TASK-20261112-006 已通过）。
- [x] Type Check：同上。
- [x] Test：同上。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.10/bin` exit 0。
- [x] Verify：本构建任务不运行 `task verify`；实现任务结果见 TASK-20261112-006。

## Verification Result

```text
scripts/set-release-version.ps1 -Version v2.2.3-beta.10
已校验 v2.2.3-beta.10，Windows 数字版本 2.2.3.0

task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.10/bin
PACKAGE_EXIT=0（makensis Total size: 20,683,630 bytes）

scripts/test-windows-metadata.ps1 -Application tmp/local-v2.2.3-beta.10/bin/omniproxy.exe -Installer bin/OmniProxy-amd64-installer.exe -Version v2.2.3-beta.10
Windows 应用和安装器版本资源校验通过（exit 0）

scripts/verify-release-assets.ps1 -Version v2.2.3-beta.10 -AssetDirectory release-assets/v2.2.3-beta.10
发布资产校验通过：v2.2.3-beta.10（exit 0）
```

- 安装器 `ProductVersion` 为 `2.2.3`；应用与安装器 Authenticode 均为 `NotSigned`。
- 打包后 `git status` 只有本任务的未跟踪记录与既有无关文件，无受跟踪文件差异。

---

# Documentation

- [x] CURRENT：已更新本地构建记录
- [x] 新增 `docs/releases/v2.2.3-beta.10.md`
- [x] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR：无需修改
  （River 工具数量行为已在 TASK-20261112-006 写入 `docs/product/RIVER.md`）

---

# Deviations

- 未调用 `scripts/build-release.ps1`（会强制 `task verify`），沿用本地 Beta 的手动版本映射 +
  `task --force package` + 两项校验脚本流程，与 beta.6–beta.9 一致。
- 版本映射脚本产生的 CRLF 被还原为 LF，否则提交会引入与 `.gitattributes` 冲突的行尾。

---

# Git

## Commit

```text
2a51e6ab chore(release): 准备本地 v2.2.3-beta.10 生产版本
```

功能提交 `4a62be80`、回填提交 `10adc6f5`、版本提交 `2a51e6ab`；本记录与 CURRENT 的归档提交
`dc1ebc5c`。未打标签、未推送、未公开发布。

---

# Result

`v2.2.3-beta.10` Windows amd64 本地 Beta 生产安装器已生成，包含 River 不再按 128 个工具拒绝的修复，
版本资源与摘要核对通过。安装器未签名、未自动安装、未启动；用户从托盘完全退出并备份数据后覆盖安装，
随后恢复 DSH 的 OpenPencil MCP 并实测 176 个工具。

---

# Completed

2026-11-12 16:58
