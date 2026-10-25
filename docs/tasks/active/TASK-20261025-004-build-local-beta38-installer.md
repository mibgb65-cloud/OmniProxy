# TASK-20261025-004: 构建 Beta.38 本地生产安装包

## Status

In Progress

## Created

2026-10-25 11:05（Asia/Singapore）

## Request

> 构建新的beta生产版本我进行本地更新

## Goal

产出 `release-assets/v2.0.0-beta.38/` 安装器与 SHA256，通过完整质量门、Windows 版本资源与
发布资产校验，供用户手动覆盖更新。

## Background

相对 Beta.37 的改动是本次两个已归档任务：

- [TASK-20261025-002](../completed/2026-10/TASK-20261025-002-quota-estimate-unknown-usage-tolerance.md)：
  额度美元估算不再被零星「成功但无 usage」的记录整窗否决（未知占比 ≤ 20% 仍出金额）。
- [TASK-20261025-003](../completed/2026-10/TASK-20261025-003-passthrough-websocket-turn-accounting.md)：
  透明转发 WebSocket 按轮记录请求、账号尝试与 Token 用量。

无数据库迁移，无新增依赖。

## Scope

- 版本同步至 `2.0.0-beta.38`，新增 `docs/releases/v2.0.0-beta.38.md`。
- 完整质量门 → 打包 → Windows 版本资源 → 发布资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md` 的当前版本段落。

## Out of Scope

- 不自动安装、不打标签、不推送、不代码签名。
- 不发送真实订阅账号请求；不验收实机 Codex WebSocket 会话与原生窗口视觉。

---

# Implementation Plan

1. `internal/version/version.go`、`frontend/package.json` 同步到 `2.0.0-beta.38`。
2. `scripts/set-release-version.ps1 -Version v2.0.0-beta.38` 同步 `build/config.yml`、
   `build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`。
3. `task --force verify` → `task --force package`（NSIS 需在 PATH）。
4. `scripts/test-windows-metadata.ps1` 校验应用与安装器版本资源。
5. 安装器复制为 `release-assets/v2.0.0-beta.38/OmniProxy-Setup-v2.0.0-beta.38-windows-amd64.exe`
   并生成 `.sha256`，跑 `scripts/verify-release-assets.ps1`。

---

# Progress

- 起始版本确认：源码／前端／Windows 元数据均在 `2.0.0-beta.37` 状态。

---

# Implementation Result

- 版本同步至 `2.0.0-beta.38`（`internal/version/version.go`、`frontend/package.json`，
  写入时保持 UTF-8 无 BOM 与 LF），`scripts/set-release-version.ps1` 校验通过并同步
  `build/windows/info.json` 的 `ProductVersion`（Windows 数字版本保持 `2.0.0.0`；
  `build/config.yml`、`wails_tools.nsh` 无需变更）。
- 新增发布说明 `docs/releases/v2.0.0-beta.38.md`。
- `release-assets/v2.0.0-beta.38/`：
  - `OmniProxy-Setup-v2.0.0-beta.38-windows-amd64.exe` 20,073,994 bytes
  - SHA256 `40a60aa74a22159c0a2ee950644e6688bf2ccd0c3ab5832525bf1f872efcf826`
- 相对 Beta.37 的 20,063,983 bytes 增加 10,011 bytes（只含额度估算容忍与透明 WebSocket 逐轮记账）。

# Verification

- PASS：`task --force verify` exit 0（日志 `tmp/verify-beta38.log`；格式／vet／staticcheck／ESLint／
  类型检查、全部 Go 测试、前端 128 文件／760 项测试、Windows production 构建）。
  用 `cmd /c` 取得真实退出码（pwsh 管道会把 `uname`／`tail` 缺失的 stderr 当 NativeCommandError）。
- PASS：`task --force package` exit 0（日志 `tmp/package-beta38.log`；NSIS 编译通过，
  安装器 20,073,994 bytes）。
- PASS：`scripts/test-windows-metadata.ps1 -Application bin\omniproxy.exe -Installer bin\OmniProxy-amd64-installer.exe -Version v2.0.0-beta.38`
  →「Windows 应用和安装器版本资源校验通过」。
- PASS：`scripts/verify-release-assets.ps1 -Version v2.0.0-beta.38 -AssetDirectory release-assets\v2.0.0-beta.38`
  →「发布资产校验通过：v2.0.0-beta.38」。
- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送、真实向导点击验收、
  实机 Codex WebSocket 会话与额度估算显示验收。

# Remaining Risks

- `scripts/build-release.ps1` 仍不能整体跑通（`$ErrorActionPreference='Stop'` 会把 Taskfile
  格式片段打印的 `uname`／`tail` 缺失 stderr 当成终止错误），本轮继续按 Beta.34～37 的等价步骤
  逐步执行（verify → package → 版本资源 → 资产校验）。
- 打包需先把 `%ProgramFiles(x86)%\NSIS` 加入 PATH。
- 安装器未签名，Windows SmartScreen 仍可能提示；未自动安装、未打标签、未推送。
- 估算与 WebSocket 记账的实际显示效果需要用户在本机覆盖安装后验收。

# Git

## Commit

待填写。

---

# Result

Beta.38 本地生产安装包已构建并校验通过（20,073,994 bytes／SHA256 `40a60aa7…f826`），
相对 Beta.37 只有额度估算容忍与透明 WebSocket 逐轮记账两项；安装器未签名、未自动安装，
供用户本地覆盖更新。

---

# Completed

待填写。
