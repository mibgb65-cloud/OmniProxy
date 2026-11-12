# TASK-20261112-010: 构建本地 v2.2.3-beta.11 生产安装包

## Status

Completed

## Created

2026-11-12 20:15（Asia/Singapore）

## Request

用户在确认 Antigravity 桌面端接入后要求：

> 可以，你提交代码后，然后构建新的 beta 生产版本我进行本地更新，记得完成任务 md 更新

即：把 TASK-20261112-009（Antigravity 桌面端接入，含 Claude／GPT-OSS 全部订阅模型）
纳入 `v2.2.3-beta.11`，产出本地 Windows amd64 生产安装包，供用户从托盘退出后覆盖更新。

## Goal

- 版本由 `2.2.3-beta.10` 提升为 `2.2.3-beta.11`，Windows 数字版本保持 `2.2.3.0`、
  NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.11.md`。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。
- 更新任务记录与 `docs/plans/CURRENT.md`。

## Background

- `v2.2.3-beta.10` 已于 2026-11-12 构建（TASK-20261112-007，版本提交 `2a51e6ab`），
  当前源码版本仍为 `2.2.3-beta.10`。
- 自 beta.10 起的用户可见改动为 TASK-20261112-009：
  - `3225476f` Antigravity 桌面端通过官方 Enterprise Gateway 接入本地网关；
  - `9fc76a87` 接受桌面端目录的 `X-Goog-Api-Key` 接入头；
  - `8aee0628` 断开按字段恢复并支持端口改写；
  - `86462aec` 目录发布 Google 订阅全部模型（Gemini、Claude、GPT-OSS）；
  - `ceaa0a36`／`bcee5e75`／`e2156308`／`6087041e` 任务记录归档与回填。
- 工作区另有一条未跟踪、与本次无关的旧记录
  `docs/tasks/completed/2026-10/TASK-20261029-010-subq-provider.md`，保持原样，不纳入构建或提交。

## Scope

- `internal/version/version.go`、`frontend/package.json` 与 `build/windows/info.json` 的版本映射。
- `docs/releases/v2.2.3-beta.11.md`。
- 生产打包、产物复制到 `release-assets/v2.2.3-beta.11/`、SHA256 侧车与两项校验脚本。
- 任务记录与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不重跑 `task verify`：TASK-20261112-009 已运行完整 Go 测试、前端 147 文件／896 项、
  typecheck、lint、format 与生产构建；本任务只做版本映射与打包。
- 不推送、不打标签、不公开发布、不签名。
- 不自动安装、不启动或停止用户实例，不修改真实用户数据。

## Existing Behavior

版本仍为 `2.2.3-beta.10`，`release-assets/v2.2.3-beta.11` 不存在；当前运行实例为旧构建，
生产端口 3000。

## Expected Behavior

用户覆盖安装后，网关页出现 Antigravity 桌面端卡片；连接后重启 Antigravity 可读取
Gemini／Claude／GPT-OSS 全部 Google 订阅模型，并通过现有账号池完成请求。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`
- `build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/releases/`、`docs/plans/CURRENT.md`
- `release-assets/`

## Dependencies

沿用 Wails v3.1.0、Task、Go 1.26.2、pnpm、NSIS 及既有版本映射／构建校验脚本。

## Risks

- 本地安装器未签名，可能触发未知发布者或 SmartScreen 提示。
- 用户当前实例仍在运行旧构建（端口 3000），未完全退出时覆盖安装会被拒绝；
  发布说明要求先从托盘退出并备份数据。
- 版本与资源映射不一致或复用旧产物会导致错误交付；用 `--force` 生产构建并核对
  原生版本资源与 SHA256。
- 工作区存在他人未提交的旧任务记录，提交时只暂存本任务文件。

---

# Implementation Plan

1. 创建本记录；确认当前版本与发布目录状态。
2. 更新 `internal/version/version.go` 与 `frontend/package.json`，运行
   `scripts/set-release-version.ps1 -Version v2.2.3-beta.11`。
3. 新增双语发行说明并审阅，仅提交本任务文件。
4. 执行 `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.11/bin`。
5. 复制安装器到 `release-assets/v2.2.3-beta.11/`，生成 SHA256 侧车，运行
   `scripts/test-windows-metadata.ps1` 与 `scripts/verify-release-assets.ps1`。
6. 填写实际结果，归档任务并提交，更新 CURRENT。

---

# Progress

- 2026-11-12 20:15：创建记录；确认当前版本 `2.2.3-beta.10`，`release-assets/v2.2.3-beta.11`
  不存在；开始版本映射与打包。
- 版本映射通过：应用／前端／Windows 显示版本 `2.2.3-beta.11`，数字版本 `2.2.3.0`，
  NSIS 基础版本 `2.2.3`（`info.json` 的 CRLF 已还原为 LF）。
- 发行说明与生产打包完成：安装器 20,715,162 bytes；元数据与发布资产校验通过。

---

# Implementation Result

## Changes

