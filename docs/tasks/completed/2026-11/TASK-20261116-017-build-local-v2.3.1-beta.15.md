# TASK-20261116-017: 构建本地 v2.3.1-beta.15 生产安装包

## Status

Completed

## Created

2026-11-16 19:35（Asia/Singapore）

## Request

WorkBuddy AI 订阅通道（TASK-20261116-015，提交 `f7ddb0fa`）已实现并通过 `task verify`，
用户要求「提交代码后，然后构建新的beta生产版本我进行本地更新」。

## Goal

产出 `v2.3.1-beta.15` Windows amd64 本地 Beta 生产安装包，内容为第十五个内置服务商
WorkBuddy AI（浏览器登录、`/v3/config` 目录、Chat 原生与跨协议转换、Credits 三窗口额度），
供用户从托盘退出后覆盖安装验收。

## Background

上一个本地构建为 `v2.3.1-beta.14`（TASK-20261116-016，21,100,525 bytes，
SHA256 `9468a335…`），内容为 Responses 输入条数上限与 custom 工具历史修复。
本次 Build 与 TASK-015 是「实现」与「打包验收」的关系：015 已提交，本任务只做版本同步、
发布说明、打包与资产校验。

## Scope

- 版本号 `2.3.1-beta.15`：`internal/version/version.go`、`frontend/package.json`；
  `build/windows/info.json` 的 `ProductVersion` 同步（数字版本仍 `2.3.1.0`）。
- `docs/releases/v2.3.1-beta.15.md`。
- `scripts/build-release.ps1 -Version v2.3.1-beta.15 -OutputDirectory release-assets\v2.3.1-beta.15`
  （内部执行 `task --force verify` 与 `task --force package`）。
- 校验安装器大小／SHA256、原生版本资源与发布资产；记录到本 Task Record 与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不打标签、不推送、不自动安装、不启动安装后的程序、不修改用户数据。
- 不执行 WorkBuddy AI 真机登录或真实调用（消耗积分，需用户单独授权）。
- 不修改 TASK-015 的 provider 实现；本任务只允许版本号、发布说明、任务记录与看板变更。

---

# Implementation Result

## Changes

- 版本号 `2.3.1-beta.15`：`internal/version/version.go`（`var Version`）、
  `frontend/package.json`（`version`）、`build/windows/info.json`（`ProductVersion`）；
  Windows 数字版本仍为 `2.3.1.0`，NSIS 基础版本仍为 `2.3.1`（`build/config.yml`、
  `build/windows/nsis/wails_tools.nsh` 由脚本同步后与提交前一致，无差异）。
- 新增 `docs/releases/v2.3.1-beta.15.md`：服务商能力、**本版本含数据库迁移**（050 追加一条
  内置服务商记录、051 新建额度快照表，启动时自动执行、只追加数据）、本地覆盖安装步骤、
  验证边界与未验证项（真实账号登录与真实调用会消耗积分，未执行）。
