# TASK-20261031-002: 构建本地 v2.2.1-beta.4 生产安装包

## Status

Completed

## Created

2026-10-31 12:25

## Request

> 提交代码，然后构建新的 beta 生产版本我进行本地更新，记得完成任务 md 更新

## Goal

产出 Windows amd64 production 安装包 `v2.2.1-beta.4`，包含账号页展开面板卡片列宽修复
（TASK-20261031-001，`6c934061`），完成质量门、安装器元数据与发布资产校验，本地提交并归档。

## Background

本地上一份产物为 `v2.2.1-beta.3`（TASK-20261030-006，安装器 20,356,443 bytes）。
此后 `main` 上新增一条用户可见改动尚未出包：`6c934061`
fix(accounts) 展开面板账号卡片恢复列宽封顶并按可用宽度增列（TASK-20261031-001）。

## Scope

- 版本源递增至 `2.2.1-beta.4`，新增对应双语发布说明。
- `scripts/build-release.ps1` 构建、打包、生成 SHA256 并校验版本资源与资产目录。
- 同步 CURRENT 与两份任务记录。

## Out of Scope

自动安装或停止用户运行中的应用、签名、打标签、推送、公开发布、修改业务代码或数据库迁移。

## Existing Behavior

本地最新安装包为 `v2.2.1-beta.3`；公开正式版仍为 `v2.2.0`。

---

# Implementation Plan

1. `internal/version/version.go` 与 `frontend/package.json` 改为 `2.2.1-beta.4`。
2. 新增 `docs/releases/v2.2.1-beta.4.md`（中英双语）。
3. 运行 `scripts/build-release.ps1 -Version v2.2.1-beta.4 -OutputDirectory release-assets/v2.2.1-beta.4`。
4. 校验 EXE、安装器、SHA256、版本资源与目录内容，`build/windows/info.json` 保持 LF。
5. 回填结果与验证、更新 CURRENT、提交并归档。

---

# Progress

- 第一次运行发布脚本时，`prettier --check` 因 `frontend/package.json` 被写入 UTF-8 BOM 失败（质量门在打包前中止，未产生任何资产）。
- 去掉 BOM 后重跑，脚本完成质量门、NSIS 打包与资产校验，exit 0。

---

# Implementation Result

## 版本源

| 位置                                                      | 值                                                  |
| --------------------------------------------------------- | --------------------------------------------------- |
| `internal/version/version.go`                             | `2.2.1-beta.4`                                      |
| `frontend/package.json`                                   | `2.2.1-beta.4`                                      |
| `build/windows/info.json`                                 | 数字版本 `2.2.1.0`／`ProductVersion` `2.2.1-beta.4` |
| `build/config.yml` / `build/windows/nsis/wails_tools.nsh` | `2.2.1`（不含预发布标识，本次未变）                 |

## 新增发布说明

`docs/releases/v2.2.1-beta.4.md`（中英双语）：展开面板卡片恢复「单卡封顶 28rem + 宽屏自动增列」
（同一容器宽度从 2 列 × ≈1075px 变为 4 列 × 448px）、窄面板单列时用满上限、新增 CSS 断言测试
禁止再写固定列数；同时说明本次只改两份 CSS 与一项测试。

## 构建与产物

命令：`scripts/build-release.ps1 -Version v2.2.1-beta.4 -OutputDirectory release-assets/v2.2.1-beta.4`
（日志 `tmp/release-v2.2.1-beta.4.log`，exit 0）：`set-release-version.ps1` → `task --force verify`
→ `task --force package`（NSIS）→ `test-windows-metadata.ps1` → 复制安装器与 `.sha256` → `verify-release-assets.ps1`。

| 文件                                                                           | 大小             |
| ------------------------------------------------------------------------------ | ---------------- |
| `release-assets/v2.2.1-beta.4/OmniProxy-Setup-v2.2.1-beta.4-windows-amd64.exe` | 20,357,883 bytes |
| 同上 `.sha256`                                                                 | 114 bytes        |
| `bin/omniproxy.exe`（内嵌前端，production）                                    | 23,304,192 bytes |

