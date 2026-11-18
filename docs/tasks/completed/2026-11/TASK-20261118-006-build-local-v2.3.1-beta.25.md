# TASK-20261118-006: 构建本地 v2.3.1-beta.25 生产安装包

## Status

Completed

## Created

2026-11-18 13:50（Asia/Singapore）

## Request

缓存命中修复（TASK-20261118-005，`9bdd5d42`）已提交，用户要求继续提交代码并构建新的 Beta
生产版本，用于本地更新，同时更新任务记录。

## Goal

产出 `v2.3.1-beta.25` Windows amd64 本地 Beta 生产安装包，内容为 Google 订阅（Antigravity）
缓存命中 Token 的映射修复。

## Background

`v2.3.1-beta.24` 之后唯一的源码变更是 TASK-20261118-005：Antigravity 适配器读取
`usageMetadata.cachedContentTokenCount` 并写入合成 Chat 的
`usage.prompt_tokens_details.cached_tokens`。本版本无数据库迁移、无 API／配置变更；
历史请求记录仍显示 `—`，因为当时该字段在转换层就被丢弃，无法回填。

## Scope

- 版本号 `2.3.1-beta.25`：`internal/version/version.go`、`frontend/package.json`；
  `build/windows/info.json` 的 `ProductVersion` 由构建脚本同步（数字版本仍 `2.3.1.0`）。
- `docs/releases/v2.3.1-beta.25.md`（构建前必须存在）。
- `scripts/build-release.ps1 -Version v2.3.1-beta.25 -OutputDirectory release-assets\v2.3.1-beta.25`
  （内部 `task --force verify` → `task --force package` → 版本资源与资产校验）。
- 校验安装器大小／SHA256，记录到 Task Record 与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不打标签、不推送、不自动安装、不启动安装后的程序、不修改用户数据。
- 不重新设计缓存展示与成本口径（沿用 TASK-20261023-014 已实现的前端字段）。

---

# Implementation Plan

1. 创建任务记录与发布说明，升版本号。
2. 运行 `scripts/build-release.ps1`，内部跑完整质量门与 NSIS 打包。
3. 校验安装器大小／SHA256、Windows 版本资源与发布资产。
4. 回填任务记录、更新 `CURRENT.md`，提交并回填提交号。

---

# Progress

- 2026-11-18 13:50 创建任务记录，开始升版本与打包。

---

# Implementation Result

## Changes

- 版本号 `2.3.1-beta.25`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`；Windows 数字版本仍 `2.3.1.0`，
  NSIS 基础版本仍 `2.3.1`。
- 新增 `docs/releases/v2.3.1-beta.25.md`（中英成对：问题修复、不兼容变更、本地更新、验证边界）。
- 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.25 -OutputDirectory release-assets\v2.3.1-beta.25`
  （内部 `task --force verify` → `task --force package` → 版本资源与资产校验）。

## Artifacts

- `release-assets/v2.3.1-beta.25/OmniProxy-Setup-v2.3.1-beta.25-windows-amd64.exe`
  **21,390,388 bytes**
  SHA256 `ff6ff50210cc25ca085b0bd68477f199817cf88178b566d0d7dd2ab312ee9f76`
- `release-assets/v2.3.1-beta.25/OmniProxy-Setup-v2.3.1-beta.25-windows-amd64.exe.sha256`
- `release-assets/` 被 `.gitignore` 忽略。

## Technical Notes

- 相对 `v2.3.1-beta.24`（21,385,746 bytes）增加 4,642 bytes：Antigravity 缓存用量映射、
  6 项适配器用例与两份产品文档说明。
- 本版本无数据库迁移；覆盖安装即可，无需重新登录。旧请求记录的缓存列保持 `—`。
- `set-release-version.ps1` 重写 `build/windows/info.json` 带 CRLF，提交前已恢复 LF；
  `frontend/bindings/` 重新生成后与已提交版本一致，无额外变更。

---

# Verification

## Tests

- [x] `scripts/build-release.ps1` 内 `task --force verify` 通过（format／lint／typecheck、
      Go 全包、前端 159 文件／977 项测试、Windows production 构建）
- [x] `scripts/test-windows-metadata.ps1`：应用与安装器版本资源校验通过（`2.3.1-beta.25`／`2.3.1.0`）
- [x] `scripts/verify-release-assets.ps1`：发布资产校验通过（`v2.3.1-beta.25`）
- [x] 安装器大小／SHA256 与 `.sha256` 一致（21,390,388 bytes／`ff6ff502…`）
- [ ] 安装后新请求的缓存命中未做截图核对（需用户安装后跑一条 Google 请求确认）

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
已校验 v2.3.1-beta.25，Windows 数字版本 2.3.1.0
task --force verify   → 通过（Test Files 159 passed / Tests 977 passed）
task --force package  → 通过（NSIS，215.5 MB 输入压缩为 20.4 MiB）
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.3.1-beta.25
BUILD_EXIT=0
```

---

# Documentation

- [x] `docs/releases/v2.3.1-beta.25.md`
- [x] `docs/plans/CURRENT.md`

---

# Deviations

None.

---

# Git

## Commit

Pending

---

# Result

已产出 `v2.3.1-beta.25` Windows amd64 本地 Beta 生产安装包（21,390,388 bytes，
SHA256 `ff6ff50210cc25ca085b0bd68477f199817cf88178b566d0d7dd2ab312ee9f76`），内容为
Google 订阅（Antigravity）缓存命中 Token 映射（TASK-20261118-005，`9bdd5d42`）。
未签名、未公开发布、未打标签、未推送、未自动安装。

---

# Completed

2026-11-18 13:57（Asia/Singapore）
