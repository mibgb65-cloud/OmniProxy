# TASK-20261113-019: 构建本地 v2.2.3-beta.23 生产安装包

## Status

Completed

## Created

2026-11-13 15:24（Asia/Singapore）

## Request

> 提交代码后，然后构建新的beta生产版本我进行本地更新，记得完成任务md更新

## Goal

- 版本由 `2.2.3-beta.22` 提升为 `2.2.3-beta.23`，Windows 数字版本保持 `2.2.3.0`、
  NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.23.md`。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。

## Background

- 本版相对 `beta.22` 只累积一项用户可见改动：
  - TASK-20261113-018（`d8d673d9`）：用量页结构区在 ≥1280px 改为两栏，
    左栏「Token 构成」+「提供商用量占比」，右栏「模型用量占比」跨两行，
    卡片内容可纵向伸展、环形图居中、模型条目均匀铺开。
- 该项任务的 `task verify` 已 exit 0（151 文件／924 项前端测试）。
- 用户当前运行 `beta.18`；`beta.19`–`beta.23` 均未安装，安装包一并包含它们的内容。
  `beta.22` 与 `beta.23` 之间无代码差异（仅布局改动），因此本版也包含
  TASK-015／TASK-016 的用量分析图表与代理探测修复。

## Scope

- 版本映射、发行说明、生产打包、发布资产与摘要、任务记录与 CURRENT。

## Out of Scope

- 不重跑 `task verify`（实现任务 TASK-018 已通过，本任务只改版本号与文案）。
- 不推送、不打标签、不公开发布、不签名、不安装、不启动。

## Existing Behavior

源码版本为 `2.2.3-beta.22`。

## Expected Behavior

覆盖安装后用量页结构区为两栏布局，三张结构卡片拼成整齐矩形，无大片空白。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/config.yml`、
  `build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/releases/`、`docs/plans/CURRENT.md`、`release-assets/`

## Dependencies

沿用既有 `set-release-version` → `task package` → 资产复制／摘要 → 校验脚本流程。

## Risks

- 本地安装器未签名，可能触发未知发布者提示。
- 用户当前实例未完全退出时覆盖安装会被拒绝。
- `set-release-version.ps1` 会用 `ConvertTo-Json` 重写 `info.json`，必须再把换行还原为 LF，
  否则整文件会被当成全行改动。

---

# Implementation Plan

1. 创建本记录，提升 `version.go`／`package.json`。
2. 运行 `scripts/set-release-version.ps1 -Version v2.2.3-beta.23`，还原 `info.json` 的 LF。
3. 新增双语发行说明。
4. `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.23/bin`。
5. 复制安装器、生成 SHA256 侧车、运行版本资源与发布资产校验。
6. 更新 Task Record 与 CURRENT，提交并归档。

---

# Progress

- 2026-11-13 15:24：创建记录。
- 2026-11-13 15:27：版本映射、发行说明与打包完成，两项校验通过。

---

# Implementation Result

## Changes

- 应用、前端与 Windows 显示版本统一为 `2.2.3-beta.23`；Windows 数字资源 `2.2.3.0`、
  NSIS 基础版本 `2.2.3` 不变。`info.json` 被脚本重写后已还原为 LF，diff 只剩版本行。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.23.md`：结构区两栏布局，
  以及 beta.22 未安装内容（用量分析视图、代理探测修复）与 beta.19–beta.21 的继承项。
- 版本包含 TASK-20261113-018（`d8d673d9`）。

## Build and Artifacts

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.23/OmniProxy-Setup-v2.2.3-beta.23-windows-amd64.exe` | 20,793,487 | `e5f21d32e57f0ac6cad49fd5ebfc5aa56ea35fd841652018097eed4450ce66af` |
| `tmp/local-v2.2.3-beta.23/bin/omniproxy.exe` | 23,938,560 | `0b135db0e5038abdd16bab5b5a185362f4b77e44552f62a2443f87d2bcb3701c` |

摘要侧车与安装器同目录；应用与安装器 Authenticode 均为 `NotSigned`。

---

# Verification

## Quality Checks

- [x] Format：版本文件与校验脚本输出均保持 LF。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.23/bin` exit 0。
- [x] Verify：原生版本资源与发布资产校验通过。

## Verification Result

```text
set-release-version v2.2.3-beta.23  → 已校验，Windows 数字版本 2.2.3.0
package（PATH 加入 NSIS）            → PACKAGE_EXIT=0（20,793,487 bytes）
test-windows-metadata                → Windows 应用和安装器版本资源校验通过
verify-release-assets                → 发布资产校验通过：v2.2.3-beta.23
```

- 本终端 `PATH` 仍不含 `makensis`，打包前先把 `C:\Program Files (x86)\NSIS` 加入 `PATH`，
  一次成功，避免上次的 exit 127。
- 功能验证沿用 TASK-20261113-018 的 `task verify` exit 0（151 文件／924 项前端测试）。
- 未签名、未自动安装、未启动、未打标签、未推送。

---

# Git

- Commit: `13e4401a`（chore(release): 准备 v2.2.3-beta.23 本地构建）
