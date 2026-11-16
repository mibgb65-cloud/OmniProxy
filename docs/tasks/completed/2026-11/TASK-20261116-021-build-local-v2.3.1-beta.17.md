# TASK-20261116-021: 构建本地 v2.3.1-beta.17 生产安装包

## Status

Completed

## Created

2026-11-16 21:40（Asia/Singapore）

## Request

WorkBuddy AI Credits 读取修复（TASK-20261116-020）完成后，用户需要在已安装的
`v2.3.1-beta.16` 上覆盖安装并验证额度显示。

## Goal

产出 `v2.3.1-beta.17` Windows amd64 本地 Beta 生产安装包，内容为 Credits 计费接口修复
（POST + 计费头集、套餐名取 `SubscriptionPackageName`、`gateway:` 错误码识别、额度卡显示套餐名）。

## Background

`v2.3.1-beta.16` 已能成功创建 WorkBuddy AI 账号并同步 25 条真实模型（`/v3/config` 正常），
但 Credits 用 `GET` 调计费接口被上游 404，且 `proxy.Error` 的 `gateway: <code>` 文案不被前端
识别，界面只显示通用失败。修复后需要新安装包验证。本版本无数据库迁移变化。

## Scope

- 版本号 `2.3.1-beta.17`：`internal/version/version.go`、`frontend/package.json`；
  `build/windows/info.json` 的 `ProductVersion` 同步（数字版本仍 `2.3.1.0`）。
- `docs/releases/v2.3.1-beta.17.md`。
- `scripts/build-release.ps1 -Version v2.3.1-beta.17 -OutputDirectory release-assets\v2.3.1-beta.17`。
- 校验安装器大小／SHA256、原生版本资源与发布资产；记录到 Task Record 与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不打标签、不推送、不自动安装、不启动安装后的程序、不修改用户数据。
- 不做真实 Chat 调用（消耗积分，需单独授权）。

---

# Implementation Result

## Changes

- 版本号 `2.3.1-beta.17`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`；Windows 数字版本仍 `2.3.1.0`，
  NSIS 基础版本仍 `2.3.1`（脚本同步后与提交一致，无差异）。
- 新增 `docs/releases/v2.3.1-beta.17.md`：计费接口形态、套餐名来源、错误码识别变化与验证边界。
- 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.17 -OutputDirectory release-assets\v2.3.1-beta.17`。

## Artifacts

- `release-assets/v2.3.1-beta.17/OmniProxy-Setup-v2.3.1-beta.17-windows-amd64.exe`
  **21,323,209 bytes**
  SHA256 `b2e63233fb5e0ce8581963e500b68c8838ac87d734ac981d155e7d2a200068f0`
- `release-assets/v2.3.1-beta.17/OmniProxy-Setup-v2.3.1-beta.17-windows-amd64.exe.sha256`
- `release-assets/` 被 `.gitignore` 忽略。

## Technical Notes

- 相对 `v2.3.1-beta.16`（21,314,100 bytes）增加 9,109 bytes：计费请求形态、套餐名、
  错误码识别与测试，无新增依赖、无新增迁移。
- 构建脚本内部的 `task --force verify` 覆盖本次修复（前端 157 files / 965 tests）。
- `set-release-version.ps1` 重写 `build/windows/info.json` 带 CRLF，已恢复 LF；
  版本值与提交一致，无提交差异。

---

# Verification

## Tests

- [x] `scripts/build-release.ps1` 内 `task --force verify` 通过（format／lint／typecheck、
      Go 全包、前端 157 文件／965 项测试、Windows production 构建）
- [x] `scripts/test-windows-metadata.ps1`：应用与安装器版本资源校验通过（`2.3.1-beta.17`／`2.3.1.0`）
- [x] `scripts/verify-release-assets.ps1`：发布资产校验通过（`v2.3.1-beta.17`）
- [x] 安装器大小／SHA256 与 `.sha256` 一致（21,323,209 bytes／`b2e63233…`）
- [ ] 安装后应用内额度显示未验证（需用户覆盖安装后打开账号卡）
- [ ] 真实 Chat 与计费扣减未验证（消耗积分）

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
已校验 v2.3.1-beta.17，Windows 数字版本 2.3.1.0
task --force verify   → 通过（Test Files 157 passed / Tests 965 passed）
task --force package  → 通过（NSIS，Total size 21,323,209 / 85,720,698 bytes）
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.3.1-beta.17
BUILD_EXIT=0
```

---

# Documentation

- [x] `docs/releases/v2.3.1-beta.17.md`
- [x] `docs/plans/CURRENT.md`

---

# Deviations

- 无偏离：与既有 Beta 构建任务同一流程。`build/windows/info.json` 由脚本重写为 CRLF，
  已恢复 LF；版本值与提交一致，本次没有提交差异。

---

# Git

## Commit

eeadc35d chore(release): 准备 v2.3.1-beta.17 本地构建

---

# Result

已产出 `v2.3.1-beta.17` Windows amd64 本地 Beta 生产安装包（21,323,209 bytes，
SHA256 `b2e63233fb5e0ce8581963e500b68c8838ac87d734ac981d155e7d2a200068f0`），
内容为 TASK-20261116-020 的 WorkBuddy AI Credits 修复。未签名、未公开发布、未打标签、
未推送、未自动安装。用户从托盘完全退出 OmniProxy 后覆盖安装，打开账号卡应直接看到
Credits 窗口与套餐名（Free），不需要重新登录。

---

# Completed

2026-11-16 22:00（Asia/Singapore）