# TASK-20261116-002: 构建本地 v2.3.1-beta.8 生产安装包

## Status

Completed

## Created

2026-11-16 08:28（Asia/Singapore）

## Request

延续 TASK-20261116-001：用户当前运行 `v2.3.1-beta.7`，需要包含 HowToFix 目录自动同步的
新安装包做本地更新。

## Goal

- 版本提升为 `2.3.1-beta.8`；Windows 数字版本仍为 `2.3.1.0`，NSIS 基础版本 `2.3.1`。
- 交付 TASK-20261116-001（HowToFix 目录自动同步、路由页手动同步入口、账号页提示修正）。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。
- 新增双语发行说明 `docs/releases/v2.3.1-beta.8.md`，更新 CURRENT。

## Background

- 本版本无数据库迁移；`v2.3.1-beta.7` 的用户升级后目录会在启动几秒内自动补齐。

## Scope

- 版本映射、发行说明、生产打包、发布资产与摘要、任务记录与 CURRENT。

## Out of Scope

- 不新增功能、不改动网关与迁移。
- 不推送、不打标签、不公开发布、不签名、不自动安装、不启动。

---

# Implementation Plan

1. `internal/version/version.go` 与 `frontend/package.json` 提升到 `2.3.1-beta.8`。
2. 新增 `docs/releases/v2.3.1-beta.8.md`。
3. 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.8 -OutputDirectory release-assets\v2.3.1-beta.8`。
4. 还原 `build/windows/info.json` 换行为 LF，确认 diff 只含版本行。
5. 记录产物大小与 SHA256，回填 Task Record 与 CURRENT，提交并归档。

---

# Progress

- 2026-11-16 08:28：提升版本、写发行说明。
- 2026-11-16 08:33：`build-release.ps1` EXIT=0——verify、打包、原生版本资源与发布资产校验全部通过；
  安装器 21,085,494 bytes。

---

# Implementation Result

## Changes

- 应用与前端版本 `2.3.1-beta.8`；Windows 数字资源保持 `2.3.1.0`，`ProductVersion` 为
  `2.3.1-beta.8`；`build/config.yml` 与 `wails_tools.nsh` 数字版本不变。
- 新增双语发行说明 `docs/releases/v2.3.1-beta.8.md`：目录自动同步、手动同步入口、
  账号页提示修正、状态合并语义与验证边界。
- 版本包含实现提交（见下方 Git）。

## Build and Artifacts

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.3.1-beta.8/OmniProxy-Setup-v2.3.1-beta.8-windows-amd64.exe` | 21,085,494 | `2875014e88f930e2ee51ea2e097505922caa3f4f19b8cbd1c19ec2af63acff6c` |
| `bin/omniproxy.exe` | 24,291,328 | `3a3170a02fd85053d2f90a1c03a3b34ee1ac4228d2a14aae0e2b17e58a9bc108` |

摘要侧车与安装器同目录；应用与安装器均未签名。

## Technical Notes

- 版本提升继续使用 `[IO.File]::WriteAllText` + `UTF8Encoding($false)`，避免
  `Set-Content -Encoding utf8` 写入 BOM（beta.7 踩过该坑）。
- 打包后 `info.json` 的 CRLF 已还原为 LF（UTF-8 无 BOM），diff 只剩 `ProductVersion` 一行。

---

# Verification

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify
- [x] 原生版本资源校验
- [x] 发布资产校验

## Verification Result

```text
build-release.ps1 -Version v2.3.1-beta.8 -OutputDirectory release-assets\v2.3.1-beta.8
  set-release-version            → 已校验 v2.3.1-beta.8，Windows 数字版本 2.3.1.0
  task --force verify            → 通过（Go 全包、前端 155 文件／943 项测试、Windows production build）
  task --force package           → 安装器 21,085,494 bytes
  test-windows-metadata          → Windows 应用和安装器版本资源校验通过
  verify-release-assets          → 发布资产校验通过：v2.3.1-beta.8
SCRIPT_EXIT=0
```

未签名、未自动安装、未启动、未打标签、未推送。

---

# Documentation

- [x] 发行说明：`docs/releases/v2.3.1-beta.8.md`
- [x] CURRENT：追加本版本摘要
- [x] 其余文档无需修改

---

# Git

Not committed by Agent.

---

# Result

已产出 `v2.3.1-beta.8` 本地 Windows amd64 生产安装包，供用户从托盘完全退出应用并备份数据后
覆盖安装。版本内容为 HowToFix 官方模型目录自动同步：启动与账号／服务商变更后后台补齐缺失目录，
路由页保留手动重新同步，账号页不再显示过期的余额提示。本版本无数据库迁移。

---

# Completed

2026-11-16 08:36（Asia/Singapore）