- 应用、前端与 Windows 显示版本统一为 `2.2.3-beta.11`；Windows 数字资源 `2.2.3.0`、
  NSIS 基础版本 `2.2.3` 不变。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.11.md`：说明 Antigravity 桌面端官方
  Enterprise Gateway 接入、Gemini／Claude／GPT-OSS 全部订阅模型、多账号复用、
  字段级断开恢复、端口改写，以及未签名本地安装与真机验证边界。
- 版本中包含 TASK-20261112-009 的实现与文档提交；无新增迁移、依赖或绑定差异。

## Build and Artifacts

构建命令（当前进程 PATH 临时加入 NSIS）：

```text
task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.11/bin
```

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.11/OmniProxy-Setup-v2.2.3-beta.11-windows-amd64.exe` | 20,715,162 | `8354c351a9acc0f68608c258574ab7024ca5247d17d715eb84dcc8710ca28af8` |
| `tmp/local-v2.2.3-beta.11/bin/omniproxy.exe` | 23,788,544 | `aca3dfc494921f2ef796fd8ea9f3d76f381d0f476851dd05bc7a555f9c16b51c` |

摘要侧车 `OmniProxy-Setup-v2.2.3-beta.11-windows-amd64.exe.sha256` 与安装器同目录，
内容为 `<hash>  <文件名>` 加结尾换行、无 BOM。原始 NSIS 安装器位于
`bin/OmniProxy-amd64-installer.exe`。产物目录与 `tmp/` 不入 Git。

## Technical Notes

- 版本映射由 `scripts/set-release-version.ps1` 完成；脚本把 `build/windows/info.json`
  写成 CRLF，已还原为 LF 后再提交，保持仓库无 CRLF。
- 本机 `pwsh` 不在 PATH，脚本用 `powershell -NoProfile -ExecutionPolicy Bypass -File`
  执行；NSIS 通过临时追加 `C:\Program Files (x86)\NSIS` 到 PATH 找到。
- 打包日志中的 `uname`／`tail` 缺失提示来自 Taskfile 的 Windows 分支探测，构建仍
  exit 0。
- 工作区未跟踪的 `TASK-20261029-010-subq-provider.md` 未纳入提交。

---

# Verification

## Tests

- [x] Unit Tests：不重跑；见 TASK-20261112-009（Go 52 个包、前端 147 文件／896 项通过）。
- [ ] Integration Tests：N/A
- [ ] E2E Tests：N/A（真实 Antigravity 桌面端请求由用户装后验收）

## Quality Checks

- [x] Format：版本提交只改版本字面量与 `info.json`，新增发行说明；Go 文件已 gofmt。
- [x] Lint：未改业务代码，不重跑（TASK-20261112-009 已通过）。
- [x] Type Check：同上。
- [x] Test：同上。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.11/bin` exit 0。
- [x] Verify：本构建任务不运行 `task verify`；实现任务结果见 TASK-20261112-009。

## Verification Result

```text
scripts/set-release-version.ps1 -Version v2.2.3-beta.11
已校验 v2.2.3-beta.11，Windows 数字版本 2.2.3.0

task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.11/bin
PACKAGE_EXIT=0（makensis Total size: 20,715,162 bytes）

scripts/test-windows-metadata.ps1 -Application tmp/local-v2.2.3-beta.11/bin/omniproxy.exe -Installer bin/OmniProxy-amd64-installer.exe -Version v2.2.3-beta.11
Windows 应用和安装器版本资源校验通过（exit 0）

scripts/verify-release-assets.ps1 -Version v2.2.3-beta.11 -AssetDirectory release-assets/v2.2.3-beta.11
发布资产校验通过：v2.2.3-beta.11（exit 0）
```

- 安装器 `ProductVersion` 为 `2.2.3`；应用与安装器 Authenticode 均为 `NotSigned`。
- 打包后 `git status` 只有本任务的版本／发行说明／记录与既有无关文件，无前端绑定差异。

---

# Documentation

- [x] CURRENT：已更新本地构建记录
- [x] 新增 `docs/releases/v2.2.3-beta.11.md`
- [x] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR：无需修改

---

# Deviations

- 未调用 `scripts/build-release.ps1`（会强制 `task verify`），沿用本地 Beta 的版本映射 +
  `task --force package` + 两项校验脚本流程，与 beta.6–beta.10 一致。
- 版本映射脚本产生的 CRLF 被还原为 LF，避免与 `.gitattributes` 冲突。
- 未执行 TASK-20261112-009 遗留的真机请求验证：本机没有 Antigravity 桌面端，验证由用户
  覆盖安装后完成。

---

# Git

## Commit

```text
fbaec345 chore(release): 准备本地 v2.2.3-beta.11 生产版本
```

Antigravity 实现提交 `3225476f`／`9fc76a87`／`8aee0628`／`86462aec`、回填提交
`ceaa0a36`／`e2156308`／`bcee5e75`／`6087041e`、版本提交 `fbaec345`；本记录与 CURRENT 的
归档提交 `95a65c01`。未打标签、未推送、未公开发布。

---

# Result

`v2.2.3-beta.11` Windows amd64 本地 Beta 生产安装器已生成，包含 Antigravity 桌面端
官方网关接入与 Google 订阅全部模型（Gemini／Claude／GPT-OSS），版本资源与摘要核对
通过。安装器未签名、未自动安装、未启动；用户从托盘完全退出并备份数据后覆盖安装，
再按卡片提示连接并重启 Antigravity。

---

# Completed

2026-11-12 20:40
