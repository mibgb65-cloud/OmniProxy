# TASK-20261113-010: 构建本地 v2.2.3-beta.19 生产安装包

## Status

Completed

## Created

2026-11-13 11:26（Asia/Singapore）

## Request

Google 卡片移除重复手动同步按钮（TASK-20261113-009）需要可安装的本地 Beta；
该版本同时把尚未交付的 beta.18（内联图片修复）与 beta.17 的内容一起交付。

## Goal

- 版本由 `2.2.3-beta.18` 提升为 `2.2.3-beta.19`，Windows 数字版本保持 `2.2.3.0`、
  NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.19.md`。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。

## Background

- `v2.2.3-beta.18` 已产出但用户尚未确认安装，本轮直接以 `beta.19` 交付更省事。
- 本版累积内容：Google 订阅账号轮换设置（TASK-20261113-005，迁移 047）、型号档位优先
  （TASK-20261113-004）、Antigravity 内联图片（TASK-20261113-007）、Google 卡片去掉
  重复的手动同步按钮（TASK-20261113-009）。
- `task --force verify` exit 0（148 文件／901 项前端测试、全部 Go 包、Windows production 构建）。

## Scope

- 版本映射、发行说明、生产打包、发布资产与摘要、任务记录与 CURRENT。

## Out of Scope

- 不重跑 `task verify`（实现任务已通过）。
- 不推送、不打标签、不公开发布、不签名、不安装、不启动。

## Existing Behavior

源码版本为 `2.2.3-beta.18`。

## Expected Behavior

覆盖安装后 Google 卡片不再显示手动同步按钮，图片消息可用，账号轮换设置可见。

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
2. 更新版本并运行 `scripts/set-release-version.ps1 -Version v2.2.3-beta.19`，还原 `info.json` 的 LF。
3. 新增双语发行说明。
4. `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.19/bin`。
5. 复制安装器、生成 SHA256 侧车、运行两项校验脚本，归档并提交。

---

# Progress

- 2026-11-13 11:26：创建记录。

---

# Implementation Result

## Changes

- 应用、前端与 Windows 显示版本统一为 `2.2.3-beta.19`；Windows 数字资源 `2.2.3.0`、
  NSIS 基础版本 `2.2.3` 不变。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.19.md`：Google 卡片瘦身、内联图片、
  Google 轮换设置、型号档位优先，以及迁移 047 与桌面端 60 秒上限的边界。
- 版本包含 TASK-20261113-004／005／007／009。

## Build and Artifacts

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.19/OmniProxy-Setup-v2.2.3-beta.19-windows-amd64.exe` | 20,755,171 | `f3dbe97bbb4e06eb8bda5511414847b32511c4bfd73900e2eb72c1ef3fbae4d5` |
| `tmp/local-v2.2.3-beta.19/bin/omniproxy.exe` | 23,862,272 | `ec9b870e1a9c7d3ab841b48796847ba07cb6171bb8cefa806b7dd3ac2cd3f3bb` |

摘要侧车与安装器同目录；应用与安装器 Authenticode 均为 `NotSigned`。

---

# Verification

## Quality Checks

- [x] Format：版本文件行尾保持 LF。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.19/bin` exit 0。
- [x] Verify：实现任务 `task --force verify` exit 0。

## Verification Result

```text
set-release-version v2.2.3-beta.19  → 已校验，Windows 数字版本 2.2.3.0
package                              → PACKAGE_EXIT=0（20,755,171 bytes）
test-windows-metadata                → 版本资源校验通过
verify-release-assets                → 发布资产校验通过：v2.2.3-beta.19
```

---

# Documentation

- [x] CURRENT
- [x] `docs/releases/v2.2.3-beta.19.md`

---

# Git

## Commit

见归档提交。未打标签、未推送、未公开发布。

---

# Result

`v2.2.3-beta.19` Windows amd64 本地 Beta 生产安装器已生成，一次性交付 Google 卡片瘦身、
Antigravity 内联图片、Google 轮换设置与型号档位修复。安装器未签名、未自动安装、未启动。

---

# Completed

2026-11-13 11:34