SHA256（安装器，`.sha256` 与独立复算一致）：`85727445caac9adb51d2e2636c42819006d7d5670eada16f9eb9384b4c435475`
SHA256（`bin/omniproxy.exe`）：`9b4cff9a79b09119e6c0992e5131d3967aff524e9ccb81c78e2848818e208d42`

NSIS 统计：`Total size: 20357883 / 81211983 bytes (25.0%)`。比 `v2.2.1-beta.3`（20,356,443 bytes）增加
1,440 bytes，与「两份 CSS + 一项测试（不进产物）+ 新增发布说明（不进产物）」的量级相符。

安装器版本资源：`FileVersion` / `ProductVersion` 均为 `2.2.1`（NSIS 只取数字段；完整语义版本在应用内
`ProductVersion` 与 `internal/version` 中为 `2.2.1-beta.4`）。

---

# Verification

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

| 项目                                | 结果                                                 |
| ----------------------------------- | ---------------------------------------------------- |
| 发布脚本内 `task --force verify`    | exit 0                                               |
| gofmt / prettier `format:check`     | 通过（`All matched files use Prettier code style!`） |
| `go vet` / `staticcheck` / `eslint` | 通过（仅既有 2 条 fast-refresh 警告）                |
| `tsc` typecheck                     | 通过                                                 |
| Vitest                              | 146 文件／890 项全部通过                             |
| Go 全量包测试                       | 通过                                                 |
| Windows production 构建             | 通过                                                 |
| NSIS 打包                           | 通过（25.0%）                                        |
| `test-windows-metadata.ps1`         | 「Windows 应用和安装器版本资源校验通过」             |
| `verify-release-assets.ps1`         | 「发布资产校验通过：v2.2.1-beta.4」                  |
| SHA256 独立复算                     | 与 `.sha256` 一致（`85727445…5475`）                 |

未覆盖（需用户实机确认）：真实覆盖安装（本次构建不自动安装、不停止正在运行的应用）、
原生 WebView2 下全屏展开面板的卡片列数与留白观感、140% 字号。

---

# Documentation

- 新增 `docs/releases/v2.2.1-beta.4.md`（中英双语）。
- 更新 `docs/plans/CURRENT.md`：顶部摘要、`## Current Version`、`## Recently Completed`。
- 回填 TASK-20261031-001 的 Git 段（`6c934061`）。

---

# Deviations

1. 第一次构建在质量门阶段失败：`frontend/package.json` 被 `Set-Content -Encoding utf8` 写入 UTF-8 BOM，
   `prettier --check` 报 `Invalid package config … Unexpected token '﻿'`。改为 `UTF8Encoding($false)` 重写后通过；
   首次失败发生在打包之前，未产生半成品资产，`release-assets/v2.2.1-beta.4` 由成功的那次创建。
2. `build/windows/info.json` 被发布脚本按 CRLF 重写（15 处 CR），提交前已还原为 LF（`CR count: 0`）。
3. 日志里 `exit 1` 是 Taskfile gofmt 检查分支的脚本正文被回显（Windows 缺 `uname` 引起的
   NativeCommandError 渲染），不是失败；脚本随后完成 NSIS 打包、版本资源与资产校验。
4. 数字版本仍为 `2.2.1.0`（patch 段不变），与 `2.2.1-beta.3` 相同，符合既有映射规则。

---

# Git

## Commit

```text
696954f2 chore(release): 构建 2.2.1-beta.4 本地生产安装包
```

未推送、未打标签、未签名。

---

# Result

`v2.2.1-beta.4` Windows amd64 production 安装包已产出并通过完整质量门（146 文件／890 项前端测试）、
NSIS 打包、Windows 版本资源与发布资产校验；安装器 20,357,883 bytes，
SHA256 `85727445caac9adb51d2e2636c42819006d7d5670eada16f9eb9384b4c435475`。
内容为账号页展开面板卡片列宽修复，未签名、未自动安装、未打标签、未推送、未公开发布，等待用户本地覆盖更新。

---

# Completed

2026-10-31 13:39
