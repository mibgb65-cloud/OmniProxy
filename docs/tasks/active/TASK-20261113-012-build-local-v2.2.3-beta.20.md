# TASK-20261113-012: 构建本地 v2.2.3-beta.20 生产安装包

## Status

Completed

## Created

2026-11-13 11:38（Asia/Singapore）

## Request

变体型号档位修复（TASK-20261113-011）需要在真实目录下可安装验证，`beta.18` 的同类修复
在生产数据上无效。

## Goal

- 版本由 `2.2.3-beta.19` 提升为 `2.2.3-beta.20`，Windows 数字版本保持 `2.2.3.0`、
  NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.20.md`。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。

## Background

- 用户当前运行 `beta.18`（安装于 2026-11-13 09:48），`beta.19` 未安装。
- `beta.18` 里那条档位修复只覆盖“目录已折叠”的形态，而 Google 这一代只发布变体，
  所以请求明细里仍显示 `low`。
- 本版包含 TASK-20261113-011（`338261e9`）与 beta.19 的全部内容（内联图片、
  Google 轮换设置、卡片瘦身）。
- `task --force verify` exit 0。

## Scope

- 版本映射、发行说明、生产打包、发布资产与摘要、任务记录与 CURRENT。

## Out of Scope

- 不重跑 `task verify`（实现任务已通过）。
- 不推送、不打标签、不公开发布、不签名、不安装、不启动。

## Existing Behavior

源码版本为 `2.2.3-beta.19`。

## Expected Behavior

覆盖安装后 `gemini-4-hydrogen-xhigh` 的推理等级显示 `xhigh`。

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
2. 更新版本并运行 `scripts/set-release-version.ps1 -Version v2.2.3-beta.20`，还原 `info.json` 的 LF。
3. 新增双语发行说明。
4. `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.20/bin`。
5. 复制安装器、生成 SHA256 侧车、运行两项校验脚本，归档并提交。

---

# Progress

- 2026-11-13 11:38：创建记录。

---

# Implementation Result

## Changes

- 应用、前端与 Windows 显示版本统一为 `2.2.3-beta.20`；Windows 数字资源 `2.2.3.0`、
  NSIS 基础版本 `2.2.3` 不变。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.20.md`：说明上一版为何在真实目录失效、
  现在如何按同族兄弟档位判定，并列出继承自 beta.19 的内容。
- 版本包含 TASK-20261113-011（`338261e9`）。

## Build and Artifacts

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.20/OmniProxy-Setup-v2.2.3-beta.20-windows-amd64.exe` | 20,761,087 | `e1754bfedb5e4104e4e870017c5a3e70b951908c3cabe8b71ab15f5a5549554c` |
| `tmp/local-v2.2.3-beta.20/bin/omniproxy.exe` | 23,870,464 | `badfcdb444f1aac16395ca77dbc5cfba1319baa59624ac9f77a9a48f9560f631` |

摘要侧车与安装器同目录；应用与安装器 Authenticode 均为 `NotSigned`。

---

# Verification

## Quality Checks

- [x] Format：版本文件行尾保持 LF。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.20/bin` exit 0。
- [x] Verify：实现任务 `task --force verify` exit 0。

## Verification Result

```text
set-release-version v2.2.3-beta.20  → 已校验，Windows 数字版本 2.2.3.0
package                              → PACKAGE_EXIT=0（20,761,087 bytes）
test-windows-metadata                → 版本资源校验通过
verify-release-assets                → 发布资产校验通过：v2.2.3-beta.20
```

---

# Documentation

- [x] CURRENT
- [x] `docs/releases/v2.2.3-beta.20.md`

---

# Git

## Commit

见归档提交。未打标签、未推送、未公开发布。

---

# Result

`v2.2.3-beta.20` Windows amd64 本地 Beta 生产安装器已生成，包含实测有效的变体型号档位修复。
安装器未签名、未自动安装、未启动。

---

# Completed

2026-11-13 11:44
