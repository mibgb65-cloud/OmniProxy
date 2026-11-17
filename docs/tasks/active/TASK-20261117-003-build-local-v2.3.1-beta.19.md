# TASK-20261117-003: 构建本地 v2.3.1-beta.19 生产安装包

## Status

Completed

## Created

2026-11-17 08:35（Asia/Singapore）

## Request

真实账号验证发现并修复了 WorkBuddy AI 对话被 11128 拒绝的问题（TASK-20261117-002），
用户需要在已安装的 `v2.3.1-beta.18` 上覆盖安装，用真实客户端验证对话。

## Goal

产出 `v2.3.1-beta.19` Windows amd64 本地 Beta 生产安装包，内容为对话会话形态修复
（首条 system 归一、11128 自动重试）。

## Background

`v2.3.1-beta.18` 上路由手动同步已修好；真实账号直连验证发现上游要求首条消息是 `system`，
否则 `400 code 11128 "first message is not system prompt"`。修复后经本仓库适配器实测
`200` + SSE（未见积分扣减，Free 计划按每日免费额度）。本版本无数据库迁移变化。

## Scope

- 版本号 `2.3.1-beta.19`：`internal/version/version.go`、`frontend/package.json`；
  `build/windows/info.json` 的 `ProductVersion` 同步（数字版本仍 `2.3.1.0`）。
- `docs/releases/v2.3.1-beta.19.md`。
- `scripts/build-release.ps1 -Version v2.3.1-beta.19 -OutputDirectory release-assets\v2.3.1-beta.19`。
- 校验安装器大小／SHA256、原生版本资源与发布资产；记录到 Task Record 与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不打标签、不推送、不自动安装、不启动安装后的程序、不修改用户数据。
- 不改前端；不新增迁移。

---

# Implementation Result

## Changes

- 版本号 `2.3.1-beta.19`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`；Windows 数字版本仍 `2.3.1.0`，
  NSIS 基础版本仍 `2.3.1`（脚本同步后与提交一致，无差异）。
- 新增 `docs/releases/v2.3.1-beta.19.md`。
- 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.19 -OutputDirectory release-assets\v2.3.1-beta.19`。

## Artifacts

- `release-assets/v2.3.1-beta.19/OmniProxy-Setup-v2.3.1-beta.19-windows-amd64.exe`
  **21,342,637 bytes**
  SHA256 `1b131a8081f849cb3ab9bcdc82225f6cdd29d0df645ff2cd0d839cfffa711495`
- `release-assets/v2.3.1-beta.19/OmniProxy-Setup-v2.3.1-beta.19-windows-amd64.exe.sha256`
- `release-assets/` 被 `.gitignore` 忽略。

## Technical Notes

- 相对 `v2.3.1-beta.18`（21,328,474 bytes）增加 14,163 bytes：会话形态归一、11128 重试与测试。
- 构建脚本内部的 `task --force verify` 覆盖本次修复（前端 157 files / 965 tests）。
- `set-release-version.ps1` 重写 `build/windows/info.json` 带 CRLF，已恢复 LF；版本值与提交一致。

---

# Verification

## Tests

- [x] `scripts/build-release.ps1` 内 `task --force verify` 通过（format／lint／typecheck、
      Go 全包、前端 157 文件／965 项测试、Windows production 构建）
- [x] `scripts/test-windows-metadata.ps1`：应用与安装器版本资源校验通过（`2.3.1-beta.19`／`2.3.1.0`）
- [x] `scripts/verify-release-assets.ps1`：发布资产校验通过（`v2.3.1-beta.19`）
- [x] 安装器大小／SHA256 与 `.sha256` 一致（21,342,637 bytes／`1b131a80…`）
- [ ] 安装后用真实客户端发一次对话未验证（需用户执行，建议先建一条 WorkBuddy 路由）
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
已校验 v2.3.1-beta.19，Windows 数字版本 2.3.1.0
task --force verify   → 通过（Test Files 157 passed / Tests 965 passed）
task --force package  → 通过（NSIS，Total size 21,342,637 / 85,815,299 bytes）
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.3.1-beta.19
BUILD_EXIT=0
```

---

# Documentation

- [x] `docs/releases/v2.3.1-beta.19.md`
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

已产出 `v2.3.1-beta.19` Windows amd64 本地 Beta 生产安装包（21,342,637 bytes，
SHA256 `1b131a8081f849cb3ab9bcdc82225f6cdd29d0df645ff2cd0d839cfffa711495`），
内容为 TASK-20261117-002 的对话会话形态修复。未签名、未公开发布、未打标签、未推送、
未自动安装。用户从托盘完全退出 OmniProxy 后覆盖安装，为某个 WorkBuddy 模型建一条路由
即可用真实客户端发消息验证。

---

# Completed

2026-11-17 08:40（Asia/Singapore）