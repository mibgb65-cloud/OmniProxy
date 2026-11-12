# TASK-20261112-014: 构建本地 v2.2.3-beta.13 生产安装包

## Status

In Progress

## Created

2026-11-12 21:44（Asia/Singapore）

## Request

用户在已安装的 `v2.2.3-beta.12` 上连接 Antigravity 后发送消息失败（`unsupported_model`），
TASK-20261112-013 已在源码修复，需要产出新的本地生产安装包供用户重试。

## Goal

- 版本由 `2.2.3-beta.12` 提升为 `2.2.3-beta.13`，Windows 数字版本保持 `2.2.3.0`、
  NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.13.md`。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。
- 更新任务记录与 `docs/plans/CURRENT.md`。

## Background

- `v2.2.3-beta.12` 已交付（TASK-20261112-012，版本提交 `89daea9f`），用户已安装。
- 用户连接 Antigravity 后，网关请求明细记录两条 `gemini-4-argon` → `400
  unsupported_model`；桌面端显示 “Agent terminated due to error”。
- TASK-20261112-013（实现提交 `c19339b8`）新增家族名解析与档位保留，Go 全量测试通过。

## Scope

- `internal/version/version.go`、`frontend/package.json` 与 `build/windows/info.json` 的版本映射。
- `docs/releases/v2.2.3-beta.13.md`。
- 生产打包、产物复制到 `release-assets/v2.2.3-beta.13/`、SHA256 侧车与两项校验脚本。
- 任务记录与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不重跑 `task verify`：TASK-20261112-013 已运行完整质量门（format check／lint／typecheck／
  Go 全量测试／前端 147 文件／898 项／Windows production 构建）。
- 不推送、不打标签、不公开发布、不签名。
- 不自动安装、不启动或停止用户实例，不修改真实用户数据。

## Existing Behavior

源码版本为 `2.2.3-beta.12`；`release-assets/v2.2.3-beta.13` 不存在；用户当前运行 beta.12。

## Expected Behavior

覆盖安装后，桌面端发送 `gemini-4-argon` 能按档位解析到上游变体并得到正常回复。

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
- 用户当前实例仍在运行 beta.12，未完全退出时覆盖安装会被拒绝。
- 版本与资源映射不一致或复用旧产物会导致错误交付；用 `--force` 生产构建并核对
  原生版本资源与 SHA256。
- 工作区存在无关的未跟踪旧任务记录，提交时只暂存本任务文件。

---

# Implementation Plan

1. 创建本记录。
2. 更新 `internal/version/version.go` 与 `frontend/package.json`，运行
   `scripts/set-release-version.ps1 -Version v2.2.3-beta.13`，还原 `info.json` 的 LF。
3. 新增双语发行说明并审阅。
4. 执行 `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.13/bin`。
5. 复制安装器到 `release-assets/v2.2.3-beta.13/`，生成 SHA256 侧车，运行
   `scripts/test-windows-metadata.ps1` 与 `scripts/verify-release-assets.ps1`。
6. 填写实际结果，归档任务并提交，更新 CURRENT。

---

# Progress

- 2026-11-12 21:44：创建记录；确认源码版本 `2.2.3-beta.12`，
  `release-assets/v2.2.3-beta.13` 不存在。
- 版本映射通过：应用／前端／Windows 显示版本 `2.2.3-beta.13`，数字版本 `2.2.3.0`，
  NSIS 基础版本 `2.2.3`（`info.json` 的 CRLF 已还原为 LF）。
- 生产打包与两项校验完成：安装器 20,725,199 bytes。

---

# Implementation Result

## Changes

- 应用、前端与 Windows 显示版本统一为 `2.2.3-beta.13`；Windows 数字资源 `2.2.3.0`、
  NSIS 基础版本 `2.2.3` 不变。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.13.md`：说明 Antigravity 家族模型名解析、
  推理档位保留与转发前剥离，以及单变体家族与精确 ID 的边界。
- 版本中包含 TASK-20261112-013 的实现提交 `c19339b8`。

## Build and Artifacts

