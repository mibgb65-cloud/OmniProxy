# TASK-20261120-009: 构建 v2.4.1-beta.5 本地安装包

## Status

In Progress

## Created

2026-11-20 20:12

## Request

用户要求：完成代理协议确认修复后，提交代码，构建新的 Beta 生产版本供本机覆盖更新，并更新任务记录。

## Goal

产出 `v2.4.1-beta.5` Windows amd64 生产安装包（未签名、本地使用），供本机覆盖更新后验证：

- 远端纯 SOCKS5 代理配成 HTTP 时，请求不再挂到超时，首次请求内自动切到 SOCKS5 并记住结果。

## Background

- 版本基线 `v2.4.1-beta.4`（`5efedf02` 记录构建产物）。
- 功能实现提交 `ab16e41c`（TASK-20261120-008）、记录哈希 `e845b69c`。
- 本版本无数据库迁移。
- `docs/releases/v2.4.1-beta.5.md` 与 `release-assets/v2.4.1-beta.5` 尚不存在。

## Scope

- `docs/releases/v2.4.1-beta.5.md` 版本说明（中英）。
- 版本号 → `2.4.1-beta.5`（`internal/version/version.go`、`frontend/package.json`）。
- `scripts/build-release.ps1 -Version v2.4.1-beta.5 -OutputDirectory release-assets\v2.4.1-beta.5`。
- 记录安装器大小、SHA256、Windows 数字版本与 NSIS 基础版本。

## Out of Scope

- 打标签、推送、公开 Release、代码签名、自动安装或启动应用。
- 桌面窗口内的实机目视检查（由用户安装后确认）。

## Existing Behavior

`v2.4.1-beta.4`：网络代理里把纯 SOCKS5 端口配成 `http` 时，提供商请求挂到调用方超时且无错误码。

## Expected Behavior

安装后：同一配置在首次请求内自愈到 SOCKS5，之后按识别结果建池；「网络代理」卡片显示实际协议。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/releases/v2.4.1-beta.5.md`
- `scripts/build-release.ps1`、`scripts/set-release-version.ps1`

## Dependencies

NSIS、`task`、`wails3`、Go 1.26、pnpm —— 已在本机验证可用。

## Risks

- 构建脚本内部运行 `task --force verify` 与 `task --force package`；构建前后核对 `git status`。
- `release-assets/v2.4.1-beta.5` 必须为空或不存在。
- 当前 shell 为 Windows PowerShell 5.1、代码页 936、`RemoteSigned` 策略，需用临时 UTF-8 BOM 副本
  + `-ExecutionPolicy Bypass` 运行构建脚本（仓库脚本不改动）。

---

# Implementation Plan

1. 写版本说明与版本号，提交 `chore(release)`。
2. 运行 `scripts/build-release.ps1`。
3. 校验安装器、版本资源与发布资产。
4. 回填任务记录、归档、更新 CURRENT，提交文档。

---

# Progress

- 2026-11-20 20:12 创建记录，版本号 → `2.4.1-beta.5`。

---

# Implementation Result

构建完成后填写。

---

# Verification

## Tests

- [ ] 构建脚本内置的 `task --force verify`
- [ ] Windows 应用与安装器版本资源校验
- [ ] 发布资产（数量、命名、SHA256）校验
- [ ] 真实安装与启动：由用户执行
- [ ] 桌面窗口目视检查：未执行

## Quality Checks

- [ ] Format
- [ ] Lint
- [ ] Type Check
- [ ] Test
- [ ] Build
- [ ] Verify

## Verification Result

构建完成后填写。

---

# Documentation

- [ ] 版本说明 `docs/releases/v2.4.1-beta.5.md`
- [ ] CURRENT
- [ ] 功能文档由 TASK-20261120-008 更新（NETWORK_PROXY）

---

# Deviations

构建完成后填写。

---

# Git

## Commit

Not committed by Agent.

---

# Result

构建完成后填写。

---

# Completed

YYYY-MM-DD HH:mm
