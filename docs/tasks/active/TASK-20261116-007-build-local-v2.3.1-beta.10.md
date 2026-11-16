# TASK-20261116-007: 构建本地 v2.3.1-beta.10 生产安装包

## Status

Planned

## Created

2026-11-16 09:55（Asia/Singapore）

## Request

DuMate 积分接入配额页（TASK-20261116-006，`69f1e1db`）已完成，需要可覆盖安装的
本地 Beta 生产版本，供用户在本机直接更新验收。

## Goal

产出 `v2.3.1-beta.10` Windows amd64 生产安装包，内容为配额页 DuMate 积分与本机
Token 用量同卡展示；保持无数据库迁移、无公共 API 变更。

## Background

上一个本地构建为 `v2.3.1-beta.9`（TASK-20261116-005，HowToFix 余额与 Token 用量）。
本版本在本地基础上叠加 DuMate 积分接入。

## Scope

- `internal/version/version.go`、`frontend/package.json`：`2.3.1-beta.10`。
- `build/windows/info.json`：`ProductVersion` 同步为 `2.3.1-beta.10`（数字版本仍 `2.3.1.0`）。
- `docs/releases/v2.3.1-beta.10.md`：发行说明与验证边界。
- `scripts/build-release.ps1 -Version v2.3.1-beta.10 -OutputDirectory release-assets\v2.3.1-beta.10`。
- 校验安装器大小／SHA256、原生版本资源与发布资产；记录到本 Task Record 与
  `docs/plans/CURRENT.md`。

## Out of Scope

- 不打标签、不推送、不自动安装、不启动安装后的程序、不修改用户数据。

---

# Implementation Result

## Changes

- 版本号：`2.3.1-beta.9` → `2.3.1-beta.10`（`internal/version/version.go`、
  `frontend/package.json`、`build/windows/info.json` 的 `ProductVersion`）。
- 新增 `docs/releases/v2.3.1-beta.10.md`（发行说明与验证边界）。
- 运行 `scripts/build-release.ps1`：内部执行 `task --force verify` 与
  `task --force package`，产出 `bin/OmniProxy-amd64-installer.exe` 并复制到
  `release-assets/v2.3.1-beta.10/`，附带 SHA256 文件。

## Files Changed

- `internal/version/version.go`
- `frontend/package.json`
- `build/windows/info.json`
- `docs/releases/v2.3.1-beta.10.md`（新）

## Artifacts

- `release-assets/v2.3.1-beta.10/OmniProxy-Setup-v2.3.1-beta.10-windows-amd64.exe`
  21,092,215 bytes
  SHA256 `745703c7c0dd41e270eaab9ad7ce8670e8ab434872dcf6b5c3d44e0be0168632`
- `release-assets/v2.3.1-beta.10/OmniProxy-Setup-v2.3.1-beta.10-windows-amd64.exe.sha256`
- `release-assets/` 在 `.gitignore` 中，安装包不进版本控制。

## Technical Notes

- 构建脚本自身会跑一遍完整质量门，因此本任务验证实际执行两次（提交前一次、打包前一次）。
- `build/windows/info.json` 被打包工具改写成 CRLF；已恢复为 LF，使提交差异只有
  `ProductVersion` 一行。
- 本版本无数据库迁移，升级不改动账号、路由、统计与客户端配置。

---

# Verification

## Tests

- [x] `scripts/build-release.ps1` 内 `task --force verify` 通过（格式、lint、类型检查、
  全部 Go 包与前端测试、Windows production 构建）
- [x] `scripts/test-windows-metadata.ps1`：应用与安装器版本资源校验通过
- [x] `scripts/verify-release-assets.ps1`：发布资产校验通过（`v2.3.1-beta.10`）
- [x] 安装器大小／SHA256 与 `.sha256` 文件一致
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
发布资产校验通过：v2.3.1-beta.10
```

---

# Documentation

- `docs/releases/v2.3.1-beta.10.md`：新功能、不兼容变更、本地更新步骤与验证边界。
- `docs/plans/CURRENT.md`：登记本构建。

---

# Git

## Commit

```text
待回填
```

未推送、无标签、无历史改写；未自动安装、未启动安装后的程序。

---

# Result

已产出 `v2.3.1-beta.10` Windows amd64 本地 Beta 生产安装包（21,092,215 bytes，
SHA256 `745703c7c0dd41e270eaab9ad7ce8670e8ab434872dcf6b5c3d44e0be0168632`），
内容为配额页 DuMate 积分与本机 Token 用量同卡展示。未签名、未公开发布，供用户从托盘
完全退出后覆盖安装。

---

# Completed

2026-11-16 10:00（Asia/Singapore）
