# TASK-20261025-006: 构建 Beta.39 本地生产安装包

## Status

In Progress

## Created

2026-10-25 11:35（Asia/Singapore）

## Request

用户更新到 Beta.38 后发现 WebSocket 请求的运行形态归属丢失，要求修复后重新出可本地更新的版本。

## Goal

产出 `release-assets/v2.0.0-beta.39/` 安装器与 SHA256，通过完整质量门、Windows 版本资源与
发布资产校验，供用户手动覆盖更新。

## Background

相对 Beta.38 的改动是 [TASK-20261025-005](../completed/2026-10/TASK-20261025-005-websocket-client-runtime-label.md)：
WebSocket 请求记录补回 `codex-cli`／`codex-desktop` 归属（提交 `dde6dc8`）。无数据库迁移。

## Scope

- 版本同步至 `2.0.0-beta.39`，新增 `docs/releases/v2.0.0-beta.39.md`。
- 完整质量门 → 打包 → Windows 版本资源 → 发布资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md`。

## Out of Scope

- 不自动安装、不打标签、不推送、不代码签名。
- 不回溯修正历史记录的 `client_id`。

---

# Implementation Result

- 版本同步至 `2.0.0-beta.39`（`internal/version/version.go`、`frontend/package.json`，UTF-8 无 BOM），
  `scripts/set-release-version.ps1` 校验通过并同步 `build/windows/info.json` 的 `ProductVersion`
  （Windows 数字版本仍 `2.0.0.0`）。
- 新增发布说明 `docs/releases/v2.0.0-beta.39.md`。
- `release-assets/v2.0.0-beta.39/`：
  - `OmniProxy-Setup-v2.0.0-beta.39-windows-amd64.exe` 20,075,276 bytes
  - SHA256 `3a1f56ec43cfcba2e52d8b824c84e87b15db59400f78188c36b34aa317ab6dd5`
- 相对 Beta.38 的 20,073,994 bytes 增加 1,282 bytes（只含 WS 归属细化与其用例）。

# Verification

- PASS：`task --force verify` exit 0（日志 `tmp/verify-beta39.log`；格式／vet／staticcheck／ESLint／
  类型检查、全部 Go 测试、前端 128 文件／760 项测试、Windows production 构建）。
- PASS：`task --force package` exit 0（日志 `tmp/package-beta39.log`；安装器 20,075,276 bytes）。
- PASS：`scripts/test-windows-metadata.ps1 … -Version v2.0.0-beta.39` →「Windows 应用和安装器版本资源校验通过」。
- PASS：`scripts/verify-release-assets.ps1 -Version v2.0.0-beta.39 -AssetDirectory release-assets\v2.0.0-beta.39`
  →「发布资产校验通过：v2.0.0-beta.39」。
- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送、实机运行形态显示验收。

# Remaining Risks

- `scripts/build-release.ps1` 仍不能整体跑通（Taskfile 格式片段的 `uname`／`tail` 缺失 stderr），
  继续按等价步骤逐步执行。
- 打包需先把 `%ProgramFiles(x86)%\NSIS` 加入 PATH。
- 安装器未签名，Windows SmartScreen 仍可能提示。
- 不回溯修正历史记录的 `client_id`。

# Git

## Commit

待填写。

---

# Result

Beta.39 本地生产安装包已构建并校验通过（20,075,276 bytes／SHA256 `3a1f56ec…dd5`），
相对 Beta.38 只有 WebSocket 运行形态归属修复；安装器未签名、未自动安装，供用户本地覆盖更新。

---

# Completed

待填写。
