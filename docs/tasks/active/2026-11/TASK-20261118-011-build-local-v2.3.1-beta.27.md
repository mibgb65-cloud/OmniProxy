# TASK-20261118-011: 构建 v2.3.1-beta.27 本地安装包

## Status

Completed

## Created

2026-11-18 21:34

## Request

用户在 TASK-20261118-010（首页模型 Token 占比 + 用量页年度活跃度热图）完成后要求「提交代码，然后构建新的 beta 生产版本我进行本地更新」。

## Goal

产出 `v2.3.1-beta.27` Windows amd64 生产安装包（未签名、本地使用），供本机覆盖更新后查看两处新增界面：

- TASK-20261118-010：仪表盘「模型 Token 占比」（`0c6bbe8c`）；
- TASK-20261118-010：用量页「年度活跃度」热图（同一提交）。

## Background

上一轮已提交功能与文档（`0c6bbe8c` 功能、`70c75c9d` 归档），工作区无未提交的代码改动。
版本基线仍为 `2.3.1-beta.26`，`release-assets/v2.3.1-beta.27` 尚不存在。

## Scope

- `docs/releases/v2.3.1-beta.27.md` 版本说明（中英）；
- `internal/version/version.go`、`frontend/package.json` 版本号 → `2.3.1-beta.27`；
- `scripts/build-release.ps1 -Version v2.3.1-beta.27 -OutputDirectory release-assets\v2.3.1-beta.27`；
- 记录产物大小与 SHA256，提交构建脚本生成的 `build/windows/info.json`。

## Out of Scope

- 打标签、推送、公开 Release、代码签名、自动安装或启动应用。
- 数据库迁移（本版本无）。
- 桌面窗口内的实机目视验收（由用户安装后完成）。

## Existing Behavior

本机运行的是 `v2.3.1-beta.26` 之前的构建；`v2.3.1-beta.26` 安装包已产出但用户尚未安装。

## Expected Behavior

安装 `v2.3.1-beta.27` 后：仪表盘出现「模型 Token 占比」卡片并随时段切换；用量页出现「年度活跃度」热图，可切换 Token／请求数并悬浮查看单日数据；既有统计数字不变。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`（版本号）
- `docs/releases/`（版本说明）
- `scripts/build-release.ps1`、`scripts/set-release-version.ps1`、`scripts/test-windows-metadata.ps1`、
  `scripts/verify-release-assets.ps1`（构建与校验链）
- `build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`（脚本写入的版本资源）

## Dependencies

- NSIS（`makensis`，脚本会在 `Program Files (x86)\NSIS` 兜底查找）
- `task`、`wails3`、Go、pnpm —— 上一轮 `task verify` 已全部可用

## Risks

- 构建脚本内部会再跑一次 `task --force verify`（约 5–6 分钟）并重新生成绑定与前端产物；
  若源码在提交后又被改动，安装包内容会与提交不一致，因此构建前后都要核对 `git status`。
- `release-assets/<version>` 必须为空目录；重跑同版本需先清理，脚本会拒绝覆盖。

---

# Implementation Plan

1. 写版本说明与版本号，提交 `chore(release)`。
2. 运行 `scripts/build-release.ps1`（内部执行 `task --force verify` 与 `task --force package`）。
3. 校验产物大小、SHA256 与 Windows 版本资源。
4. 回填构建结果、归档本记录。

---

# Progress

- 2026-11-18 21:34 创建记录，写版本说明与版本号。

---

# Implementation Result

## Changes

- 新增 `docs/releases/v2.3.1-beta.27.md`（中英版本说明：两处新增界面、无不兼容变更、本地更新步骤、
  验证边界与未目视确认的范围）。
- 版本号：`internal/version/version.go`、`frontend/package.json` → `2.3.1-beta.27`；
  `build/windows/info.json` 的 `ProductVersion` 同步为 `2.3.1-beta.27`。
- 安装包：`release-assets/v2.3.1-beta.27/OmniProxy-Setup-v2.3.1-beta.27-windows-amd64.exe`
  （21,410,046 bytes，SHA256 `864c35516877d32a80163b308aef125d41dd185e7fc1f393d811796e42f95327`），
  附 `.sha256`。
- 构建脚本自动写入的 `build/config.yml`（数字版本 `2.3.1`）与
  `build/windows/nsis/wails_tools.nsh`（`INFO_PRODUCTVERSION "2.3.1"`）与既有值相同，没有产生改动。

## Files Changed

- `docs/releases/v2.3.1-beta.27.md`
- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- 产物目录 `release-assets/v2.3.1-beta.27/`（被 `.gitignore` 忽略，不入库）

## Technical Notes

- `scripts/build-release.ps1` 内部依次执行 `task --force verify`、`task --force package`、
  Windows 应用／安装器版本资源校验与发布资产校验，任一失败即中止。
- 脚本用 Windows PowerShell 写 `build/windows/info.json` 会产生 CRLF；按仓库 `.gitattributes`
  的要求已还原为 LF（内容与提交一致，`git diff` 为空）。
- 安装包未签名（按项目 `codeSigningPolicy`），未自动安装、未启动应用、未打标签、未推送。

---

# Verification

## Tests

- [x] 构建脚本内置的 `task --force verify`（格式、lint、`tsc --noEmit`、Go 全包、前端 162 文件／990 项测试、Windows 生产构建）
- [ ] 真实请求验证：本版本只改界面，由用户安装后打开仪表盘与用量页确认
- [ ] 桌面窗口目视确认：未执行（构建期无法观察 GUI）

## Quality Checks

- [x] Format／Lint／Type Check／Test／Build
- [x] Windows 版本资源校验：`Windows 应用和安装器版本资源校验通过`
- [x] 发布资产校验：`发布资产校验通过：v2.3.1-beta.27`

## Verification Result

```text
scripts\build-release.ps1 -Version v2.3.1-beta.27 -OutputDirectory release-assets\v2.3.1-beta.27
BUILD_EXIT=0
安装器：21,410,046 bytes
SHA256：864c35516877d32a80163b308aef125d41dd185e7fc1f393d811796e42f95327
Windows 数字版本：2.3.1.0；NSIS 基础版本：2.3.1
前端测试：162 文件／990 项通过
```

---

# Documentation

- [x] 版本说明 `docs/releases/v2.3.1-beta.27.md`
- [x] CURRENT
- [ ] 其它：No documentation changes required.

---

# Deviations

- 计划里写「提交构建脚本生成的 `build/windows/info.json`」，实际该文件的版本字段已在
  `dc379d4b` 一并提交，构建后只有换行符差异（已还原为 LF），因此本轮没有 info.json 的内容变更。

---

# Git

## Commit

```text
dc379d4b chore(release): 准备 v2.3.1-beta.27 本地构建
```

---

# Result

`v2.3.1-beta.27` 本地安装包已产出并校验通过，包含仪表盘「模型 Token 占比」与用量页「年度活跃度」
两处新增界面；未签名、未安装、未启动、未打标签、未推送，无数据库迁移。

---

# Completed

2026-11-18 21:45
