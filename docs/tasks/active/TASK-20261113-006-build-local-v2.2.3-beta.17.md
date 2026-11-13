# TASK-20261113-006: 构建本地 v2.2.3-beta.17 生产安装包

## Status

Completed

## Created

2026-11-13 09:50（Asia/Singapore）

## Request

本轮两项改动需要可安装的本地 Beta：型号档位优先（TASK-20261113-004）与
Google 订阅账号轮换方式（TASK-20261113-005，含迁移 047）。

## Goal

- 版本由 `2.2.3-beta.16` 提升为 `2.2.3-beta.17`，Windows 数字版本保持 `2.2.3.0`、
  NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.17.md`。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。
- 更新任务记录与 `docs/plans/CURRENT.md`。

## Background

- 用户当前运行 `v2.2.3-beta.16`（TASK-20261113-003）。
- 本版包含：`gemini-4-hydrogen-high` 等变体型号的推理等级不再被客户端档位字段降级；
  Google 账号页新增「Google 账号设置」，可选队列轮换／平衡切换（迁移 047 新建
  `google_settings`，默认 `queue`）。
- 两项都已通过 `task --force verify`（148 文件／902 项前端测试、全部 Go 包、Windows
  production 构建）。

## Scope

- 版本映射：`internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`。
- `docs/releases/v2.2.3-beta.17.md`。
- 生产打包、产物复制到 `release-assets/v2.2.3-beta.17/`、SHA256 侧车与两项校验脚本。
- 任务记录与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不重跑 `task verify`：两项实现任务已运行完整质量门。
- 不推送、不打标签、不公开发布、不签名。
- 不自动安装、不启动或停止用户实例，不修改真实用户数据。

## Existing Behavior

源码版本为 `2.2.3-beta.16`；`release-assets/v2.2.3-beta.17` 不存在。

## Expected Behavior

覆盖安装后，`gemini-4-hydrogen-high` 的推理等级显示为 high；Google 账号页出现
「Google 账号设置」，可切换轮换方式；升级后首次启动执行迁移 047。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`
- `build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/releases/`、`docs/plans/CURRENT.md`、`release-assets/`

## Dependencies

沿用 Wails v3.1.0、Task、Go 1.26.2、pnpm、NSIS 及既有版本映射／构建校验脚本。

## Risks

- 本地安装器未签名，可能触发未知发布者或 SmartScreen 提示。
- 新增迁移 047：首次启动会在 `google_settings` 建表并写入默认 `queue`，不改动其它表。
- 用户当前实例仍在运行 beta.16，未完全退出时覆盖安装会被拒绝。

---

# Implementation Plan

1. 创建本记录。
2. 更新版本并运行 `scripts/set-release-version.ps1 -Version v2.2.3-beta.17`，还原 `info.json` 的 LF。
3. 新增双语发行说明。
4. 执行 `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.17/bin`。
5. 复制安装器到 `release-assets/v2.2.3-beta.17/`，生成 SHA256 侧车，运行两项校验脚本。
6. 填写实际结果，归档并提交，更新 CURRENT。

---

# Progress

- 2026-11-13 09:50：创建记录。

---

# Implementation Result

## Changes

- 应用、前端与 Windows 显示版本统一为 `2.2.3-beta.17`；Windows 数字资源 `2.2.3.0`、
  NSIS 基础版本 `2.2.3` 不变。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.17.md`：说明 Google 账号轮换设置、型号档位
  优先的修复、迁移 047，以及不修复的桌面端 60 秒超时边界。
- 版本包含 TASK-20261113-004（`97d15cdb`）与 TASK-20261113-005（`40f1f92a`）。

## Build and Artifacts

```text
task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.17/bin
```

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.17/OmniProxy-Setup-v2.2.3-beta.17-windows-amd64.exe` | 20,745,659 | `a4147c27e2071d6191d3c705f878288efc59657395a07561f52ee0d67fdba637` |
| `tmp/local-v2.2.3-beta.17/bin/omniproxy.exe` | 23,847,424 | `cca8d4d462268569ad2e981b4af5e821b1194f179ad4bf1f57417e7b3079f391` |

摘要侧车 `OmniProxy-Setup-v2.2.3-beta.17-windows-amd64.exe.sha256` 与安装器同目录，内容为
`<hash>  <文件名>` 加结尾换行、无 BOM；应用与安装器 Authenticode 均为 `NotSigned`。

---

# Verification

## Quality Checks

- [x] Format：版本文件行尾保持 LF。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.17/bin` exit 0。
- [x] Verify：实现任务 `task --force verify` exit 0（148 文件／902 项前端测试）。

## Verification Result

```text
scripts/set-release-version.ps1 -Version v2.2.3-beta.17
已校验 v2.2.3-beta.17，Windows 数字版本 2.2.3.0

task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.17/bin
PACKAGE_EXIT=0（Total size: 20,745,659 bytes）

scripts/test-windows-metadata.ps1 -Version v2.2.3-beta.17
Windows 应用和安装器版本资源校验通过

scripts/verify-release-assets.ps1 -Version v2.2.3-beta.17
发布资产校验通过：v2.2.3-beta.17
```

---

# Documentation

- [x] CURRENT
- [x] `docs/releases/v2.2.3-beta.17.md`

---

# Git

## Commit

见归档提交。

未打标签、未推送、未公开发布。

---

# Result

`v2.2.3-beta.17` Windows amd64 本地 Beta 生产安装器已生成，包含 Google 账号轮换设置与
型号档位优先的修复，新增迁移 047（`google_settings`）。安装器未签名、未自动安装、未启动。

---

# Completed

2026-11-13 09:56
