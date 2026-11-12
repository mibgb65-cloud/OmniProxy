# TASK-20261112-012: 构建本地 v2.2.3-beta.12 生产安装包

## Status

In Progress

## Created

2026-11-12 20:52（Asia/Singapore）

## Request

用户查看已安装的 `v2.2.3-beta.11` 后要求 Antigravity 卡片与其它客户端卡片一致
（TASK-20261112-011）。修复已在源码完成并通过 `task verify`，需要产出新的本地生产安装包
供用户覆盖更新，并更新任务记录。

## Goal

- 版本由 `2.2.3-beta.11` 提升为 `2.2.3-beta.12`，Windows 数字版本保持 `2.2.3.0`、
  NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.12.md`。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。
- 更新任务记录与 `docs/plans/CURRENT.md`。

## Background

- `v2.2.3-beta.11` 已交付（TASK-20261112-010，版本提交 `fbaec345`），用户已安装。
- 用户指出该版本中 Antigravity 卡片占满整行、与下方客户端卡片版式不一致；TASK-20261112-011
  已将其改为网格分组卡片（实现提交 `b2e501bb`）。
- 自 beta.11 起无后端、数据库或绑定变化，仅前端界面与文案。

## Scope

- `internal/version/version.go`、`frontend/package.json` 与 `build/windows/info.json` 的版本映射。
- `docs/releases/v2.2.3-beta.12.md`。
- 生产打包、产物复制到 `release-assets/v2.2.3-beta.12/`、SHA256 侧车与两项校验脚本。
- 任务记录与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不重跑 `task verify`：TASK-20261112-011 已运行完整质量门（format check／lint／typecheck／
  147 文件／898 项前端测试／Windows production 构建）。
- 不推送、不打标签、不公开发布、不签名。
- 不自动安装、不启动或停止用户实例，不修改真实用户数据。

## Existing Behavior

源码版本为 `2.2.3-beta.11`；`release-assets/v2.2.3-beta.12` 不存在；用户当前运行 beta.11。

## Expected Behavior

用户覆盖安装后，网关页「本地 Agent 接入」中 Antigravity 与其余客户端同为网格卡片，
连接、选择默认模型、断开能力不变。

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
- 用户当前实例仍在运行 beta.11，未完全退出时覆盖安装会被拒绝。
- 版本与资源映射不一致或复用旧产物会导致错误交付；用 `--force` 生产构建并核对
  原生版本资源与 SHA256。
- 工作区存在无关的未跟踪旧任务记录，提交时只暂存本任务文件。

---

# Implementation Plan

1. 创建本记录。
2. 更新 `internal/version/version.go` 与 `frontend/package.json`，运行
   `scripts/set-release-version.ps1 -Version v2.2.3-beta.12`，还原 `info.json` 的 LF。
3. 新增双语发行说明并审阅。
4. 执行 `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.12/bin`。
5. 复制安装器到 `release-assets/v2.2.3-beta.12/`，生成 SHA256 侧车，运行
   `scripts/test-windows-metadata.ps1` 与 `scripts/verify-release-assets.ps1`。
6. 填写实际结果，归档任务并提交，更新 CURRENT。

---

# Progress

- 2026-11-12 20:52：创建记录；确认源码版本 `2.2.3-beta.11`，`release-assets/v2.2.3-beta.12`
  不存在。
- 版本映射通过：应用／前端／Windows 显示版本 `2.2.3-beta.12`，数字版本 `2.2.3.0`，
  NSIS 基础版本 `2.2.3`（`info.json` 的 CRLF 已还原为 LF）。
- 发行说明与生产打包完成：安装器 20,717,553 bytes；元数据与发布资产校验通过。

---

# Implementation Result

## Changes

- 应用、前端与 Windows 显示版本统一为 `2.2.3-beta.12`；Windows 数字资源 `2.2.3.0`、
  NSIS 基础版本 `2.2.3` 不变。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.12.md`：说明 Antigravity 卡片改为网格
  普通卡片、状态徽标语义与文案精简，并说明无后端／迁移变化。
- 版本中包含 TASK-20261112-011 的实现提交 `b2e501bb`。

## Build and Artifacts

构建命令（当前进程 PATH 临时加入 NSIS）：

```text
task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.12/bin
```

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.12/OmniProxy-Setup-v2.2.3-beta.12-windows-amd64.exe` | 20,717,553 | `8aecc54d75eaabdf02dc1e60e5ee805c20e9725946e3e873ca393856c93807d1` |
| `tmp/local-v2.2.3-beta.12/bin/omniproxy.exe` | 23,793,664 | `6f386deb904bdce1469aea746f844620fddfe48a3141029930d4c17e2bdc17c8` |

摘要侧车 `OmniProxy-Setup-v2.2.3-beta.12-windows-amd64.exe.sha256` 与安装器同目录，
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

- [x] Unit Tests：不重跑；见 TASK-20261112-011（前端 147 文件／898 项、Go 全部包通过）。
- [ ] Integration Tests：N/A
- [ ] E2E Tests：N/A（原生窗口观感由用户装后确认）

## Quality Checks

- [x] Format：版本提交只改版本字面量与 `info.json`，新增发行说明；未改业务代码。
- [x] Lint：不重跑（TASK-20261112-011 已通过）。
- [x] Type Check：同上。
- [x] Test：同上。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.12/bin` exit 0。
- [x] Verify：本构建任务不运行 `task verify`；实现任务结果为 `task --force verify` exit 0。

## Verification Result

```text
scripts/set-release-version.ps1 -Version v2.2.3-beta.12
已校验 v2.2.3-beta.12，Windows 数字版本 2.2.3.0

task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.12/bin
PACKAGE_EXIT=0（Total size: 20,717,553 bytes）

scripts/test-windows-metadata.ps1 -Application tmp/local-v2.2.3-beta.12/bin/omniproxy.exe -Installer bin/OmniProxy-amd64-installer.exe -Version v2.2.3-beta.12
Windows 应用和安装器版本资源校验通过（exit 0）

scripts/verify-release-assets.ps1 -Version v2.2.3-beta.12 -AssetDirectory release-assets/v2.2.3-beta.12
发布资产校验通过：v2.2.3-beta.12（exit 0）
```

- 安装器 `ProductVersion` 为 `2.2.3`；应用与安装器 Authenticode 均为 `NotSigned`。
- 打包后 `git status` 只有本任务的版本／发行说明／记录与既有无关文件。

---

# Documentation

- [x] CURRENT：已更新本地构建记录
- [x] 新增 `docs/releases/v2.2.3-beta.12.md`
- [x] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR：无需修改

---

# Deviations

- 未调用 `scripts/build-release.ps1`（会强制 `task verify`），沿用本地 Beta 的版本映射 +
  `task --force package` + 两项校验脚本流程。

---

# Git

## Commit

```text
89daea9f chore(release): 准备本地 v2.2.3-beta.12 生产版本
```

界面修复提交 `b2e501bb`、版本提交 `89daea9f`；本记录与 CURRENT 的归档提交 `bd1e81ea`。
未打标签、未推送、未公开发布。

---

# Result

`v2.2.3-beta.12` Windows amd64 本地 Beta 生产安装器已生成，把网关页的 Antigravity 入口
改为与其它客户端一致的网格卡片，版本资源与摘要核对通过。安装器未签名、未自动安装、
未启动；用户从托盘完全退出并备份数据后覆盖安装即可看到新版卡片。

---

# Completed

2026-11-12 21:00
