# TASK-20261117-008: 构建本地 v2.3.1-beta.21 生产安装包

## Status

Planned

## Created

2026-11-17 16:34（Asia/Singapore）

## Request

WorkBuddy 国内版接入完成（TASK-20261117-007），用户需要在已安装的 `v2.3.1-beta.20`
上覆盖安装，用真实国内版账号验证登录、目录与对话。

## Goal

产出 `v2.3.1-beta.21` Windows amd64 本地 Beta 生产安装包，内容为 WorkBuddy 国内版内置服务商
（`builtin_workbuddy`、迁移 052、Claude 模型 Messages 原生路由）。

## Background

`v2.3.1-beta.20` 已包含 WorkBuddy 推理档位补全；本轮新增国内版构建接入，`task verify` 已通过
（Go 全包、前端 158 文件／967 项测试）。本版本含一条只追加迁移（052），首次启动自动执行。

## Scope

- 版本号 `2.3.1-beta.21`：`internal/version/version.go`、`frontend/package.json`；
  `build/windows/info.json` 的 `ProductVersion` 同步（数字版本仍 `2.3.1.0`）。
- `docs/releases/v2.3.1-beta.21.md`。
- `scripts/build-release.ps1 -Version v2.3.1-beta.21 -OutputDirectory release-assets\v2.3.1-beta.21`。
- 校验安装器大小／SHA256、原生版本资源与发布资产；记录到 Task Record 与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不打标签、不推送、不自动安装、不启动安装后的程序、不修改用户数据。
- 不改前端；不新增迁移。

---

# Implementation Result

## Changes

- 版本号 `2.3.1-beta.21`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`；Windows 数字版本仍 `2.3.1.0`，
  NSIS 基础版本仍 `2.3.1`。
- 新增 `docs/releases/v2.3.1-beta.21.md`。
- 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.21 -OutputDirectory release-assets\v2.3.1-beta.21`。

## Artifacts

- `release-assets/v2.3.1-beta.21/OmniProxy-Setup-v2.3.1-beta.21-windows-amd64.exe`
  **21,372,038 bytes**
  SHA256 `e790b40042c39c4ff1c45a8383768a53a9dcfd5c80fa50202b6ff43f00c5a4e4`
- `release-assets/v2.3.1-beta.21/OmniProxy-Setup-v2.3.1-beta.21-windows-amd64.exe.sha256`
- `release-assets/` 被 `.gitignore` 忽略。

## Technical Notes

- 相对 `v2.3.1-beta.20`（21,346,879 bytes）增加 25,159 bytes：国内版适配器参数化、
  Messages 原生转发、provider/前端接线与测试。
- 构建脚本内部的 `task --force verify` 覆盖本次实现（前端 158 files / 967 tests）。
- 含一条只追加迁移 052（插入 `builtin_workbuddy` 提供商行），覆盖安装后首次启动执行。
- `set-release-version.ps1` 重写 `build/windows/info.json` 带 CRLF，已恢复 LF；版本值与提交一致。

---

# Verification

## Tests

- [x] `scripts/build-release.ps1` 内 `task --force verify` 通过（format／lint／typecheck、
      Go 全包、前端 158 文件／967 项测试、Windows production 构建）
- [x] `scripts/test-windows-metadata.ps1`：应用与安装器版本资源校验通过（`2.3.1-beta.21`／`2.3.1.0`）
- [x] `scripts/verify-release-assets.ps1`：发布资产校验通过（`v2.3.1-beta.21`）
- [x] 安装器大小／SHA256 与 `.sha256` 一致（21,372,038 bytes／`e790b400…`）
- [ ] 安装后国内版登录、目录与一次真实对话未验证（需用户执行）
- [ ] 付费账号的积分扣减未验证

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
已校验 v2.3.1-beta.21，Windows 数字版本 2.3.1.0
task --force verify   → 通过（Test Files 158 passed / Tests 967 passed）
task --force package  → 通过（NSIS，Total size 21,372,038 / 85,968,014 bytes）
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.3.1-beta.21
BUILD_EXIT=0
```

---

# Documentation

- [x] `docs/releases/v2.3.1-beta.21.md`
- [x] `docs/plans/CURRENT.md`

---

# Deviations

- 无偏离：沿用既有 Beta 构建流程。`build/windows/info.json` 由脚本重写为 CRLF，已恢复 LF；
  版本值与提交一致，本次没有因构建产生的额外差异。

---

# Git

## Commit

TBD

---

# Result

已产出 `v2.3.1-beta.21` Windows amd64 本地 Beta 生产安装包（21,372,038 bytes，
SHA256 `e790b40042c39c4ff1c45a8383768a53a9dcfd5c80fa50202b6ff43f00c5a4e4`），内容为
WorkBuddy 国内版接入。未签名、未公开发布、未打标签、未推送、未自动安装。
用户从托盘完全退出 OmniProxy 后覆盖安装；首次启动执行迁移 052，服务商页出现
「WorkBuddy（国内版）」。

---

# Completed

2026-11-17 16:40（Asia/Singapore）
