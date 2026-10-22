# TASK-20261023-010: 构建 Beta.25 本地生产安装包

## Status

In Progress

## Created

2026-10-23 22:30

## Request

用户要求构建新的 Beta 生产版本，供其在本机覆盖更新。

## Goal

基于当前提交 `65d81d3` 及工作区现状产出 `release-assets/v2.0.0-beta.25/` 安装器与 SHA256，并通过完整质量门、Windows 版本资源与发布资产校验。

## Scope

- 版本同步至 `2.0.0-beta.25`：`internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`（仅 `ProductVersion` 一行）。
- 新增 `docs/releases/v2.0.0-beta.25.md`。
- 完整质量门、NSIS 打包、Windows 元数据、SHA256、资产验证。

## Out of Scope

- 不提交工作区中其他未提交改动（TASK-20261022-006 前端改动随之打包，但不代为提交）。
- 不自动安装、不停止或替换正在运行的生产进程、不打标签、不推送。

## Background

Beta.24 已包含图片能力与推理档位修复。本次 Beta.25 在此基础上加入：

- TASK-20261023-009（已提交 `211a98f`）：配额页对没有官方额度接口的账号显示本机已用 Token；Token 用量悬浮改用统一 Tooltip；迁移 040 与请求 Agent 归属。
- TASK-20261022-006（工作区未提交）：用量卡片细圆角滚动条、本地 Agent 分组卡片「一键配置」。

---

# Implementation Plan

1. 版本同步至 `2.0.0-beta.25`（源码、前端、Windows 资源）。
2. 撰写 `docs/releases/v2.0.0-beta.25.md`。
3. 依次执行 `task --force verify` → `task --force package` → `scripts/test-windows-metadata.ps1` → 资产复制与 SHA256 → `scripts/verify-release-assets.ps1`。
4. 记录结果并提交。

---

# Progress

- 2026-10-23 22:30 建立任务记录，确认源码版本仍为 `2.0.0-beta.24`、`release-assets/v2.0.0-beta.24/` 已存在（不覆盖）。
- 2026-10-23 22:40 版本同步至 `2.0.0-beta.25`（源码／前端／Windows `ProductVersion`），新增发布说明。
- 2026-10-23 22:50 `task --force verify` exit 0（123 文件 / 731 项前端测试、Go 全部包、Windows production 构建）。
- 2026-10-23 22:55 首次 `task --force package` 失败：`makensis` 不在 PATH（exit 127）。按 `build-release.ps1` 的做法把 `%ProgramFiles(x86)%\NSIS` 加入本次 PATH 后打包成功（makensis 3.12）。
- 2026-10-23 23:00 版本资源校验、资产复制与 SHA256、资产校验全部通过。

---

# Implementation Result

## Changes

- 版本同步至 `2.0.0-beta.25`：`internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`（仅 `0409.ProductVersion` 一行；Windows 数字版本维持 `2.0.0.0`，`build/config.yml` 与 NSIS 模板无需改动）。
- 新增 `docs/releases/v2.0.0-beta.25.md`，覆盖本次 Agent 归属、配额本机用量、统一悬浮提示，以及随工作区一并打包的滚动条／分组「一键配置」。
- 产出 `release-assets/v2.0.0-beta.25/` 两个文件；20 个发布目录各 2 个文件，Beta.6–Beta.24 历史资产未变。

## Files Changed

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `docs/releases/v2.0.0-beta.25.md`（新增）
- `docs/plans/CURRENT.md`、`docs/tasks/active/TASK-20261023-010-build-local-beta25-installer.md`

## Technical Notes

按用户要求以工作区现状构建：包含已提交的 `65d81d3`（TASK-20261023-009）以及仍未提交的 TASK-20261022-006 前端改动（用量卡片滚动条、本地 Agent 分组「一键配置」）。代码本身未由本任务提交。

---

# Verification

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

- PASS：`task --force verify` exit 0 —— 格式 → lint → 类型 → Go 全套 → 123 文件／731 项前端测试 → Windows production 构建。
- PASS：`task --force package` exit 0（makensis 3.12，安装器总大小 19,588,499 bytes）。
- PASS：`scripts/test-windows-metadata.ps1 -Version v2.0.0-beta.25` 输出「Windows 应用和安装器版本资源校验通过」（应用与安装器均经原生版本 API 核对）。
- PASS：`scripts/verify-release-assets.ps1 -Version v2.0.0-beta.25` 输出「发布资产校验通过」。
- PASS：安装器 `OmniProxy-Setup-v2.0.0-beta.25-windows-amd64.exe` 19,588,499 bytes，SHA256 `639fbf77410abdc37d65e74512a907a1a38232fa37099c05f349841fd79277b8`。
- PASS：`build/windows/info.json` diff 仅 `ProductVersion` 一行（`2.0.0-beta.21` → `2.0.0-beta.25`，Beta.22–24 的该行此前未提交）；`gofmt -l` 无输出。
- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送。真实覆盖安装、数据保留、Agent 归属与界面表现由用户本地验收。

---

# Documentation

- [x] `docs/releases/v2.0.0-beta.25.md`
- [x] `docs/plans/CURRENT.md`（当前版本条目）

---

# Deviations

- `scripts/build-release.ps1` 未整体执行（沿用 TASK-20261023-008 的结论：本机 PATH 缺 `uname`，`task` 把 stderr 当终止错误），改为逐项执行同一组步骤，全部返回 0，未修改脚本。
- `set-release-version.ps1` 只校验版本一致性并改写 Windows 资源；本次直接手工同步三个版本文件后执行后续步骤，`build/config.yml` 与 `wails_tools.nsh` 数字版本未变，避免 `ConvertTo-Json` 重排 `info.json`。

---

# Git

## Commit

见下方哈希记录。

---

# Result

Beta.25 production 本地安装包已构建，通过完整质量门、Windows 版本资源与发布资产校验。未签名、未自动安装、未打标签、未推送，供用户本地覆盖更新。

---

# Remaining Risks

- 工作区仍含未提交的 TASK-20261022-006 前端改动，本安装包的来源不能完全对应某次提交。
- 安装包未签名，Windows 会提示未知发布者。
- 升级后需重新应用客户端配置并完全退出重启，否则客户端可能沿用缓存菜单。
- 请求 Agent 归属只对新请求生效；历史记录显示「—」。
- 真实覆盖安装、数据保留与界面表现未经本任务验证。

