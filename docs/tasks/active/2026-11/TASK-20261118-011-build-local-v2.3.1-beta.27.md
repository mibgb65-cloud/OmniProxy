# TASK-20261118-011: 构建 v2.3.1-beta.27 本地安装包

## Status

In Progress

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

（进行中）

---

# Verification

（待构建完成后填写）

---

# Documentation

- [ ] 版本说明 `docs/releases/v2.3.1-beta.27.md`
- [ ] CURRENT

---

# Git

## Commit

（未开始）

---

# Result

（未开始）

---

# Completed

（未完成）
