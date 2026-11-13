# TASK-20261113-017: 构建本地 v2.2.3-beta.22 生产安装包

## Status

In Progress

## Created

2026-11-13 14:50（Asia/Singapore）

## Request

> 提交代码后，然后构建新的beta生产版本我进行本地更新，记得完成任务md更新

## Goal

- 版本由 `2.2.3-beta.21` 提升为 `2.2.3-beta.22`，Windows 数字版本保持 `2.2.3.0`、
  NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.22.md`。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。

## Background

- 本版累积两项用户可见改动：
  - TASK-20261113-015（`7f642910`）：用量页扩展为完整分析页（KPI、每日趋势切换、
    构成占比、Top 6 模型堆叠、时段分布、请求质量、账号排行）。
  - TASK-20261113-016（`fbc0364a`）：本机代理探测只回填主机名，地址非法时就地提示。
- 两项任务的 `task verify` 均 exit 0（最近一次 151 文件／924 项前端测试）。
- 用户当前运行 `beta.18`，`beta.19`–`beta.21` 未安装；本版一并包含它们的内容。

## Scope

- 版本映射、发行说明、生产打包、发布资产与摘要、任务记录与 CURRENT。

## Out of Scope

- 不重跑 `task verify`（两项实现任务已通过，本任务只改版本号与文案）。
- 不推送、不打标签、不公开发布、不签名、不安装、不启动。

## Existing Behavior

源码版本为 `2.2.3-beta.21`。

## Expected Behavior

覆盖安装后用量页展示完整分析图表；「探测本机代理」回填 `127.0.0.1` 而非
`auto://127.0.0.1`，保存与测试通过。

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

---

# Implementation Plan

1. 创建本记录，提升 `version.go`／`package.json`。
2. 运行 `scripts/set-release-version.ps1 -Version v2.2.3-beta.22`，还原 `info.json` 的 LF。
3. 新增双语发行说明。
4. `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.22/bin`。
5. 复制安装器、生成 SHA256 侧车、运行版本资源与发布资产校验。
6. 更新 Task Record 与 CURRENT，提交并归档。

---

# Progress

- 2026-11-13 14:50：创建记录。
- 2026-11-13 15:02：版本映射、发行说明与打包完成，三项校验通过。

---

# Implementation Result

## Changes

- 应用、前端与 Windows 显示版本统一为 `2.2.3-beta.22`；Windows 数字资源 `2.2.3.0`、
  NSIS 基础版本 `2.2.3` 不变。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.22.md`：用量页分析图表、
  本机代理探测修复，以及继承自 beta.19–beta.21 的内容。
- 版本包含 TASK-20261113-015（`7f642910`）与 TASK-20261113-016（`fbc0364a`）。

## Build and Artifacts

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.22/OmniProxy-Setup-v2.2.3-beta.22-windows-amd64.exe` | 20,791,060 | `d5f175876dc7fc3f66b628096bc755f3f4432237cdb33b42b2b1dfd93f84994f` |
| `tmp/local-v2.2.3-beta.22/bin/omniproxy.exe` | 23,934,464 | `bab8ac182cbe3bc8215a8dc84485bc8fa03be39212478b92b2229f7d5dbb3337` |

摘要侧车与安装器同目录；应用与安装器 Authenticode 均为 `NotSigned`。

---

# Verification

## Quality Checks

- [x] Format：版本文件与校验脚本输出均保持 LF。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.22/bin` exit 0。
- [x] Verify：原生版本资源与发布资产校验通过。

## Verification Result

```text
set-release-version v2.2.3-beta.22  → 已校验，Windows 数字版本 2.2.3.0
package（PATH 加入 NSIS）            → PACKAGE_EXIT=0（20,791,060 bytes）
test-windows-metadata                → Windows 应用和安装器版本资源校验通过
verify-release-assets                → 发布资产校验通过：v2.2.3-beta.22
```

- 首次打包因当前终端 `PATH` 没有 `makensis`（exit 127）失败；把
  `C:\Program Files (x86)\NSIS` 加入 `PATH` 后重跑成功，产物与版本资源一致。
- 功能验证沿用两项实现任务的 `task verify` exit 0（151 文件／924 项前端测试）。
- 未签名、未自动安装、未启动、未打标签、未推送。

---

# Git

- Commit: 见归档提交。
