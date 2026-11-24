# TASK-20261124-002: 构建 v2.4.1-beta.7 本地安装包

## Status

In Progress

## Created

2026-11-24 10:00

## Request

用户要求：完成「提供商代理覆盖额度、目录与续期等资源调用」后，提交代码，构建新的 Beta 生产版本供本机覆盖更新，并更新任务记录。

## Goal

产出 `v2.4.1-beta.7` Windows amd64 生产安装包（未签名、本地使用），供本机覆盖更新后验证：

- 只给某个提供商配置「走代理」时，额度、余额、积分、模型目录、登录轮询与令牌续期都走该代理；
- Google 订阅账号卡不再出现「模型列表暂时不可读」「操作未完成」而实际直连；
- 推理、透明转发与既有代理语义不变。

## Background

- 版本基线 `v2.4.1-beta.6`（`2d07b624` 记录构建产物，安装器 SHA256 `0555bcf4…`）。
- 功能实现提交 `71fc1664`（TASK-20261124-001）、记录哈希 `26f42800`。
- 本版本无数据库迁移。
- `docs/releases/v2.4.1-beta.7.md` 与 `release-assets/v2.4.1-beta.7` 尚不存在。

## Scope

- `docs/releases/v2.4.1-beta.7.md` 版本说明（中英）。
- 版本号 → `2.4.1-beta.7`（`internal/version/version.go`、`frontend/package.json`）。
- `scripts/build-release.ps1 -Version v2.4.1-beta.7 -OutputDirectory release-assets\v2.4.1-beta.7`。
- 记录安装器大小、SHA256、Windows 数字版本与 NSIS 基础版本。

## Out of Scope

- 打标签、推送、公开 Release、代码签名、自动安装或启动应用。
- 桌面窗口内的实机目视检查（由用户安装后确认）。

## Existing Behavior

`v2.4.1-beta.6`：按提供商配置的代理只覆盖推理、透明转发与连通性探针；额度、余额、目录与续期依赖
`providers.base_url` 主机名回退，Google 订阅与 Codex 订阅的官方主机不在表内，显式「走代理」会被绕过。

## Expected Behavior

安装后：提供商设置为「走代理」时，该提供商的额度、模型目录、登录轮询、身份核对与令牌续期都经同一
代理端点发出；代理不可用返回 `proxy_unavailable` 等既有错误码，不再静默直连。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/config.yml`、`build/windows/info.json`、
  `build/windows/nsis/wails_tools.nsh`
- `docs/releases/v2.4.1-beta.7.md`
- `scripts/build-release.ps1`、`scripts/set-release-version.ps1`、`scripts/test-windows-metadata.ps1`、
  `scripts/verify-release-assets.ps1`

## Dependencies

NSIS、`task`、`wails3`、Go 1.26、pnpm —— 已在本机验证可用。

## Risks

- `scripts/build-release.ps1` 内部运行 `task --force verify`；当前 Windows `cmd.exe` 下该任务的
  POSIX `format:check` 配方无法解析（与本版本代码无关），本次按五个阶段逐项执行等价质量门后
  再打包，并在任务记录中说明。
- `release-assets/v2.4.1-beta.7` 必须为空或不存在。
- 仓库 `.ps1` 为 UTF-8 无 BOM，Windows PowerShell 5.1 按 ANSI 解析会失败；沿用 BOM 副本并以
  `powershell.exe -NoProfile -ExecutionPolicy Bypass -File` 运行。

---

# Implementation Plan

1. 写版本说明与版本号，`set-release-version.ps1` 同步构建资源。
2. 运行 `task --force package`（质量门已等价执行）。
3. 校验安装器、版本资源与发布资产。
4. 回填任务记录、归档、更新 CURRENT，提交文档。

---

# Progress

---

# Implementation Result

任务完成后填写。

---

# Verification

## Tests

- [ ] Unit Tests
- [ ] Integration Tests
- [ ] E2E Tests

## Quality Checks

- [ ] Format
- [ ] Lint
- [ ] Type Check
- [ ] Test
- [ ] Build
- [ ] Verify

## Verification Result

---

# Documentation

- [ ] PRD
- [ ] Architecture
- [ ] Tech Stack
- [ ] Coding Rules
- [ ] Testing
- [ ] Build
- [ ] ADR
- [ ] CURRENT

---

# Deviations

None.

---

# Git

## Commit

Not committed by Agent.

---

# Result

---

# Completed
