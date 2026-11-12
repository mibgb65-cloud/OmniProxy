# TASK-20261112-018: 构建本地 v2.2.3-beta.15 生产安装包

## Status

In Progress

## Created

2026-11-12 22:07（Asia/Singapore）

## Request

`v2.2.3-beta.14` 仍因 30 秒响应头超时返回 502（TASK-20261112-017 已修复），需要产出新的本地
生产安装包供用户重试。

## Goal

- 版本由 `2.2.3-beta.14` 提升为 `2.2.3-beta.15`，Windows 数字版本保持 `2.2.3.0`、
  NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.15.md`。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。
- 更新任务记录与 `docs/plans/CURRENT.md`。

## Background

- `v2.2.3-beta.14` 已交付（TASK-20261112-016，提交 `02e2a304`），用户已安装。
- 安装后请求明细变为 `502 operation_failed`，每次约 30.1 秒，仅 Antigravity；
  根因是订阅客户端的 30 秒 `ResponseHeaderTimeout`，TASK-20261112-017 已修复。
- 修复后隔离实例上一个 60.3 秒的真实流式请求返回 200。

## Scope

- `internal/version/version.go`、`frontend/package.json` 与 `build/windows/info.json` 的版本映射。
- `docs/releases/v2.2.3-beta.15.md`。
- 生产打包、产物复制到 `release-assets/v2.2.3-beta.15/`、SHA256 侧车与两项校验脚本。
- 任务记录与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不重跑 `task verify`：TASK-20261112-017 已运行完整质量门。
- 不推送、不打标签、不公开发布、不签名。
- 不自动安装、不启动或停止用户实例，不修改真实用户数据。

## Existing Behavior

源码版本为 `2.2.3-beta.14`；`release-assets/v2.2.3-beta.15` 不存在；用户当前运行 beta.14。

## Expected Behavior

覆盖安装后思考型模型的会话可以正常完成，不再有 30 秒截断。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`
- `build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/releases/`、`docs/plans/CURRENT.md`
- `release-assets/`

## Dependencies

沿用 Wails v3.1.0、Task、Go 1.26.2、pnpm、NSIS 及既有版本映射／构建校验脚本。

## Risks

- 本地安装器未签名，可能触发未知发布者或 SmartScreen 提示。
- 用户当前实例仍在运行 beta.14，未完全退出时覆盖安装会被拒绝。
- 放宽时限后，真正无响应的上游会占用连接更久（上限仍为 5 分钟响应头／10 分钟整请求）。

---

# Implementation Plan

1. 创建本记录。
2. 更新版本并运行 `scripts/set-release-version.ps1 -Version v2.2.3-beta.15`，还原 `info.json` 的 LF。
3. 新增双语发行说明并审阅。
4. 执行 `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.15/bin`。
5. 复制安装器到 `release-assets/v2.2.3-beta.15/`，生成 SHA256 侧车，运行两项校验脚本。
6. 填写实际结果，归档任务并提交，更新 CURRENT。

---

# Progress

- 2026-11-12 22:07：创建记录。
- 版本映射通过：应用／前端／Windows 显示版本 `2.2.3-beta.15`，数字版本 `2.2.3.0`。
- 生产打包与两项校验完成：安装器 20,730,171 bytes。

---

# Implementation Result

## Changes

- 应用、前端与 Windows 显示版本统一为 `2.2.3-beta.15`；Windows 数字资源 `2.2.3.0`、
  NSIS 基础版本 `2.2.3` 不变。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.15.md`：说明 30 秒响应头超时的根因、5 分钟
  响应头等待与 10 分钟客户端时限，以及 60.3 秒流式请求的隔离实例验证。
- 版本中包含 TASK-20261112-017 的实现。

## Build and Artifacts

构建命令（当前进程 PATH 临时加入 NSIS）：

```text
task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.15/bin
```

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.15/OmniProxy-Setup-v2.2.3-beta.15-windows-amd64.exe` | 20,730,171 | `ca4d3e87d3e1c46be7bbc1ee1dffc13175104f6b9fa50d9bcf86322624a0c5c5` |
| `tmp/local-v2.2.3-beta.15/bin/omniproxy.exe` | 23,817,728 | `3ccd3f6241436d22c65fc761a2d0ebbfc58241c87d88d18fc2b91c0df9b8840d` |

摘要侧车 `OmniProxy-Setup-v2.2.3-beta.15-windows-amd64.exe.sha256` 与安装器同目录，
内容为 `<hash>  <文件名>` 加结尾换行、无 BOM。原始 NSIS 安装器位于
`bin/OmniProxy-amd64-installer.exe`。产物目录与 `tmp/` 不入 Git。

## Technical Notes

- 版本映射由 `scripts/set-release-version.ps1` 完成；`info.json` 的 CRLF 已还原为 LF。
- 隔离验证使用生产数据目录副本与独立端口，验证后已删除副本与临时二进制。

---

# Verification

## Tests

- [x] Unit Tests：不重跑；见 TASK-20261112-017。
- [ ] Integration Tests：见 TASK-20261112-017 的隔离实例结果
- [ ] E2E Tests：桌面端真机重试由用户完成

## Quality Checks

- [x] Format：版本文件行尾保持 LF。
- [x] Lint：不重跑（TASK-20261112-017 已通过）。
- [x] Type Check：同上。
- [x] Test：同上。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.15/bin` exit 0。
- [x] Verify：实现任务 `task --force verify` exit 0。

## Verification Result

```text
scripts/set-release-version.ps1 -Version v2.2.3-beta.15
已校验 v2.2.3-beta.15，Windows 数字版本 2.2.3.0

task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.15/bin
PACKAGE_EXIT=0（Total size: 20,730,171 bytes）

scripts/test-windows-metadata.ps1 -Application tmp/local-v2.2.3-beta.15/bin/omniproxy.exe -Installer bin/OmniProxy-amd64-installer.exe -Version v2.2.3-beta.15
Windows 应用和安装器版本资源校验通过

scripts/verify-release-assets.ps1 -Version v2.2.3-beta.15 -AssetDirectory release-assets/v2.2.3-beta.15
发布资产校验通过：v2.2.3-beta.15
```

- 安装器 `ProductVersion` 为 `2.2.3`；应用与安装器 Authenticode 均为 `NotSigned`。

---

# Documentation

- [x] CURRENT：已更新本地构建记录
- [x] 新增 `docs/releases/v2.2.3-beta.15.md`
- [x] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR：无需修改

---

# Deviations

- 未调用 `scripts/build-release.ps1`（会强制 `task verify`），沿用本地 Beta 的版本映射 +
  `task --force package` + 两项校验脚本流程。

---

# Git

## Commit

```text
042d00be fix(gateway): 放宽 Antigravity 响应头与请求时限并准备 v2.2.3-beta.15
```

归档提交 `1bad6f6f`。未打标签、未推送、未公开发布。

---

# Result

`v2.2.3-beta.15` Windows amd64 本地 Beta 生产安装器已生成，修复 Antigravity 会话 30 秒
截断（`502 operation_failed`），并在隔离实例上确认 60.3 秒的流式请求可以正常完成。
安装器未签名、未自动安装、未启动；用户覆盖安装后直接重发消息即可。

---

# Completed

2026-11-12 22:12
