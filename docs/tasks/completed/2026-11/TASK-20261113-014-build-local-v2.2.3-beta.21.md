# TASK-20261113-014: 构建本地 v2.2.3-beta.21 生产安装包

## Status

Completed

## Created

2026-11-13 12:03（Asia/Singapore）

## Request

请求页「进行中置顶 + 分隔」（TASK-20261113-013）需要可安装的本地 Beta。

## Goal

- 版本由 `2.2.3-beta.20` 提升为 `2.2.3-beta.21`，Windows 数字版本保持 `2.2.3.0`、
  NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.21.md`。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。

## Background

- 用户当前运行 `beta.18`，`beta.19`／`beta.20` 未安装。
- 本版累积内容：变体型号档位在真实目录下生效（TASK-20261113-011）、请求页进行中置顶
  与分组分隔（TASK-20261113-013），以及 beta.19 的 Google 卡片瘦身、Antigravity 内联图片、
  Google 轮换设置（迁移 047）。
- `task --force verify` exit 0（149 文件／904 项前端测试、全部 Go 包、Windows production 构建）。

## Scope

- 版本映射、发行说明、生产打包、发布资产与摘要、任务记录与 CURRENT。

## Out of Scope

- 不重跑 `task verify`（实现任务已通过）。
- 不推送、不打标签、不公开发布、不签名、不安装、不启动。

## Existing Behavior

源码版本为 `2.2.3-beta.20`。

## Expected Behavior

覆盖安装后请求页把进行中的记录固定置顶，并显示「已结束」分隔行。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `docs/releases/`、`docs/plans/CURRENT.md`、`release-assets/`

## Dependencies

沿用既有版本映射与打包／校验脚本。

## Risks

- 本地安装器未签名，可能触发未知发布者提示。
- 用户当前实例运行中，未完全退出时覆盖安装会被拒绝。

---

# Implementation Plan

1. 创建本记录。
2. 更新版本并运行 `scripts/set-release-version.ps1 -Version v2.2.3-beta.21`，还原 `info.json` 的 LF。
3. 新增双语发行说明。
4. `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.21/bin`。
5. 复制安装器、生成 SHA256 侧车、运行两项校验脚本，归档并提交。

---

# Progress

- 2026-11-13 12:03：创建记录。

---

# Implementation Result

## Changes

- 应用、前端与 Windows 显示版本统一为 `2.2.3-beta.21`；Windows 数字资源 `2.2.3.0`、
  NSIS 基础版本 `2.2.3` 不变。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.21.md`：进行中置顶与分隔、变体档位修复、
  以及继承自 beta.19／beta.20 的内容。
- 版本包含 TASK-20261113-011（`338261e9`）与 TASK-20261113-013（`b0ac93b1`）。

## Build and Artifacts

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.21/OmniProxy-Setup-v2.2.3-beta.21-windows-amd64.exe` | 20,763,665 | `c4016808b63836a39f23c955f65efc183ce22ffd125bea5b42213a29226ee338` |
| `tmp/local-v2.2.3-beta.21/bin/omniproxy.exe` | 23,876,096 | `1b12550053094cc8d8c449fa6e142a5a2d078953047436fadba0fa6833415fb3` |

摘要侧车与安装器同目录；应用与安装器 Authenticode 均为 `NotSigned`。

---

# Verification

## Quality Checks

- [x] Format：版本文件行尾保持 LF。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.21/bin` exit 0。
- [x] Verify：实现任务 `task --force verify` exit 0（149 文件／904 项前端测试）。

## Verification Result

```text
set-release-version v2.2.3-beta.21  → 已校验，Windows 数字版本 2.2.3.0
package                              → PACKAGE_EXIT=0（20,763,665 bytes）
test-windows-metadata                → 版本资源校验通过
verify-release-assets                → 发布资产校验通过：v2.2.3-beta.21
```

---

# Documentation

- [x] CURRENT
- [x] `docs/releases/v2.2.3-beta.21.md`

---

# Git

## Commit

见归档提交。未打标签、未推送、未公开发布。

---

# Result

`v2.2.3-beta.21` Windows amd64 本地 Beta 生产安装器已生成：请求页进行中置顶并分组分隔，
变体型号档位修复一并交付。安装器未签名、未自动安装、未启动。

---

# Completed

2026-11-13 12:10