- 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.15 -OutputDirectory release-assets\v2.3.1-beta.15`：
  脚本先校验源码／前端版本与标签一致，再同步 `build/config.yml`、`build/windows/info.json`、
  `build/windows/nsis/wails_tools.nsh`，随后执行 `task --force verify` 与 `task --force package`，
  最后校验原生版本资源与发布资产。

## Files Changed

- `internal/version/version.go`（`2.3.1-beta.15`）
- `frontend/package.json`（`2.3.1-beta.15`）
- `build/windows/info.json`（`ProductVersion`；脚本重写后已恢复 LF，提交差异只有版本行）
- `docs/releases/v2.3.1-beta.15.md`（新增）

## Artifacts

- `release-assets/v2.3.1-beta.15/OmniProxy-Setup-v2.3.1-beta.15-windows-amd64.exe`
  **21,304,208 bytes**
  SHA256 `ab3846b608fe9d844062c9e0cfa9729d550b4def40a0fe56120217e2cf3b1e5e`
- `release-assets/v2.3.1-beta.15/OmniProxy-Setup-v2.3.1-beta.15-windows-amd64.exe.sha256`
- 构建产物 `bin/omniproxy.exe` 24,624,640 bytes、`bin/OmniProxy-amd64-installer.exe` 21,304,208 bytes
  （安装包与发布目录中的副本同一文件）。
- `release-assets/` 在 `.gitignore` 中，安装包不进版本控制。

## Technical Notes

- 构建脚本自身执行完整质量门，因此本任务的 `task verify` 实际由脚本在打包前跑一遍
  （包含重新生成绑定与 Windows production 构建），本次源码未改动任何 Go／前端实现。
- `set-release-version.ps1` 用 PowerShell 5.1 的 `ConvertTo-Json` 重写 `build/windows/info.json`，
  输出为 CRLF；已按既有做法恢复为 LF，并确认格式与提交前一致（仅版本值变化）。
- 与 `v2.3.1-beta.14`（21,100,525 bytes）相比安装包增加 203,683 bytes，主要来自新增 provider
  适配器、Wails 绑定与前端额度组件。
- 本版本**有数据库迁移**（与 beta.13／beta.14 不同）：050／051 在覆盖安装后首次启动时自动执行，
  只追加数据，不重置既有服务商、账号、路由与统计。

---

# Verification

## Tests

- [x] `scripts/build-release.ps1` 内 `task --force verify` 通过（格式、lint、类型检查、
      后端全部 Go 包、前端 156 文件／961 项测试、Windows production 构建）
- [x] `scripts/test-windows-metadata.ps1`：应用与安装器版本资源校验通过（`2.3.1-beta.15`／`2.3.1.0`）
- [x] `scripts/verify-release-assets.ps1`：发布资产校验通过（`v2.3.1-beta.15`）
- [x] 安装器大小／SHA256 与 `.sha256` 文件一致（21,304,208 bytes／`ab3846b6…`）
- [ ] 安装与原生窗口验收未执行（需用户自行覆盖安装后确认）
- [ ] WorkBuddy AI 真实账号登录与真实 Chat／Credits 未验证（消耗积分，需单独授权）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify
- [x] Packaging

## Verification Result

```text
已校验 v2.3.1-beta.15，Windows 数字版本 2.3.1.0
task --force verify            → 通过
  Test Files  156 passed (156)
  Tests       961 passed (961)
task --force package           → 通过（NSIS，Total size 21,304,208 / 85,646,742 bytes）
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.3.1-beta.15
BUILD_EXIT=0
```

---

# Documentation

- [x] `docs/releases/v2.3.1-beta.15.md`
- [x] `docs/plans/CURRENT.md`

---

# Deviations

- 计划中的 `build/windows/info.json` 由脚本同步；实际脚本用 PowerShell 5.1 的 `ConvertTo-Json`
  写回 CRLF，已按 TASK-016 的既有做法恢复 LF，提交差异只有 `ProductVersion` 一行。
- 无其他偏离。未新增或修改任何 Go／前端实现文件；`bin/` 与 `release-assets/` 均被 `.gitignore` 忽略。

---

# Git

## Commit

a8209137 chore(release): 准备 v2.3.1-beta.15 本地构建

---

# Result

已产出 `v2.3.1-beta.15` Windows amd64 本地 Beta 生产安装包（21,304,208 bytes，
SHA256 `ab3846b608fe9d844062c9e0cfa9729d550b4def40a0fe56120217e2cf3b1e5e`），内容为
TASK-20261116-015 的 WorkBuddy AI 订阅通道（第十五个内置服务商），并含迁移 050／051。
未签名、未公开发布、未打标签、未推送、未自动安装。用户从托盘完全退出 OmniProxy、备份数据目录后
覆盖安装，首次启动会自动执行新增迁移；随后可在账号页用浏览器登录 WorkBuddy AI 验收。

---

# Completed

2026-11-16 19:42（Asia/Singapore）
