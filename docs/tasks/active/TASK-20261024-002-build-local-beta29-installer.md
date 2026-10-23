# TASK-20261024-002: 构建 Beta.29 本地生产安装包

## Status

In Progress（版本准备与打包进行中）

## Created

2026-10-24 01:40（Asia/Singapore）

## Request

用户要求：构建新的 Beta 生产版本，供其本机覆盖更新。

## Goal

产出 `release-assets/v2.0.0-beta.29/` 安装器与 SHA256，并通过完整质量门、Windows 版本资源与发布资产校验，供用户本地覆盖安装。

## Background

- 当前生产安装为 Beta.28（`release-assets/v2.0.0-beta.28/`，19,626,700 bytes，SHA256 `8ef4cd32…0af269db`）。
- Beta.28 之后已提交的差异只有任务记录提交（`d89e866`），没有功能提交。
- 工作区包含两项**已完成但未提交**的任务改动，本版一并打包：
  - TASK-20261024-001：每个受管 Agent 改为固定入口前缀，Codex 恢复 `/codex/v1` 并保留官方登录（本版主变更）。
  - TASK-20261023-003／005：River 图片输入（含 Claude Messages／Responses 转换）与受管客户端目录的图片能力声明。
- 沿用 Beta.26–Beta.28 的做法：不代为提交其他任务的改动，但把它们打进包内。

## Scope

- 版本同步至 `2.0.0-beta.29`：`internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`（仅 `0409.ProductVersion`）。
- 新增 `docs/releases/v2.0.0-beta.29.md`。
- 完整质量门、NSIS 打包、Windows 版本资源、SHA256 与发布资产校验。
- 更新 `docs/plans/CURRENT.md`（当前版本条目与 In Progress）。

## Out of Scope

- 不提交工作区中其他在办任务的改动（随包打包，但不代为提交）。
- 不自动安装、不停止或替换正在运行的生产进程、不打标签、不推送。
- 不改 `build/config.yml` 与 `wails_tools.nsh`：数字版本仍为 `2.0.0`。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `Taskfile.yml`（`verify`／`package`）、`scripts/test-windows-metadata.ps1`、`scripts/verify-release-assets.ps1`

## Dependencies

无新增依赖；NSIS（`C:\Program Files (x86)\NSIS\makensis.exe`）需临时加入 PATH。

## Risks

- 安装包未签名，Windows 会提示未知发布者。
- 打包来源包含未提交的其他任务改动，不完全对应单一提交（与 Beta.26–Beta.28 相同）。
- 本版更换客户端入口地址：已安装的客户配置在重新「预览 → 应用」前仍使用旧 `/v1` 或 `/agent/codex/<Key>/v1`，旧地址保留可用，不会立即失效。
- 未执行真实覆盖安装；版本读取、数据保留与客户端重连由用户本地验收。

---

# Implementation Plan

1. 版本同步至 `2.0.0-beta.29`，撰写发布说明。
2. `task --force verify` → `task --force package`（NSIS 加入 PATH）。
3. `scripts/test-windows-metadata.ps1` → 复制资产与 SHA256 → `scripts/verify-release-assets.ps1`。
4. 更新 `docs/plans/CURRENT.md` 与本记录，提交版本准备。

---

# Progress

- 版本同步至 `2.0.0-beta.29`（源码／前端／Windows `ProductVersion`），新增发布说明；`build/config.yml` 与 `wails_tools.nsh` 的数字版本仍为 `2.0.0`，未改。
- `task --force verify` exit 0（126 文件 / 750 项前端测试、Go 全部包、Windows production 构建，日志 `tmp/verify-beta29.log`）。
- `task --force package` exit 0（安装器 19,636,234 bytes，日志 `tmp/package-beta29.log`），NSIS 目录临时加入 PATH。
- 版本资源校验、资产复制与 SHA256、资产校验全部通过；Beta.6–Beta.28 共 23 个历史发布目录仍各 2 个文件，新增后共 24 个。

---

# Implementation Result

## Changes

