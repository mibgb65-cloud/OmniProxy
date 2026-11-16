# TASK-20261116-023: 构建本地 v2.3.1-beta.18 生产安装包

## Status

Completed

## Created

2026-11-16 22:30（Asia/Singapore）

## Request

WorkBuddy AI 手动同步模型失败的修复（TASK-20261116-022）完成后，用户需要在已安装的
`v2.3.1-beta.17` 上覆盖安装验证「同步官方模型」按钮。

## Goal

产出 `v2.3.1-beta.18` Windows amd64 本地 Beta 生产安装包，内容为目录服务接线修复
（手动同步与自动同步共用 `wireModelCatalog`、同步失败稳定码进日志）。

## Background

`v2.3.1-beta.17` 上 WorkBuddy AI 账号已建、25 条模型已同步、Credits 已可读；仅
「路由 → 模型目录 → 同步官方模型」失败（Wails 层目录服务漏接 WorkBuddy 订阅）。
本版本无数据库迁移变化。

## Scope

- 版本号 `2.3.1-beta.18`：`internal/version/version.go`、`frontend/package.json`；
  `build/windows/info.json` 的 `ProductVersion` 同步（数字版本仍 `2.3.1.0`）。
- `docs/releases/v2.3.1-beta.18.md`。
- `scripts/build-release.ps1 -Version v2.3.1-beta.18 -OutputDirectory release-assets\v2.3.1-beta.18`。
- 校验安装器大小／SHA256、原生版本资源与发布资产；记录到 Task Record 与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不打标签、不推送、不自动安装、不启动安装后的程序、不修改用户数据。
- 不做真实对话调用（消耗积分，需单独授权）。

---

# Implementation Result

## Changes

- 版本号 `2.3.1-beta.18`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`；Windows 数字版本仍 `2.3.1.0`，
  NSIS 基础版本仍 `2.3.1`（脚本同步后与提交一致，无差异）。
- 新增 `docs/releases/v2.3.1-beta.18.md`。
- 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.18 -OutputDirectory release-assets\v2.3.1-beta.18`。

## Artifacts

- `release-assets/v2.3.1-beta.18/OmniProxy-Setup-v2.3.1-beta.18-windows-amd64.exe`
  **21,328,474 bytes**
  SHA256 `d0e18021c40be3dcf8810f412c8845cb1311efaa6e942064332a0b4f2660880a`
- `release-assets/v2.3.1-beta.18/OmniProxy-Setup-v2.3.1-beta.18-windows-amd64.exe.sha256`
- `release-assets/` 被 `.gitignore` 忽略。

## Technical Notes

- 相对 `v2.3.1-beta.17`（21,323,209 bytes）增加 5,265 bytes：接线收敛、诊断与两个回归测试。
- 构建脚本内部的 `task --force verify` 覆盖本次修复（前端 157 files / 965 tests）。
- `set-release-version.ps1` 重写 `build/windows/info.json` 带 CRLF，已恢复 LF；版本值与提交一致。

---

# Verification

## Tests

- [x] `scripts/build-release.ps1` 内 `task --force verify` 通过（format／lint／typecheck、
      Go 全包、前端 157 文件／965 项测试、Windows production 构建）
- [x] `scripts/test-windows-metadata.ps1`：应用与安装器版本资源校验通过（`2.3.1-beta.18`／`2.3.1.0`）
- [x] `scripts/verify-release-assets.ps1`：发布资产校验通过（`v2.3.1-beta.18`）
- [x] 安装器大小／SHA256 与 `.sha256` 一致（21,328,474 bytes／`d0e18021…`）
- [ ] 安装后在界面点「同步官方模型」未验证（需用户确认）
- [ ] 真实对话与计费扣减未验证（消耗积分）

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
已校验 v2.3.1-beta.18，Windows 数字版本 2.3.1.0
task --force verify   → 通过（Test Files 157 passed / Tests 965 passed）
task --force package  → 通过（NSIS，Total size 21,328,474 / 85,747,831 bytes）
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.3.1-beta.18
BUILD_EXIT=0
```

---

# Documentation

- [x] `docs/releases/v2.3.1-beta.18.md`
- [x] `docs/plans/CURRENT.md`

---

# Deviations

- 无偏离：沿用既有 Beta 构建流程。`build/windows/info.json` 由脚本重写为 CRLF，已恢复 LF；
  版本值与提交一致，本次没有提交差异。

---

# Git

## Commit

Not committed by Agent.

---

# Result

已产出 `v2.3.1-beta.18` Windows amd64 本地 Beta 生产安装包（21,328,474 bytes，
SHA256 `d0e18021c40be3dcf8810f412c8845cb1311efaa6e942064332a0b4f2660880a`），
内容为 TASK-20261116-022 的手动同步修复。未签名、未公开发布、未打标签、未推送、
未自动安装。用户从托盘完全退出 OmniProxy 后覆盖安装，再点一次「同步官方模型」应成功。

---

# Completed

2026-11-16 22:33（Asia/Singapore）