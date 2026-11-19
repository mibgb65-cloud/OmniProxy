# TASK-20261119-002: 构建 v2.3.1-beta.28 本地安装包

## Status

In Progress

## Created

2026-11-19 10:36

## Request

用户在 TASK-20261119-001（Google 账号页按额度分组分区）完成并提交后要求：

> 完成后，提交代码，然后构建新的 beta 生产版本我进行本地更新，记得完成任务md更新

代码与任务记录已在 `da458b60`、`675e144f` 提交，本轮只做版本号、版本说明与本地生产包。

## Goal

产出 `v2.3.1-beta.28` Windows amd64 生产安装包（未签名、本地使用），供本机覆盖更新后查看
Google 账号页的额度分组分区：

- TASK-20261119-001：Google 账号页按官方额度分组集合分区（`da458b60`）。

## Background

上一轮已提交功能与文档（`da458b60` 功能、`675e144f` 归档），工作区仅有既有的未跟踪任务记录
（`docs/tasks/completed/2026-10/TASK-20261029-010-subq-provider.md`，不属于本任务，保持不动）。
版本基线为 `2.3.1-beta.27`，`release-assets/v2.3.1-beta.28` 尚不存在。

## Scope

- `docs/releases/v2.3.1-beta.28.md` 版本说明（中英）；
- `internal/version/version.go`、`frontend/package.json` 版本号 → `2.3.1-beta.28`；
- `scripts/set-release-version.ps1 -Version v2.3.1-beta.28` 同步 Windows 版本资源；
- `scripts/build-release.ps1 -Version v2.3.1-beta.28 -OutputDirectory release-assets\v2.3.1-beta.28`；
- 记录产物大小与 SHA256，提交构建脚本生成的版本资源文件。

## Out of Scope

- 打标签、推送、公开 Release、代码签名、自动安装或启动应用。
- 数据库迁移（本版本无）。
- 桌面窗口内的实机目视验收（由用户安装后完成）。

## Existing Behavior

本机运行的是 `v2.3.1-beta.27` 或更早的构建；Google 账号页按后端返回顺序平铺账号卡。

## Expected Behavior

安装 `v2.3.1-beta.28` 后：Google 账号页把订阅账号按额度分组集合分区，含 `Gemini 4 Argon`
这类额外分组的账号自成一区并排在前面，API Key 账号与额度未知的账号排在最后；既有的额度数值、
模型列表、启停与删除操作不变。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`（版本号）
- `docs/releases/`（版本说明）
- `scripts/set-release-version.ps1`、`scripts/build-release.ps1`、`scripts/test-windows-metadata.ps1`、
  `scripts/verify-release-assets.ps1`（构建与校验链）
- `build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`（脚本写入的版本资源）

## Dependencies

- NSIS（`makensis`，脚本会在 `Program Files (x86)\NSIS` 兜底查找）
- `task`、`wails3`、Go、pnpm —— 上一轮 `task verify` 已全部可用

## Risks

- 构建脚本内部会再跑一次 `task --force verify`（约 5–6 分钟）并重新生成绑定与前端产物；
  若源码在提交后又被改动，安装包内容会与提交不一致，因此构建前后都要核对 `git status`。
- `release-assets/<version>` 必须为空目录；脚本会拒绝覆盖。

---

# Implementation Plan

1. 写版本说明与版本号，运行 `set-release-version.ps1` 同步 Windows 版本资源，提交 `chore(release)`。
2. 运行 `scripts/build-release.ps1`（内部执行 `task --force verify` 与 `task --force package`）。
3. 校验产物大小、SHA256 与 Windows 版本资源。
4. 回填构建结果、归档本记录、更新 `docs/plans/CURRENT.md`。

---

# Progress

- 2026-11-19 10:36 创建记录，开始写版本说明与版本号。

---

# Implementation Result

任务完成后填写。

---

# Verification

## Tests

- [ ] 构建脚本内置的 `task --force verify`
- [ ] Windows 版本资源校验
- [ ] 发布资产校验
- [ ] 真实请求验证：本版本只改界面，由用户安装后确认
- [ ] 桌面窗口目视确认：未执行（构建期无法观察 GUI）

## Quality Checks

- [ ] Format
- [ ] Lint
- [ ] Type Check
- [ ] Test
- [ ] Build
- [ ] Verify

## Verification Result

待填写。

---

# Documentation

- [ ] 版本说明 `docs/releases/v2.3.1-beta.28.md`
- [ ] CURRENT
- [ ] 其它：N/A

---

# Deviations

待填写。

---

# Git

## Commit

待填写。

---

# Result

待填写。

---

# Completed

待填写。
