# TASK-20261121-001: 构建 v2.4.1-beta.6 本地安装包

## Status

Planned

## Created

2026-11-21 11:05

## Request

用户要求：完成 Antigravity 提前响应头修复后，提交代码，构建新的 Beta 生产版本供本机覆盖更新，并更新任务记录。

## Goal

产出 `v2.4.1-beta.6` Windows amd64 生产安装包（未签名、本地使用），供本机覆盖更新后验证：

- Antigravity 桌面端等待上游响应头超过 60 秒时不再出现「已取消／未报告」；
- 网关在受理 3 秒后先下发 `200` 与 SSE 响应头，并按 10 秒心跳保活；
- 其它客户端的错误状态码语义不变。

## Background

- 版本基线 `v2.4.1-beta.5`（`82171576` 记录构建产物）。
- 功能实现提交 `ac4b6c06`（TASK-20261120-010）、记录哈希 `2ca280e4`。
- 本版本无数据库迁移。
- `docs/releases/v2.4.1-beta.6.md` 与 `release-assets/v2.4.1-beta.6` 尚不存在。

## Scope

- `docs/releases/v2.4.1-beta.6.md` 版本说明（中英）。
- 版本号 → `2.4.1-beta.6`（`internal/version/version.go`、`frontend/package.json`）。
- `scripts/build-release.ps1 -Version v2.4.1-beta.6 -OutputDirectory release-assets\v2.4.1-beta.6`。
- 记录安装器大小、SHA256、Windows 数字版本与 NSIS 基础版本。

## Out of Scope

- 打标签、推送、公开 Release、代码签名、自动安装或启动应用。
- 桌面窗口内的实机目视检查（由用户安装后确认）。

## Existing Behavior

`v2.4.1-beta.5`：Antigravity 的流式请求要等 provider 拿到上游响应头后才可能写响应；上游超过 60 秒不回响应头时，桌面端按响应头超时断开，请求页出现「已取消／未报告」，4 秒后重发的同一请求通常成功。

## Expected Behavior

安装后：Antigravity 请求在受理 3 秒后就能看到 `200` 与 SSE 响应头，思考期间每 10 秒收到 `: keep-alive`；上游响应头到达后正常转发，不再触发 60 秒响应头超时。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/releases/v2.4.1-beta.6.md`
- `scripts/build-release.ps1`、`scripts/set-release-version.ps1`、`scripts/test-windows-metadata.ps1`、`scripts/verify-release-assets.ps1`

## Dependencies

NSIS、`task`、`wails3`、Go 1.26、pnpm —— 已在本机验证可用。

## Risks

- 构建脚本内部运行 `task --force verify` 与 `task --force package`；构建前后核对 `git status`。
- `release-assets/v2.4.1-beta.6` 必须为空或不存在。
- Windows PowerShell 5.1 按 ANSI 解析 UTF-8 无 BOM 的仓库脚本，含中文 `throw` 文本时会报语法错误；
  沿用上一版本的 BOM 副本方案（临时目录 `scripts-bom/`，保持 `$PSScriptRoot` 推导的仓库根不变）。

---

# Implementation Plan

1. 写版本说明与版本号，提交 `chore(release)`。
2. 运行 `scripts/build-release.ps1`（BOM 副本）。
3. 校验安装器、版本资源与发布资产。
4. 回填任务记录、归档、更新 CURRENT，提交文档。

---

# Progress

- 2026-11-21 11:05 创建记录。
