# TASK-20261023-018: 构建 Beta.28 本地生产安装包

## Status

Completed（2026-10-27 随 v2.0.0 正式发布归档）

## Created

2026-10-23 22:40（Asia/Singapore）

## Request

用户要求：构建新的 Beta 生产版本，供其本机覆盖更新。

## Goal

基于 `9e8dc95`（含 TASK-20261023-016 仪表盘调用趋势重做、TASK-20261023-017 数据存储分类与更新残留清理）产出 `release-assets/v2.0.0-beta.28/` 安装器与 SHA256，并通过完整质量门、Windows 版本资源与发布资产校验。

## Background

- 当前生产安装为 Beta.27（`release-assets/v2.0.0-beta.27/`，19,618,795 bytes）。
- Beta.27 之后的功能提交只有两项：`4cc13a6`（调用趋势刻度、柱形与横向滚动条重做）、`59f4674`（数据存储拆出更新缓存与登录凭据并支持清理残留）；其余为任务记录提交。
- 工作区仍含其他在办任务的未提交改动；逐项核对文件时间均为 2026-10-22 18:02–19:44，早于 Beta.27 构建时间（2026-10-23 07:29），因此这些改动在 Beta.27 安装包中已经存在，本版相对 Beta.27 的实际差异只有上述两个功能提交。
- 数据目录实测：`updates/` 102 MB 为安装残留，本版新增的分类与清理入口正是针对该项。

## Scope

- 版本同步至 `2.0.0-beta.28`：`internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`（仅 `0409.ProductVersion`）。
- 新增 `docs/releases/v2.0.0-beta.28.md`。
- 完整质量门、NSIS 打包、Windows 版本资源、SHA256 与发布资产校验。
- 更新 `docs/plans/CURRENT.md`（当前版本条目与 In Progress）。

## Out of Scope

- 不提交工作区中其他在办任务的改动（随包打包，但不代为提交）。
- 不自动安装、不停止或替换正在运行的生产进程、不打标签、不推送。
- 不改 `build/config.yml` 与 `wails_tools.nsh`：数字版本仍为 `2.0.0`，无需变更。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `Taskfile.yml`（`verify` / `package`）、`scripts/test-windows-metadata.ps1`、`scripts/verify-release-assets.ps1`

## Dependencies

无新增依赖；NSIS（`C:\Program Files (x86)\NSIS\makensis.exe`）需临时加入 PATH 供 `task package` 调用。

## Risks

- 安装包未签名，Windows 会提示未知发布者。
- 打包来源包含未提交的其他任务改动，不完全对应单一提交（与 Beta.26／Beta.27 相同，差异未变）。
- 未执行真实覆盖安装，版本读取与数据保留由用户本地验收。
- 安装残留清理需要用户在「数据存储」显式点击，不会在升级时自动执行。

---

# Implementation Plan

1. 版本同步至 `2.0.0-beta.28`，撰写发布说明。
2. `task --force verify` → `task --force package`（NSIS 加入 PATH）。
3. `scripts/test-windows-metadata.ps1` → 复制资产与 SHA256 → `scripts/verify-release-assets.ps1`。
4. 更新 `docs/plans/CURRENT.md` 与本记录，提交版本准备。

---

# Progress

- 版本同步至 `2.0.0-beta.28`（源码／前端／Windows `ProductVersion`），新增发布说明；`build/config.yml` 与 `wails_tools.nsh` 的数字版本仍为 `2.0.0`，未改。
- `task --force verify` exit 0（126 文件 / 750 项前端测试、Go 全部包、Windows production 构建，日志 `tmp/verify-beta28.log`）。
- `task --force package` exit 0（makensis 3.12，安装器 19,626,700 bytes，日志 `tmp/package-beta28.log`）——需把 `C:\Program Files (x86)\NSIS` 临时加入 PATH。
- 版本资源校验、资产复制与 SHA256、资产校验全部通过；Beta.6–Beta.27 共 22 个发布目录仍各 2 个文件。

---

# Implementation Result

## Changes

- 版本同步至 `2.0.0-beta.28`：`internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`（仅 `0409.ProductVersion`；Windows 数字版本维持 `2.0.0.0`）。
- 新增 `docs/releases/v2.0.0-beta.28.md`，说明数据存储六类占用与「清理更新残留」的删除／保留范围，以及调用趋势的重做要点。
- 产出 `release-assets/v2.0.0-beta.28/` 两个文件（安装器 + `.sha256`），共 23 个发布目录各 2 个文件，历史资产未变（`release-assets/` 仍在 `.gitignore` 内，不入库）。

## Files Changed

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `docs/releases/v2.0.0-beta.28.md`（新增）
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

- PASS：`task --force verify` exit 0 —— 格式 → lint → 类型 → Go 全套 → 126 文件／750 项前端测试 → Windows production 构建（日志 `tmp/verify-beta28.log`）。
- PASS：`task --force package` exit 0（makensis 3.12，安装器 19,626,700 bytes，日志 `tmp/package-beta28.log`）。
- PASS：`scripts/test-windows-metadata.ps1 -Application bin/omniproxy.exe -Installer <资产> -Version v2.0.0-beta.28` 输出「Windows 应用和安装器版本资源校验通过」。
- PASS：`scripts/verify-release-assets.ps1 -Version v2.0.0-beta.28 -AssetDirectory release-assets/v2.0.0-beta.28` 输出「发布资产校验通过」。
- SHA256：`8ef4cd32d08a4ddd01b986ad94a265476867b2618f6504f01443fb7b0af269db`（`OmniProxy-Setup-v2.0.0-beta.28-windows-amd64.exe`，19,626,700 bytes）。
- 历史资产基线：`v2.0.0-beta.6`–`v2.0.0-beta.27` 共 22 个目录仍各 2 个文件；Beta.27 安装器 SHA256 仍为 `25d236fd…8ad12f`。
- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送。真实覆盖安装与数据保留由用户本地验收。

---

# Documentation

- [x] `docs/releases/v2.0.0-beta.28.md`
- [x] `docs/plans/CURRENT.md`（当前版本条目与 In Progress）

---

# Deviations

- 沿用 Beta.26／Beta.27 的做法，未整体执行 `scripts/build-release.ps1`（本机 PATH 缺 `uname`，`task` 会把 stderr 当终止错误），改为逐项执行同一组步骤；未修改脚本。
- 未执行 `scripts/set-release-version.ps1`（会经 `ConvertTo-Json` 重排 `info.json`，且数字版本无变化），直接同步三个版本文件。
- 打包时额外把 NSIS 目录加入 PATH（`task package` 依赖 `makensis` 在 PATH 中）。
- 「清理更新残留」不会在升级时自动执行，用户仍需在数据存储页点一次才能释放约 84 MB。

---

# Git

## Commit

```text
c215571 chore(release): 准备 Beta.28 本地生产安装包（版本同步、发布说明、状态看板与构建记录）
```

---

# Result

Beta.28 production 本地安装包已构建，通过完整质量门、Windows 版本资源与发布资产校验。未签名、未自动安装、未打标签、未推送，供用户本地覆盖更新。

---

# Remaining Risks

- 安装包未签名，Windows 会提示未知发布者。
- 工作区仍含其他在办任务的未提交改动（时间早于 Beta.27 构建，本版差异未变），包来源不完全对应单次提交。
- 安装残留清理依赖文件名规则（`runner-<32hex>.exe`、`*.tmp`）；正在运行的辅助程序副本可能删除失败，界面提示失败并保留其余结果。
- 原生窗口视觉确认（趋势图滑块、存储卡片分栏与清理按钮、深浅主题）仍需用户在实际安装版中复核。

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
