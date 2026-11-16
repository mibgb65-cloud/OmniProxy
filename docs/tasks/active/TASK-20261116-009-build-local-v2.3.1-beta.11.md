# TASK-20261116-009: 构建本地 v2.3.1-beta.11 生产安装包

## Status

In Progress

## Created

2026-11-16 11:50（Asia/Singapore）

## Request

Codex 桌面端 + River `DeepSeek-V4.1-Flash-1017` 的流中断修复（TASK-20261116-008，
`69c372b1`）已完成并通过质量门，需要可覆盖安装的本地 Beta 生产版本，供用户在本机直接
更新验收。

## Goal

产出 `v2.3.1-beta.11` Windows amd64 生产安装包，内容为 Codex 入口移除孤儿工具结果、
`response.incomplete` 终结事件与 `incomplete_stream` 分类；保持无数据库迁移、无公共 API 变更。

## Background

上一个本地构建为 `v2.3.1-beta.10`（TASK-20261116-007，DuMate 积分）。本版本在本地基础上
叠加 TASK-20261116-008 的网关修复；用户当前运行 `v2.3.1-beta.10`，需覆盖安装后重启 Codex 复测。

## Scope

- `internal/version/version.go`、`frontend/package.json`：`2.3.1-beta.11`（已在修复任务中改好）。
- `build/windows/info.json`：`ProductVersion` 同步为 `2.3.1-beta.11`（数字版本仍 `2.3.1.0`）。
- `docs/releases/v2.3.1-beta.11.md`：发行说明与验证边界（已在修复任务中新增）。
- `scripts/build-release.ps1 -Version v2.3.1-beta.11 -OutputDirectory release-assets\v2.3.1-beta.11`。
- 校验安装器大小／SHA256、原生版本资源与发布资产；记录到本 Task Record 与
  `docs/plans/CURRENT.md`。

## Out of Scope

- 不打标签、不推送、不自动安装、不启动安装后的程序、不修改用户数据。

---

# Implementation Result

## Changes

- 版本号：`2.3.1-beta.10` → `2.3.1-beta.11`（`internal/version/version.go`、
  `frontend/package.json` 已在修复任务中改好；`build/windows/info.json` 的
  `ProductVersion` 由构建脚本同步，数字版本仍 `2.3.1.0`）。
- 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.11 -OutputDirectory release-assets\v2.3.1-beta.11`：
  内部执行 `task --force verify` 与 `task --force package`，产出
  `bin/OmniProxy-amd64-installer.exe` 并复制到发布目录，附带 SHA256 文件。

## Files Changed

- `build/windows/info.json`（`ProductVersion`）
- `docs/releases/v2.3.1-beta.11.md`（修复任务中新增，本任务沿用）

## Artifacts

- `release-assets/v2.3.1-beta.11/OmniProxy-Setup-v2.3.1-beta.11-windows-amd64.exe`
  21,096,005 bytes
  SHA256 `563e0f2cbfe242c2cb374f6266b01b1b2d72e58ee00f8dd8517ee76a58c5c3ff`
- `release-assets/v2.3.1-beta.11/OmniProxy-Setup-v2.3.1-beta.11-windows-amd64.exe.sha256`
- `release-assets/` 在 `.gitignore` 中，安装包不进版本控制。

## Technical Notes

- 构建脚本自身会跑一遍完整质量门，因此本任务验证实际执行两次（修复任务提交前一次、打包前一次）。
- `set-release-version.ps1` 用 `ConvertTo-Json` 重写 `build/windows/info.json` 时会带 CRLF；
  已恢复为 LF，使提交差异只有 `ProductVersion` 一行。
- 本版本无数据库迁移，升级不改动账号、路由、统计与客户端配置；安装后需重启 Codex。

---

# Verification

## Tests

- [x] `scripts/build-release.ps1` 内 `task --force verify`
- [x] `scripts/test-windows-metadata.ps1`
- [x] `scripts/verify-release-assets.ps1`
- [x] 安装器大小／SHA256 与 `.sha256` 一致
- [ ] 安装与原生窗口验收未执行（需用户自行覆盖安装后确认）

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
build-release.ps1 EXIT=0
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.3.1-beta.11
```

---

# Documentation

- [x] `docs/releases/v2.3.1-beta.11.md`
- [x] `docs/plans/CURRENT.md`

---

# Git

## Commit

```text
Not committed by Agent.
```

---

# Result

已产出 `v2.3.1-beta.11` Windows amd64 本地 Beta 生产安装包（21,096,005 bytes，
SHA256 `563e0f2cbfe242c2cb374f6266b01b1b2d72e58ee00f8dd8517ee76a58c5c3ff`），
内容为 TASK-20261116-008 的 Codex/River 流中断修复。未签名、未公开发布，供用户从托盘
完全退出后覆盖安装，安装后重启 Codex 复测。

---

# Completed

2026-11-16 11:58（Asia/Singapore）
