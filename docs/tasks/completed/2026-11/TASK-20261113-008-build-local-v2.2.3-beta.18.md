# TASK-20261113-008: 构建本地 v2.2.3-beta.18 生产安装包

## Status

Completed

## Created

2026-11-13 09:44（Asia/Singapore）

## Request

Antigravity 发图片的修复（TASK-20261113-007）需要可安装的本地 Beta。

## Goal

- 版本由 `2.2.3-beta.17` 提升为 `2.2.3-beta.18`，Windows 数字版本保持 `2.2.3.0`、
  NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.18.md`。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。

## Background

- `v2.2.3-beta.17` 已产出但未交付（用户尚未确认安装）。
- 本版只包含 TASK-20261113-007（`0b55d05d`）：用户消息内联图片转成上游 `inlineData`，
  隔离实例非流式／流式各一次真实请求 200。
- `task --force verify` exit 0（148 文件／902 项前端测试、全部 Go 包、Windows production 构建）。

## Scope

- 版本映射、发行说明、生产打包、发布资产与摘要、任务记录与 CURRENT。

## Out of Scope

- 不重跑 `task verify`（实现任务已通过）。
- 不推送、不打标签、不公开发布、不签名、不安装、不启动。

## Existing Behavior

源码版本为 `2.2.3-beta.17`。

## Expected Behavior

覆盖安装后 Antigravity 里的图片消息不再返回 `400 unsupported_content`。

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
2. 更新版本并运行 `scripts/set-release-version.ps1 -Version v2.2.3-beta.18`，还原 `info.json` 的 LF。
3. 新增双语发行说明。
4. `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.18/bin`。
5. 复制安装器、生成 SHA256 侧车、运行两项校验脚本，归档并提交。

---

# Progress

- 2026-11-13 09:44：创建记录。

---

# Implementation Result

## Changes

- 应用、前端与 Windows 显示版本统一为 `2.2.3-beta.18`；Windows 数字资源 `2.2.3.0`、
  NSIS 基础版本 `2.2.3` 不变。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.18.md`：内联图片修复、无法内联的形式仍
  明确失败、`supportsImages` 现状与桌面端 60 秒上限的边界。
- 版本包含 TASK-20261113-007（`0b55d05d`），并沿用 beta.17 的 Google 轮换设置与型号档位修复。

## Build and Artifacts

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.18/OmniProxy-Setup-v2.2.3-beta.18-windows-amd64.exe` | 20,753,619 | `05d5773b45c2b9e2b1d0b43b6ab02652bc9d860904659ba16d75f9d29887f270` |
| `tmp/local-v2.2.3-beta.18/bin/omniproxy.exe` | 23,858,688 | `387406fae9cc931d5c18e3c8f396535196cda5e0f7221b0c5f30b1e33dad2267` |

摘要侧车与安装器同目录；应用与安装器 Authenticode 均为 `NotSigned`。

---

# Verification

## Quality Checks

- [x] Format：版本文件行尾保持 LF。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.18/bin` exit 0。
- [x] Verify：实现任务 `task --force verify` exit 0。

## Verification Result

```text
set-release-version v2.2.3-beta.18  → 已校验，Windows 数字版本 2.2.3.0
package                              → PACKAGE_EXIT=0（20,753,619 bytes）
test-windows-metadata                → 版本资源校验通过
verify-release-assets                → 发布资产校验通过：v2.2.3-beta.18
```

---

# Documentation

- [x] CURRENT
- [x] `docs/releases/v2.2.3-beta.18.md`

---

# Git

## Commit

见归档提交。未打标签、未推送、未公开发布。

---

# Result

`v2.2.3-beta.18` Windows amd64 本地 Beta 生产安装器已生成，包含 Antigravity 内联图片修复。
安装器未签名、未自动安装、未启动。

---

# Completed

2026-11-13 09:56
