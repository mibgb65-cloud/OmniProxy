# TASK-20261118-004: 构建本地 v2.3.1-beta.24 生产安装包

## Status

Completed

## Created

2026-11-18 10:55（Asia/Singapore）

## Request

套餐徽章区分的改动（TASK-20261118-003，`c8edf0ea`）已提交，用户要求继续打包新的 Beta 生产版本用于本地更新。
用户已自行升级 Codex CLI，OpenAI 目录中的 GPT-6.2 系列已出现，不需要本任务处理。

## Goal

产出 `v2.3.1-beta.24` Windows amd64 本地 Beta 生产安装包，内容为 Antigravity「Ultra 5x／Ultra 20x」
套餐徽章区分。

## Background

`v2.3.1-beta.23` 之后唯一的源码变更是 TASK-20261118-003：`g1-ultra-tier` 显示「Ultra 20x」（rose 实底）、
`g1-ultra-lite-tier` 显示「Ultra 5x」（rose 描边）。本版本无数据库迁移、无后端改动、无 OpenAI 目录逻辑改动。

## Scope

- 版本号 `2.3.1-beta.24`：`internal/version/version.go`、`frontend/package.json`；
  `build/windows/info.json` 的 `ProductVersion` 由构建脚本同步（数字版本仍 `2.3.1.0`）。
- `docs/releases/v2.3.1-beta.24.md`（构建前必须存在）。
- `scripts/build-release.ps1 -Version v2.3.1-beta.24 -OutputDirectory release-assets\v2.3.1-beta.24`。
- 校验安装器大小／SHA256、原生版本资源与发布资产；记录到 Task Record 与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不打标签、不推送、不自动安装、不启动安装后的程序、不修改用户数据。
- 不修改 Codex CLI、不改 OpenAI 模型同步逻辑。

---

# Implementation Result

## Changes

- 版本号 `2.3.1-beta.24`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`；Windows 数字版本仍 `2.3.1.0`，
  NSIS 基础版本仍 `2.3.1`。
- 新增 `docs/releases/v2.3.1-beta.24.md`（中英成对：问题修复、不兼容变更、本地更新、验证边界）。
- 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.24 -OutputDirectory release-assets\v2.3.1-beta.24`
  （内部 `task --force verify` → `task --force package` → 版本资源与资产校验）。

## Artifacts

- `release-assets/v2.3.1-beta.24/OmniProxy-Setup-v2.3.1-beta.24-windows-amd64.exe`
  **21,385,746 bytes**
  SHA256 `942e5bad8aa3208bf03fbcc8713ad007c7b13497e73f6ca90ae0a9b2836b8f97`
- `release-assets/v2.3.1-beta.24/OmniProxy-Setup-v2.3.1-beta.24-windows-amd64.exe.sha256`
- `release-assets/` 被 `.gitignore` 忽略。

## Technical Notes

- 相对 `v2.3.1-beta.23`（21,383,326 bytes）增加 2,420 bytes：Ultra 5x／20x 对照与配色、2 项徽章测试。
- 本版本无数据库迁移；覆盖安装即可，无需重新登录。
- `set-release-version.ps1` 重写 `build/windows/info.json` 带 CRLF，构建后已恢复 LF。

---

# Verification

## Tests

- [x] `scripts/build-release.ps1` 内 `task --force verify` 通过（format／lint／typecheck、
      Go 全包、前端 159 文件／977 项测试、Windows production 构建）
- [x] `scripts/test-windows-metadata.ps1`：应用与安装器版本资源校验通过（`2.3.1-beta.24`／`2.3.1.0`）
- [x] `scripts/verify-release-assets.ps1`：发布资产校验通过（`v2.3.1-beta.24`）
- [x] 安装器大小／SHA256 与 `.sha256` 一致（21,385,746 bytes／`942e5bad…`）
- [ ] 安装后界面徽章未做截图核对（需用户安装后确认）

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
已校验 v2.3.1-beta.24，Windows 数字版本 2.3.1.0
task --force verify   → 通过（Test Files 159 passed / Tests 977 passed）
task --force package  → 通过（NSIS）
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.3.1-beta.24
BUILD_EXIT=0
```

---

# Documentation

- [x] `docs/releases/v2.3.1-beta.24.md`
- [x] `docs/plans/CURRENT.md`

---

# Deviations

None.

---

# Git

## Commit

（完成后填写）

---

# Result

已产出 `v2.3.1-beta.24` Windows amd64 本地 Beta 生产安装包（21,385,746 bytes，
SHA256 `942e5bad8aa3208bf03fbcc8713ad007c7b13497e73f6ca90ae0a9b2836b8f97`），内容为
Antigravity「Ultra 5x／Ultra 20x」套餐徽章区分。未签名、未公开发布、未打标签、未推送、未自动安装。

---

# Completed

2026-11-18 11:00（Asia/Singapore）
