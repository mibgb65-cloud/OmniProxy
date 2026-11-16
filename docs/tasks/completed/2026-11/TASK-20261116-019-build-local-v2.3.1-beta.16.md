# TASK-20261116-019: 构建本地 v2.3.1-beta.16 生产安装包

## Status

Completed

## Created

2026-11-16 21:00（Asia/Singapore）

## Request

WorkBuddy AI 浏览器登录后账号无法保存的修复（TASK-20261116-018）已完成并通过质量门，
用户需要在已安装的 `v2.3.1-beta.15` 上覆盖安装一个新版本验证登录。

## Goal

产出 `v2.3.1-beta.16` Windows amd64 本地 Beta 生产安装包，内容为 WorkBuddy AI 登录保存修复
（新增账号 INSERT、`save_failed`/`invalid_input` 错误码、登录失败日志、浏览器授权说明）。

## Background

`v2.3.1-beta.15` 首次引入 WorkBuddy AI 订阅通道（TASK-20261116-015）。用户实测：浏览器登录
成功，但 OmniProxy 报通用错误、账号未创建。定位为新建账号 INSERT 列数与值数不匹配
（TASK-20261116-018），修复后需要新安装包才能验证。本版本无数据库迁移变化
（050／051 已随 beta.15 发布并已在用户机器上执行）。

## Scope

- 版本号 `2.3.1-beta.16`：`internal/version/version.go`、`frontend/package.json`；
  `build/windows/info.json` 的 `ProductVersion` 同步（数字版本仍 `2.3.1.0`）。
- `docs/releases/v2.3.1-beta.16.md`。
- `scripts/build-release.ps1 -Version v2.3.1-beta.16 -OutputDirectory release-assets\v2.3.1-beta.16`。
- 校验安装器大小／SHA256、原生版本资源与发布资产；记录到 Task Record 与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不打标签、不推送、不自动安装、不启动安装后的程序、不修改用户数据。
- 不执行 WorkBuddy AI 真实 Chat 或额度读取（消耗积分，需单独授权）。

---

# Implementation Result

## Changes

- 版本号 `2.3.1-beta.16`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`；Windows 数字版本仍 `2.3.1.0`，
  NSIS 基础版本仍 `2.3.1`（脚本同步后与提交前一致，无差异）。
- 新增 `docs/releases/v2.3.1-beta.16.md`：根因（11 列 / 12 值）、新增错误码、
  覆盖安装步骤与验证边界（真实重试未验证）。
- 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.16 -OutputDirectory release-assets\v2.3.1-beta.16`：
  校验版本一致 → `task --force verify` → `task --force package` → 原生版本资源与发布资产校验。

## Files Changed

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `docs/releases/v2.3.1-beta.16.md`（新增）

## Artifacts

- `release-assets/v2.3.1-beta.16/OmniProxy-Setup-v2.3.1-beta.16-windows-amd64.exe`
  **21,314,100 bytes**
  SHA256 `ad955b5cbe778f54ca15a2d81d6d41845439bd3d1f49f150c9d3280b1cf1a5d8`
- `release-assets/v2.3.1-beta.16/OmniProxy-Setup-v2.3.1-beta.16-windows-amd64.exe.sha256`
- 构建产物 `bin/OmniProxy-amd64-installer.exe` 同为 21,314,100 bytes；`release-assets/` 被 `.gitignore` 忽略。

## Technical Notes

- 相对 `v2.3.1-beta.15`（21,304,208 bytes）增加 9,892 bytes：仅 SQL 修正、错误码与日志、
  前端提示与测试，无新增依赖、无新增迁移。
- 构建脚本内部的 `task --force verify` 已覆盖本次修复的 Go 与前端测试
  （157 files / 964 tests）。
- `set-release-version.ps1` 重写 `build/windows/info.json` 时带 CRLF，已恢复 LF；
  因版本值与 HEAD 相同，本任务对 `info.json` 没有产生提交差异。

---

# Verification

## Tests

- [x] `scripts/build-release.ps1` 内 `task --force verify` 通过（format／lint／typecheck、
      后端全部 Go 包、前端 157 文件／964 项测试、Windows production 构建）
- [x] `scripts/test-windows-metadata.ps1`：应用与安装器版本资源校验通过（`2.3.1-beta.16`／`2.3.1.0`）
- [x] `scripts/verify-release-assets.ps1`：发布资产校验通过（`v2.3.1-beta.16`）
- [x] 安装器大小／SHA256 与 `.sha256` 文件一致（21,314,100 bytes／`ad955b5c…`）
- [ ] 安装后真实登录验收未执行：需用户覆盖安装后重新点一次 WorkBuddy AI 登录
- [ ] `/v3/config` 目录同步、真实 Chat、Credits 读数未验证（消耗积分，需单独授权）

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
已校验 v2.3.1-beta.16，Windows 数字版本 2.3.1.0
task --force verify   → 通过（Test Files 157 passed / Tests 964 passed）
task --force package  → 通过（NSIS，Total size 21,314,100 / 85,683,694 bytes）
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.3.1-beta.16
BUILD_EXIT=0
```

---

# Documentation

- [x] `docs/releases/v2.3.1-beta.16.md`
- [x] `docs/plans/CURRENT.md`

---

# Deviations

- 无偏离：版本号、发布说明、打包脚本与既有两个 Beta 构建任务一致。uild/windows/info.json 由脚本重写为 CRLF，已恢复 LF；因版本值与提交一致，本次没有产生提交差异。elease-assets/v2.3.1-beta.16/ 与 in/ 均被 .gitignore 忽略。

---

# Git

## Commit

a78537ab chore(release): 准备 v2.3.1-beta.16 本地构建

---

# Result

已产出 `v2.3.1-beta.16` Windows amd64 本地 Beta 生产安装包（21,314,100 bytes，
SHA256 `ad955b5cbe778f54ca15a2d81d6d41845439bd3d1f49f150c9d3280b1cf1a5d8`），
内容为 TASK-20261116-018 的 WorkBuddy AI 登录保存修复。未签名、未公开发布、未打标签、
未推送、未自动安装。用户从托盘完全退出 OmniProxy、备份数据目录后覆盖安装，
随后重新点一次「登录 WorkBuddy AI」即可验证账号是否落库（上次失败没有留下账号）。

---

# Completed

2026-11-16 21:01（Asia/Singapore）