构建命令（当前进程 PATH 临时加入 NSIS）：

```text
task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.13/bin
```

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.13/OmniProxy-Setup-v2.2.3-beta.13-windows-amd64.exe` | 20,725,199 | `b9f6c09e6bcf9dd28d9caa42d1ffabe5bb5e27cb17899210b74bf4bed83bd767` |
| `tmp/local-v2.2.3-beta.13/bin/omniproxy.exe` | 23,806,976 | `5689c6c1f0f311593bf7cc517238b9f80d512ee0f562100e1e55f19ae4cd45b1` |

摘要侧车 `OmniProxy-Setup-v2.2.3-beta.13-windows-amd64.exe.sha256` 与安装器同目录，
内容为 `<hash>  <文件名>` 加结尾换行、无 BOM。原始 NSIS 安装器位于
`bin/OmniProxy-amd64-installer.exe`。产物目录与 `tmp/` 不入 Git。

## Technical Notes

- 版本映射由 `scripts/set-release-version.ps1` 完成；脚本把 `build/windows/info.json`
  写成 CRLF，已还原为 LF 后再提交。
- 脚本用 `powershell -NoProfile -ExecutionPolicy Bypass -File` 执行；NSIS 通过临时追加
  `C:\Program Files (x86)\NSIS` 到 PATH 找到。
- 工作区未跟踪的 `TASK-20261029-010-subq-provider.md` 未纳入提交。

---

# Verification

## Tests

- [x] Unit Tests：不重跑；见 TASK-20261112-013（Go 全量、`task --force verify` exit 0）。
- [ ] Integration Tests：N/A
- [ ] E2E Tests：真实桌面端重试由用户完成

## Quality Checks

- [x] Format：版本提交只改版本字面量与 `info.json`，新增发行说明。
- [x] Lint：不重跑（TASK-20261112-013 已通过）。
- [x] Type Check：同上。
- [x] Test：同上。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.13/bin` exit 0。
- [x] Verify：本构建任务不运行 `task verify`；实现任务结果为 `task --force verify` exit 0。

## Verification Result

```text
scripts/set-release-version.ps1 -Version v2.2.3-beta.13
已校验 v2.2.3-beta.13，Windows 数字版本 2.2.3.0

task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.13/bin
PACKAGE_EXIT=0（Total size: 20,725,199 bytes）

scripts/test-windows-metadata.ps1 -Application tmp/local-v2.2.3-beta.13/bin/omniproxy.exe -Installer bin/OmniProxy-amd64-installer.exe -Version v2.2.3-beta.13
Windows 应用和安装器版本资源校验通过（exit 0）

scripts/verify-release-assets.ps1 -Version v2.2.3-beta.13 -AssetDirectory release-assets/v2.2.3-beta.13
发布资产校验通过：v2.2.3-beta.13（exit 0）
```

- 安装器 `ProductVersion` 为 `2.2.3`；应用与安装器 Authenticode 均为 `NotSigned`。
- 打包后 `git status` 只有本任务的版本／发行说明／记录与既有无关文件。

---

# Documentation

- [x] CURRENT：已更新本地构建记录
- [x] 新增 `docs/releases/v2.2.3-beta.13.md`
- [x] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR：无需修改

---

# Deviations

- 未调用 `scripts/build-release.ps1`（会强制 `task verify`），沿用本地 Beta 的版本映射 +
  `task --force package` + 两项校验脚本流程。

---

# Git

## Commit

```text
dc88968a chore(release): 准备本地 v2.2.3-beta.13 生产版本
```

修复提交 `c19339b8`、版本提交 `dc88968a`；归档提交 `af7f0b7f`。未打标签、未推送、未公开发布。

---

# Result

`v2.2.3-beta.13` Windows amd64 本地 Beta 生产安装器已生成，修复 Antigravity 家族模型名
导致的 `400 unsupported_model`，版本资源与摘要核对通过。安装器未签名、未自动安装、
未启动；用户从托盘完全退出并备份数据后覆盖安装即可重试对话。

---

# Completed

2026-11-12 21:50
