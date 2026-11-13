# TASK-20261113-021: 构建本地 v2.2.3-beta.24 生产安装包

## Status

Completed

## Created

2026-11-13 21:02（Asia/Singapore）

## Request

> 提交代码后，然后构建新的beta生产版本我进行本地更新，记得完成任务md更新

## Goal

- 版本由 `2.2.3-beta.23` 提升为 `2.2.3-beta.24`，Windows 数字版本保持 `2.2.3.0`、
  NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.24.md`。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。

## Background

- 实现任务 TASK-20261113-020（`741a5d29`）已提交：新增第十三个内置服务商
  百度搭子 DuMate 个人版，包含登录态导入、网关转发、积分查询与手动签到、
  迁移 048、Wails 绑定与前端界面。
- 该任务 `task verify` exit 0（154 文件／935 项前端测试、全部 Go 包、
  Windows production 构建）；自那以后只有版本文案变更（本任务），
  因此不重跑完整质量门，只跑生产打包与版本资源校验。
- user 当前安装版本未知；本版为本地 Beta 覆盖安装包。

## Scope

- 版本映射、发行说明、生产打包、发布资产与摘要、任务记录与 CURRENT。

## Out of Scope

- 不重跑 `task verify`（实现任务已通过；本任务只改版本号与文案）。
- 不推送、不打标签、不公开发布、不签名、不安装、不启动。
- 不新增功能、不修改业务代码。

## Existing Behavior

源码版本为 `2.2.3-beta.23`，包含 TASK-20261113-018 的用量页两栏布局。

## Expected Behavior

覆盖安装后能在账号页看到「百度搭子 DuMate」提供商，可导入登录态、查看积分并手动签到；
其余功能与 beta.23 一致。

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
- `set-release-version.ps1` 会用 `ConvertTo-Json` 重写 `info.json`，需检查换行是否被改成 CRLF。

---

# Implementation Plan

1. 创建本记录，提升 `version.go`／`package.json`。
2. 运行 `scripts/set-release-version.ps1 -Version v2.2.3-beta.24`，检查 `info.json` 换行。
3. 新增双语发行说明。
4. `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.24/bin`。
5. 复制安装器、生成 SHA256 侧车、运行版本资源与发布资产校验。
6. 更新 Task Record 与 CURRENT，提交并归档。

---

# Progress

- 2026-11-13 21:02：创建记录。
- 2026-11-13 21:05：版本映射、发行说明与打包完成；`info.json` 换行已还原为 LF。
- 2026-11-13 21:07：安装器复制到 `release-assets/v2.2.3-beta.24/`，生成 SHA256 侧车；
  原生版本资源与发布资产校验通过。

---

# Implementation Result

## Changes

- 应用、前端与 Windows 显示版本统一为 `2.2.3-beta.24`；Windows 数字资源 `2.2.3.0`、
  NSIS 基础版本 `2.2.3` 不变。`info.json` 被脚本重写后已还原为 LF，diff 只剩版本行。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.24.md`：DuMate 个人版接入、
  已知边界（未验证真实账号、自动签到未实现）与继承自 beta.19–beta.23 的内容。
- 版本包含实现提交 `741a5d29`（TASK-20261113-020）。

## Build and Artifacts

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.24/OmniProxy-Setup-v2.2.3-beta.24-windows-amd64.exe` | 20,986,834 | `56b033b66505a966c243f68c317b43aff2d6c5ab1911d979b4a6999747f953ee` |
| `tmp/local-v2.2.3-beta.24/bin/omniproxy.exe` | 24,152,064 | `80bbed5a75ee07891b05f1305910df826481b35ee15c9cceb8f2e4afa466bb1b` |

摘要侧车与安装器同目录；应用与安装器均未签名。

---

# Verification

## Quality Checks

- [x] Format：版本文件与 `info.json` 均保持 LF，diff 只含版本行。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.24/bin` exit 0。
- [x] Verify：原生版本资源与发布资产校验通过。

## Verification Result

```text
set-release-version v2.2.3-beta.24  → 已校验，Windows 数字版本 2.2.3.0
package（PATH 加入 NSIS）            → PACKAGE_EXIT=0（20,986,834 bytes）
test-windows-metadata                → Windows 应用和安装器版本资源校验通过
verify-release-assets                → 发布资产校验通过：v2.2.3-beta.24
```

- 功能验证沿用 TASK-20261113-020 的 `task verify` exit 0（154 文件／935 项前端测试、
  全部 Go 包、Windows production 构建）。
- 未签名、未自动安装、未启动、未打标签、未推送。

---

# Git

- Commit: `de68b5a2`（chore(release): 准备 v2.2.3-beta.24 本地构建）
- 任务记录归档与 CURRENT 在同一文档提交中补齐；未推送、未打标签。

---

# Result

已产出 `v2.2.3-beta.24` 本地 Windows amd64 生产安装包，供用户退出应用并备份数据后覆盖安装。
