# TASK-20261114-004: 构建本地 v2.3.1-beta.1 生产安装包

## Status

Completed

## Created

2026-11-14 21:40（Asia/Singapore）

## Request

> 提交代码后，然后构建新的beta生产版本我进行本地更新，记得完成任务md更新

## Goal

- 版本由 `2.3.0` 提升为 `2.3.1-beta.1`；Windows 数字版本 `2.3.1.0`，NSIS 基础版本 `2.3.1`。
- 交付 TASK-20261114-002（流式响应头／心跳）与 TASK-20261114-003（DuMate 图片输入声明）。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。
- 新增双语发行说明 `docs/releases/v2.3.1-beta.1.md`。

## Background

- 实现提交已入库：`4347a6d0`（流式修复）、`313a7833`（DuMate 图片能力），archive 文档提交 `9b108cee`、`c4c4dffa`。
- 两个实现任务的质量门均通过：Go 全包、前端 154 文件／935 项、format／lint／typecheck／build。
- `v2.3.0` 已是公开正式版；本次为本地 Beta 覆盖安装包，用于验证流式修复与 DuMate 图片。

## Scope

- 版本映射（`version.go`、`frontend/package.json`、`build/config.yml`、`build/windows/info.json`、`wails_tools.nsh`）、发行说明、生产打包、发布资产与摘要、任务记录与 CURRENT。

## Out of Scope

- 不重跑完整 `task verify`（两个实现任务已逐项通过；本任务只改版本与文案）。
- 不推送、不打标签、不公开发布、不签名、不自动安装、不启动。
- 不新增功能、不修改业务代码。

## Existing Behavior

源码版本为 `2.3.0`，包含公开正式版内容，不含本轮流式与 DuMate 修复。

## Expected Behavior

覆盖安装后：长时间思考的 Antigravity 请求不再在 60 秒被「已取消」；DuMate 模型在客户端目录里声明可发图；其余功能与 v2.3.0 一致。

---

# Analysis

## Related Modules

- 版本：`internal/version/version.go`、`frontend/package.json`、`build/config.yml`、
  `build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- 产物：`tmp/local-v2.3.1-beta.1/bin/`、`release-assets/v2.3.1-beta.1/`
- 文档：`docs/releases/`、`docs/plans/CURRENT.md`

## Dependencies

沿用 `set-release-version.ps1` → `task package` → 资产复制／摘要 → 校验脚本流程。
NSIS 位于 `C:\Program Files (x86)\NSIS`，需要加入 PATH。

## Risks

- 安装器未签名，可能触发未知发布者提示。
- 运行中的实例未完全退出时覆盖安装会被拒绝。
- `set-release-version.ps1` 用 `ConvertTo-Json` 重写 `info.json`，需把换行还原为 LF。
- 版本跨过正式版 `2.3.0`，数字资源随之从 `2.3.0.0` 变为 `2.3.1.0`，需要同时核对安装器与应用。

---

# Implementation Plan

1. 创建本记录。
2. 提升 `version.go` 与 `frontend/package.json` 到 `2.3.1-beta.1`。
3. 运行 `scripts/set-release-version.ps1 -Version v2.3.1-beta.1`，检查 `info.json` 换行。
4. 新增双语发行说明。
5. `task --force package DEV=false BIN_DIR=tmp/local-v2.3.1-beta.1/bin`。
6. 复制安装器、生成 SHA256 侧车，运行版本资源与发布资产校验。
7. 更新 Task Record 与 CURRENT，提交并归档。

---

# Progress

- 2026-11-14 21:40：创建记录。
- 2026-11-14 21:38：版本映射完成，`info.json` 换行还原为 LF。
- 2026-11-14 21:40：生产编译与 NSIS 打包完成（21,003,545 bytes）。
- 2026-11-14 21:44：发布资产复制与 SHA256 侧车生成，版本资源与资产校验通过。

---

# Implementation Result

## Changes

- 应用与前端版本 `2.3.1-beta.1`；`build/config.yml` 基础版本 `2.3.1`；Windows 资源 `2.3.1.0`、
  `ProductVersion` 为 `2.3.1-beta.1`；NSIS `INFO_PRODUCTVERSION` 为 `2.3.1`。`info.json`
  被脚本重写后换行已还原为 LF，diff 只剩版本行。
- 新增双语发行说明 `docs/releases/v2.3.1-beta.1.md`：流式响应头／心跳修复、DuMate 图片能力声明、
  无迁移与验证边界。
- 版本包含实现提交 `4347a6d0`（流式修复）与 `313a7833`（DuMate 图片能力）。

## Build and Artifacts

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.3.1-beta.1/OmniProxy-Setup-v2.3.1-beta.1-windows-amd64.exe` | 21,003,545 | `5505ad3697e470eeab248f0e1dfc2e121eaabf653cff7e183a1cfa89608ef089` |
| `tmp/local-v2.3.1-beta.1/bin/omniproxy.exe` | 24,173,568 | `18f550d6a668e370fe079989e859e231e0ea452ae57937c6f77d18a5074700f1` |

摘要侧车与安装器同目录；应用与安装器均未签名。

---

# Verification

## Quality Checks

- [x] Format：`info.json` 保持 LF，diff 只含版本行。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.3.1-beta.1/bin` exit 0。
- [x] Verify：原生版本资源与发布资产校验通过。

## Verification Result

```text
set-release-version v2.3.1-beta.1  → 已校验，Windows 数字版本 2.3.1.0
package（PATH 加入 NSIS）           → PACKAGE_EXIT=0（21,003,545 bytes）
test-windows-metadata              → Windows 应用和安装器版本资源校验通过
verify-release-assets              → 发布资产校验通过：v2.3.1-beta.1
```

- 功能验证沿用两个实现任务的门禁结果：Go 全包、前端 154 文件／935 项、
  format／lint／typecheck／build 均 exit 0。
- 未签名、未自动安装、未启动、未打标签、未推送。

---

# Documentation

- [x] 发行说明：`docs/releases/v2.3.1-beta.1.md`
- [x] CURRENT：已追加本版本摘要
- [x] 其余文档无需修改

---

# Deviations

None.

---

# Git

## Commit

```text
a970ca6b chore(release): 准备 v2.3.1-beta.1 本地构建
```

版本映射、发行说明、本记录与 `CURRENT` 在同一提交；未推送、未打标签。

---

# Result

已产出 `v2.3.1-beta.1` 本地 Windows amd64 生产安装包，供用户退出应用并备份数据后覆盖安装。

---

# Completed

2026-11-14 21:44（Asia/Singapore）