- 版本同步至 `2.0.0-beta.29`：`internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`（仅 `0409.ProductVersion`；Windows 数字版本维持 `2.0.0.0`）。
- 新增 `docs/releases/v2.0.0-beta.29.md`：说明固定入口前缀、Codex `/codex/v1` 与官方登录保留、`403 client_mismatch` 归属校验、River 图片输入与客户端目录图片能力，以及「已应用配置需重新应用、旧地址仍可用」的升级说明。
- 产出 `release-assets/v2.0.0-beta.29/` 两个文件（安装器 + `.sha256`）；`release-assets/` 在 `.gitignore` 内，不入库。

## Files Changed

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `docs/releases/v2.0.0-beta.29.md`（新增）
- `docs/plans/CURRENT.md`、本记录

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

- PASS：`task --force verify` exit 0 —— 格式 → lint → 类型 → Go 全套 → 126 文件／750 项前端测试 → Windows production 构建（日志 `tmp/verify-beta29.log`）。
- PASS：`task --force package` exit 0（安装器 19,636,234 bytes，日志 `tmp/package-beta29.log`）。
- PASS：`scripts/test-windows-metadata.ps1 -Application bin/omniproxy.exe -Installer bin/OmniProxy-amd64-installer.exe -Version v2.0.0-beta.29` 输出「Windows 应用和安装器版本资源校验通过」。
- PASS：`scripts/verify-release-assets.ps1 -Version v2.0.0-beta.29 -AssetDirectory release-assets/v2.0.0-beta.29` 输出「发布资产校验通过」。
- SHA256：`86652098a4e8e3582e420e15d8e71d75cebdf4a4537049bb75eeba07bbf7b9c3`（`OmniProxy-Setup-v2.0.0-beta.29-windows-amd64.exe`，19,636,234 bytes）。
- 历史资产基线：`release-assets/` 共 24 个目录，全部各 2 个文件；Beta.28 安装器资产未变。
- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送。真实覆盖安装与客户端重连由用户本地验收。

---

# Result

Beta.29 production 本地安装包已构建，通过完整质量门、Windows 版本资源与发布资产校验。未签名、未自动安装、未打标签、未推送，供用户本地覆盖更新。

---

# Deviations

- 沿用 Beta.26–Beta.28 的做法：不执行 `scripts/build-release.ps1`（本机 PATH 缺 `uname`，`task` 会把原生命令的 stderr 当终止错误），改为逐项执行同一组步骤；未修改脚本。
- 未执行 `scripts/set-release-version.ps1`（会经 `ConvertTo-Json` 重排 `info.json`，且数字版本无变化），直接同步三个版本文件。
- 打包时把 NSIS 目录临时加入 PATH（`task --force package` 依赖 `makensis`）。
- **只提交版本准备文件**（版本号三处、发布说明、状态看板、本记录）。包内源码包含两项未提交的改动：本会话的 TASK-20261024-001（固定入口前缀）与 TASK-20261023-003／005（River 图片、客户端图片能力）。按 AGENTS.md「不提交其他在办任务的改动」保留未提交状态；其中 `agentconfig/opencode.go` 同时含两类改动，无法单独剥离。

---

# Remaining Risks

- 安装包未签名，Windows 会提示未知发布者。
- 包来源不完全对应单一提交：固定入口前缀与 River 图片两项改动仍在工作区未提交（与 Beta.26–Beta.28 相同做法）。
- 本版更换客户端入口地址：已应用的配置在重新应用前继续使用旧地址；`/codex` 入口在 Codex 接入启用时按前缀放行，属对齐 v1 的明确取舍（ADR-021）。
- 原生客户端实机验收未执行：Codex 在 `/codex/v1` 下的官方登录与长会话、Claude CLI／Desktop 共用 `/claude` 的归属统计、River 图片实际转发，均需用户本地复核。

---

# Git

## Commit

```text
8cc395c chore(release): 准备 Beta.29 本地生产安装包（版本同步、发布说明、状态看板与构建记录）
```

---

# Completed

Pending.
