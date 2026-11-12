# TASK-20261112-016: 构建本地 v2.2.3-beta.14 生产安装包

## Status

In Progress

## Created

2026-11-12 22:04（Asia/Singapore）

## Request

`v2.2.3-beta.13` 修复家族名后仍报 `unsupported_protocol`（TASK-20261112-015 已修复协议标记），
需要产出新的本地生产安装包供用户重试。

## Goal

- 版本由 `2.2.3-beta.13` 提升为 `2.2.3-beta.14`，Windows 数字版本保持 `2.2.3.0`、
  NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.14.md`。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。
- 更新任务记录与 `docs/plans/CURRENT.md`。

## Background

- `v2.2.3-beta.13` 已交付（TASK-20261112-014，版本提交 `dc88968a`），用户已安装。
- 安装后请求明细变成 `gemini-4-argon-high` + `400 unsupported_protocol`；本地用
  `curl` 对网关复现，任意 Google 模型（含精确 ID）同样失败。
- 根因是 `parseChatRequest` 没有填 `Protocol`，只有桌面端走这条解析分支；
  TASK-20261112-015 已修复（实现提交待回填）。

## Scope

- `internal/version/version.go`、`frontend/package.json` 与 `build/windows/info.json` 的版本映射。
- `docs/releases/v2.2.3-beta.14.md`。
- 生产打包、产物复制到 `release-assets/v2.2.3-beta.14/`、SHA256 侧车与两项校验脚本。
- 任务记录与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不重跑 `task verify`：TASK-20261112-015 已运行完整质量门。
- 不推送、不打标签、不公开发布、不签名。
- 不自动安装、不启动或停止用户实例，不修改真实用户数据。

## Existing Behavior

源码版本为 `2.2.3-beta.13`；`release-assets/v2.2.3-beta.14` 不存在；用户当前运行 beta.13。

## Expected Behavior

覆盖安装后，Antigravity 的 Chat 请求能进入 Google 订阅转发路径并得到回复。

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
- 用户当前实例仍在运行 beta.13，未完全退出时覆盖安装会被拒绝。
- 修复后请求会真正到达上游，可能暴露内容／工具层面的下一层错误，需要按错误码继续跟进。

---

# Implementation Plan

1. 创建本记录。
2. 更新版本并运行 `scripts/set-release-version.ps1 -Version v2.2.3-beta.14`，还原 `info.json` 的 LF。
3. 新增双语发行说明并审阅。
4. 执行 `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.14/bin`。
5. 复制安装器到 `release-assets/v2.2.3-beta.14/`，生成 SHA256 侧车，运行两项校验脚本。
6. 填写实际结果，归档任务并提交，更新 CURRENT。

---

# Progress

- 2026-11-12 22:04：创建记录；确认源码版本 `2.2.3-beta.13`，目标目录不存在。
- 版本映射通过：应用／前端／Windows 显示版本 `2.2.3-beta.14`，数字版本 `2.2.3.0`，
  NSIS 基础版本 `2.2.3`（`info.json` 的 CRLF 已还原为 LF）。
- 生产打包与两项校验完成：安装器 20,727,670 bytes。

---

# Implementation Result

## Changes

- 应用、前端与 Windows 显示版本统一为 `2.2.3-beta.14`；Windows 数字资源 `2.2.3.0`、
  NSIS 基础版本 `2.2.3` 不变。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.14.md`：说明协议标记修复、beta.13 家族名
  修复的端到端结果、隔离实例的 5 组真实请求，以及思考型模型撞 30 秒非流式时限的边界。
- 版本中包含 TASK-20261112-015 的实现。

## Build and Artifacts

构建命令（当前进程 PATH 临时加入 NSIS）：

```text
task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.14/bin
```

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.14/OmniProxy-Setup-v2.2.3-beta.14-windows-amd64.exe` | 20,727,670 | `5f913efb24141c9e1a89d38ec4761998cc8fd261feb10cd589867abc2c40df97` |
| `tmp/local-v2.2.3-beta.14/bin/omniproxy.exe` | 23,812,096 | `e2ab944f3d413160830cd7a973f4c167ebb44f79b91ab8e335f73b2b54333b16` |

摘要侧车 `OmniProxy-Setup-v2.2.3-beta.14-windows-amd64.exe.sha256` 与安装器同目录，
内容为 `<hash>  <文件名>` 加结尾换行、无 BOM。原始 NSIS 安装器位于
`bin/OmniProxy-amd64-installer.exe`。产物目录与 `tmp/` 不入 Git。

## Technical Notes

- 版本映射由 `scripts/set-release-version.ps1` 完成；脚本把 `build/windows/info.json`
  写成 CRLF，已还原为 LF 后再提交。
- 内部预验证用隔离实例完成（`OMNIPROXY_PROFILE=development`、生产数据目录副本、
  端口 3011），验证后已删除该副本与临时二进制，避免凭据副本留在工作区。

---

# Verification

## Tests

- [x] Unit Tests：不重跑；见 TASK-20261112-015。
- [ ] Integration Tests：N/A（本轮构建）
- [ ] E2E Tests：桌面端真机重试由用户完成

## Quality Checks

- [x] Format：版本文件行尾已统一为 LF，无多余空行。
- [x] Lint：不重跑（TASK-20261112-015 已通过）。
- [x] Type Check：同上。
- [x] Test：同上。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.14/bin` exit 0。
- [x] Verify：实现任务 `task --force verify` exit 0。

## Verification Result

```text
scripts/set-release-version.ps1 -Version v2.2.3-beta.14
已校验 v2.2.3-beta.14，Windows 数字版本 2.2.3.0

task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.14/bin
PACKAGE_EXIT=0（Total size: 20,727,670 bytes）

scripts/test-windows-metadata.ps1 -Application tmp/local-v2.2.3-beta.14/bin/omniproxy.exe -Installer bin/OmniProxy-amd64-installer.exe -Version v2.2.3-beta.14
Windows 应用和安装器版本资源校验通过

scripts/verify-release-assets.ps1 -Version v2.2.3-beta.14 -AssetDirectory release-assets/v2.2.3-beta.14
发布资产校验通过：v2.2.3-beta.14
```

- 安装器 `ProductVersion` 为 `2.2.3`；应用与安装器 Authenticode 均为 `NotSigned`。

---

# Documentation

- [x] CURRENT：已更新本地构建记录
- [x] 新增 `docs/releases/v2.2.3-beta.14.md`
- [x] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR：无需修改

---

# Deviations

- 未调用 `scripts/build-release.ps1`（会强制 `task verify`），沿用本地 Beta 的版本映射 +
  `task --force package` + 两项校验脚本流程。
- 本轮先做了隔离实例的真实上游验证再打包，属于超出既有流程的额外确认。

---

# Git

## Commit

```text
02e2a304 fix(proxy): 标记 Antigravity Chat 请求协议并准备 v2.2.3-beta.14
```

修复与版本映射在同一提交；归档提交 `TBD`。未打标签、未推送、未公开发布。

---

# Result

`v2.2.3-beta.14` Windows amd64 本地 Beta 生产安装器已生成，修复 Antigravity 请求缺少协议
标记导致的 `400 unsupported_protocol`，并在隔离实例上确认家族名解析、档位选择与上游回复
全部正常。安装器未签名、未自动安装、未启动；用户覆盖安装后直接重发消息即可。

---

# Completed

2026-11-12 22:14
