# TASK-20261023-015: 构建 Beta.27 本地生产安装包

## Status

Completed（2026-10-27 随 v2.0.0 正式发布归档）

## Created

2026-10-23 00:20（Asia/Singapore）

## Request

用户要求：构建新的 Beta 生产版本，供其本机覆盖更新。

## Goal

基于 `ebf117e`（TASK-20261023-014：请求页缓存命中／成本／速度／首字）产出 `release-assets/v2.0.0-beta.27/` 安装器与 SHA256，并通过完整质量门、Windows 版本资源与发布资产校验。

## Background

- 当前生产安装为 Beta.26（`release-assets/v2.0.0-beta.26/`，19,613,101 bytes）。
- Beta.26 之后已提交的代码只有 `ebf117e`（请求页四项指标）与 `a3d15d0`（任务记录哈希）。
- 工作区仍含其他在办任务的未提交改动；核对文件时间为 2026-10-22 18:02–19:44，均早于 Beta.26 安装包构建时间（2026-10-22 23:53），因此本版相对 Beta.26 的实际差异只有请求页四项指标。

## Scope

- 版本同步至 `2.0.0-beta.27`：`internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`（仅 `0409.ProductVersion`）。
- 新增 `docs/releases/v2.0.0-beta.27.md`。
- 完整质量门、NSIS 打包、Windows 版本资源、SHA256 与发布资产校验。

## Out of Scope

- 不提交工作区中其他在办任务的改动（随包打包，但不代为提交）。
- 不自动安装、不停止或替换正在运行的生产进程、不打标签、不推送。
- 不改 `build/config.yml` 与 `wails_tools.nsh`：数字版本仍为 `2.0.0`，无需变更。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `Taskfile.yml`（`verify` / `package`）、`scripts/test-windows-metadata.ps1`、`scripts/verify-release-assets.ps1`

## Risks

- 安装包未签名，Windows 会提示未知发布者。
- 打包来源包含未提交的其他任务改动，不完全对应单一提交（沿用 Beta.26 的既有情况，本版差异未变）。
- 未执行真实覆盖安装，版本读取与数据保留由用户本地验收。

---

# Implementation Plan

1. 版本同步至 `2.0.0-beta.27`，撰写发布说明。
2. `task --force verify` → `task --force package`（NSIS 加入 PATH）。
3. `scripts/test-windows-metadata.ps1` → 复制资产与 SHA256 → `scripts/verify-release-assets.ps1`。
4. 更新 `docs/plans/CURRENT.md` 与本记录，提交版本准备。

---

# Progress

- 版本同步至 `2.0.0-beta.27`（源码／前端／Windows `ProductVersion`），新增发布说明；`build/config.yml` 与 `wails_tools.nsh` 的数字版本仍为 `2.0.0`，未改。
- `task --force verify` exit 0（125 文件 / 740 项前端测试、Go 全部包、Windows production 构建，日志 `tmp/verify-beta27.log`）。
- `task --force package` exit 0（makensis 3.12，安装器 19,618,795 bytes，日志 `tmp/package-beta27.log`）。
- 版本资源校验、资产复制与 SHA256、资产校验全部通过；Beta.26 及更早 21 个发布目录仍各 2 个文件。

---

# Implementation Result

## Changes

- 版本同步至 `2.0.0-beta.27`：`internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`（仅 `0409.ProductVersion`；Windows 数字版本维持 `2.0.0.0`）。
- 新增 `docs/releases/v2.0.0-beta.27.md`，说明请求页四项指标的口径、未知语义、成本边界与不回应用客户端配置。
- 产出 `release-assets/v2.0.0-beta.27/` 两个文件（安装器 + `.sha256`），共 22 个发布目录各 2 个文件，历史资产未变（`release-assets/` 仍在 `.gitignore` 内，不入库）。

## Files Changed

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `docs/releases/v2.0.0-beta.27.md`（新增）
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

- PASS：`task --force verify` exit 0 —— 格式 → lint → 类型 → Go 全套 → 125 文件／740 项前端测试 → Windows production 构建（日志 `tmp/verify-beta27.log`）。
- PASS：`task --force package` exit 0（makensis 3.12，安装器 19,618,795 bytes，日志 `tmp/package-beta27.log`）。
- PASS：`scripts/test-windows-metadata.ps1 -Application bin/omniproxy.exe -Installer <资产> -Version v2.0.0-beta.27` 输出「Windows 应用和安装器版本资源校验通过」。
- PASS：`scripts/verify-release-assets.ps1 -Version v2.0.0-beta.27 -AssetDirectory release-assets/v2.0.0-beta.27` 输出「发布资产校验通过」。
- SHA256：`25d236fdf878b9efcf526bfaeb6daa811d5e575d75c489f59219f612428ad12f`（`OmniProxy-Setup-v2.0.0-beta.27-windows-amd64.exe`，19,618,795 bytes）。
- 历史资产基线：`v2.0.0-beta.6`–`v2.0.0-beta.26` 共 21 个目录仍各 2 个文件；Beta.26 安装器 SHA256 仍为 `3b595828…bf9123`。
- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送。真实覆盖安装与数据保留由用户本地验收。

---

# Documentation

- [x] `docs/releases/v2.0.0-beta.27.md`
- [x] `docs/plans/CURRENT.md`（当前版本条目与 In Progress）

---

# Deviations

- 沿用 Beta.26 的做法，未整体执行 `scripts/build-release.ps1`（本机 PATH 缺 `uname`，`task` 会把 stderr 当终止错误），改为逐项执行同一组步骤；未修改脚本。
- 未执行 `scripts/set-release-version.ps1`（会经 `ConvertTo-Json` 重排 `info.json`，且数字版本无变化），直接同步三个版本文件。

---

# Git

## Commit

```text
1b965e8 chore(release): 准备 Beta.27 本地生产安装包（版本同步、发布说明、状态看板与构建记录）
```

---

# Result

Beta.27 production 本地安装包已构建，通过完整质量门、Windows 版本资源与发布资产校验。未签名、未自动安装、未打标签、未推送，供用户本地覆盖更新。

---

# Remaining Risks

- 安装包未签名，Windows 会提示未知发布者。
- 工作区仍含其他在办任务的未提交改动（时间早于 Beta.26 构建，本版差异未变），包来源不完全对应单次提交。
- 请求页四项指标的口径来自上游 usage；未报告时显示 `—`，不代表该项为零。
- 原生窗口视觉确认（12 列宽度、表头提示、深浅主题）仍需用户在实际安装版中复核。

---

# Completed

2026-10-27 20:40（Asia/Singapore）

---

# Archival (v2.0.0)

- 原状态：In Progress（安装包已构建并校验，等待用户本地覆盖安装验收）
- 2026-10-27：v2.0.0 正式版已公开（标签源码 `c9c48b4a31988a6cc3fa2f4b1d423e61bd10f71b`，
  Actions 完整质量门与 Windows 构建通过），本任务交付内容已随该版本交付。
- 归档依据：归档时工作区无未提交代码改动，本记录引用的实现提交均为上述发布提交的祖先提交。
- 未执行项（实机客户端／原生窗口视觉／真实覆盖安装验收等）不伪造为已验收，转为发布后
  跟进事项，见 `docs/plans/CURRENT.md` 的 Post-release Follow-ups。
