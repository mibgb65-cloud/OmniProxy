# TASK-20261117-005: 构建本地 v2.3.1-beta.20 生产安装包

## Status

Completed

## Created

2026-11-17 09:50（Asia/Singapore）

## Request

用户问「他们的模型都没有推理等级的吗？」并需要在已安装的 `v2.3.1-beta.19` 上覆盖安装，
验证 WorkBuddy AI 模型档位。

## Goal

产出 `v2.3.1-beta.20` Windows amd64 本地 Beta 生产安装包，内容为 TASK-20261117-004 的
推理档位补全（只给 `effort` 的模型补 `low/high/max` 加自身档位）。

## Background

`v2.3.1-beta.19` 上对话链路已通（首条 system + 11128 重试）。真实 `/v3/config` 显示 4 个模型
（`max-model`、`gemini-4-carbon`、`gemini-4-argon`、`kimi-k3`）只有 `reasoning.effort`，
目录侧因此没有档位；TASK-20261117-004 已按社区插件实测口径补齐。本版本无数据库迁移。

## Scope

- 版本号 `2.3.1-beta.20`：`internal/version/version.go`、`frontend/package.json`；
  `build/windows/info.json` 的 `ProductVersion` 同步（数字版本仍 `2.3.1.0`）。
- `docs/releases/v2.3.1-beta.20.md`。
- `scripts/build-release.ps1 -Version v2.3.1-beta.20 -OutputDirectory release-assets\v2.3.1-beta.20`。
- 校验安装器大小／SHA256、原生版本资源与发布资产；记录到 Task Record 与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不打标签、不推送、不自动安装、不启动安装后的程序、不修改用户数据。
- 不改前端；不新增迁移。

---

# Implementation Result

## Changes

- 版本号 `2.3.1-beta.20`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`；Windows 数字版本仍 `2.3.1.0`，
  NSIS 基础版本仍 `2.3.1`。
- 新增 `docs/releases/v2.3.1-beta.20.md`。
- 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.20 -OutputDirectory release-assets\v2.3.1-beta.20`。

## Artifacts

- `release-assets/v2.3.1-beta.20/OmniProxy-Setup-v2.3.1-beta.20-windows-amd64.exe`
  **21,346,879 bytes**
  SHA256 `48cd41fa2364b469360cc784cb58471ab9c1e6d15e4d75d49c87b151558c0a23`
- `release-assets/v2.3.1-beta.20/OmniProxy-Setup-v2.3.1-beta.20-windows-amd64.exe.sha256`
- `release-assets/` 被 `.gitignore` 忽略。

## Technical Notes

- 相对 `v2.3.1-beta.19`（21,342,637 bytes）增加 4,242 bytes：档位补全、排序辅助与两条目录测试。
- 构建脚本内部的 `task --force verify` 覆盖本次修复（前端 157 files / 965 tests）。
- `set-release-version.ps1` 重写 `build/windows/info.json` 带 CRLF，已恢复 LF；版本值与提交一致。

---

# Verification

## Tests

- [x] `scripts/build-release.ps1` 内 `task --force verify` 通过（format／lint／typecheck、
      Go 全包、前端 157 文件／965 项测试、Windows production 构建）
- [x] `scripts/test-windows-metadata.ps1`：应用与安装器版本资源校验通过（`2.3.1-beta.20`／`2.3.1.0`）
- [x] `scripts/verify-release-assets.ps1`：发布资产校验通过（`v2.3.1-beta.20`）
- [x] 安装器大小／SHA256 与 `.sha256` 一致（21,346,879 bytes／`48cd41fa…`）
- [ ] 安装后在「路由 → 模型目录」同步一次并确认档位未验证（需用户执行）
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
已校验 v2.3.1-beta.20，Windows 数字版本 2.3.1.0
task --force verify   → 通过（Test Files 157 passed / Tests 965 passed）
task --force package  → 通过（NSIS，Total size 21,346,879 / 85,842,293 bytes）
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.3.1-beta.20
BUILD_EXIT=0
```

---

# Documentation

- [x] `docs/releases/v2.3.1-beta.20.md`
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

已产出 `v2.3.1-beta.20` Windows amd64 本地 Beta 生产安装包（21,346,879 bytes，
SHA256 `48cd41fa2364b469360cc784cb58471ab9c1e6d15e4d75d49c87b151558c0a23`），
内容为 TASK-20261117-004 的推理档位补全。未签名、未公开发布、未打标签、未推送、未自动安装。
用户从托盘完全退出 OmniProxy 后覆盖安装，然后在「路由 → 模型目录」点一次**同步**，
即可在模型目录与网关的模型档位选择器里看到补齐后的档位。

---

# Completed

2026-11-17 09:57（Asia/Singapore）
