# TASK-20261116-013: 构建本地 v2.3.1-beta.13 生产安装包

## Status

Completed

## Created

2026-11-16 15:41（Asia/Singapore）

## Request

Codex `view_image` 工具结果内联图片被本地误拒（TASK-20261116-012）的修复已完成，需要可覆盖
安装的本地 Beta 生产版本供用户验收。

## Goal

产出 `v2.3.1-beta.13` Windows amd64 生产安装包，内容为 River Responses 图片校验按形式分档；
保持无数据库迁移、无公共 API 变更。

## Background

上一个本地构建为 `v2.3.1-beta.12`（TASK-20261116-011）。用户安装后反馈发图报
`400 unsupported_content`（请求 `d91090a0b2e8567ff9697c1329edcc6c`）。

## Scope

- 版本号 `2.3.1-beta.13`（`internal/version/version.go`、`frontend/package.json` 已在修复任务改好）。
- `build/windows/info.json`：`ProductVersion` 同步（数字版本仍 `2.3.1.0`）。
- `docs/releases/v2.3.1-beta.13.md`（修复任务中新增）。
- `scripts/build-release.ps1 -Version v2.3.1-beta.13 -OutputDirectory release-assets\v2.3.1-beta.13`。
- 校验安装器大小／SHA256、原生版本资源与发布资产；记录到本 Task Record 与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不打标签、不推送、不自动安装、不启动安装后的程序、不修改用户数据。

---

# Implementation Result

## Changes

- 版本号 `2.3.1-beta.13`（`internal/version/version.go`、`frontend/package.json` 由修复任务改好，
  `build/windows/info.json` 的 `ProductVersion` 由构建脚本同步，数字版本仍 `2.3.1.0`）。
- 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.13 -OutputDirectory release-assets\v2.3.1-beta.13`：
  内部执行 `task --force verify` 与 `task --force package`，产出
  `bin/OmniProxy-amd64-installer.exe` 并复制到发布目录，附带 SHA256 文件。

## Files Changed

- `build/windows/info.json`（`ProductVersion`）

## Artifacts

- `release-assets/v2.3.1-beta.13/OmniProxy-Setup-v2.3.1-beta.13-windows-amd64.exe`
  21,098,055 bytes
  SHA256 `3064e84930a0f63ea8c7658045a09d9c34766849b75b524df01dba3ecdac3b23`
- `release-assets/v2.3.1-beta.13/OmniProxy-Setup-v2.3.1-beta.13-windows-amd64.exe.sha256`
- `release-assets/` 在 `.gitignore` 中，安装包不进版本控制。

## Technical Notes

- 构建脚本自身会跑一遍完整质量门（格式、lint、类型检查、155 文件／948 项前端测试、
  全部 Go 包、Windows production 构建），因此本任务验证实际执行两次。
- `set-release-version.ps1` 重写 `build/windows/info.json` 时会带 CRLF；已恢复为 LF，
  提交差异只有 `ProductVersion` 一行。
- 本版本无数据库迁移；覆盖安装后需重启 Codex。

---

# Verification

## Tests

- [x] `scripts/build-release.ps1` 内 `task --force verify` 通过（格式、lint、类型检查、
  155 文件／948 项前端测试、全部 Go 包与 Windows production 构建）
- [x] `scripts/test-windows-metadata.ps1`：应用与安装器版本资源校验通过
- [x] `scripts/verify-release-assets.ps1`：发布资产校验通过（`v2.3.1-beta.13`）
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
发布资产校验通过：v2.3.1-beta.13
```

---

# Documentation

- [x] `docs/releases/v2.3.1-beta.13.md`
- [x] `docs/plans/CURRENT.md`

---

# Git

## Commit

```text
Not committed by Agent.
```

---

# Result

已产出 `v2.3.1-beta.13` Windows amd64 本地 Beta 生产安装包（21,098,055 bytes，
SHA256 `3064e84930a0f63ea8c7658045a09d9c34766849b75b524df01dba3ecdac3b23`），
内容为 TASK-20261116-012 的 River 响应内联图片校验修复。未签名、未公开发布，供用户从托盘
完全退出后覆盖安装，安装后重启 Codex 复测。

---

# Completed

2026-11-16 15:53（Asia/Singapore）
