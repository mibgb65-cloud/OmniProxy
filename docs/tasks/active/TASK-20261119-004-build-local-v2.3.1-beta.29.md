# TASK-20261119-004: 构建 v2.3.1-beta.29 本地安装包

## Status

In Progress

## Created

2026-11-19 11:38

## Request

用户在 TASK-20261119-003（Google 订阅账号按额度窗口参与选号）完成后继续本地更新流程（沿用
TASK-20261118-011／TASK-20261119-002 的「提交代码 → 构建 beta 生产版本」约定）。

## Goal

产出 `v2.3.1-beta.29` Windows amd64 生产安装包（未签名、本地使用），供本机覆盖更新后立即生效：

- TASK-20261119-003：Google 订阅账号按额度窗口跳过与 429 换号（`dca88996`）。

## Background

功能与文档已提交（`dca88996` 功能、`bff0730b` 记录），工作区只有既有的未跟踪任务记录
（`docs/tasks/completed/2026-10/TASK-20261029-010-subq-provider.md`，不属于本任务）。
版本基线 `2.3.1-beta.28`，`release-assets/v2.3.1-beta.29` 尚不存在。

## Scope

- `docs/releases/v2.3.1-beta.29.md` 版本说明（中英）；
- 版本号 → `2.3.1-beta.29`（`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json`）；
- `scripts/build-release.ps1 -Version v2.3.1-beta.29 -OutputDirectory release-assets\v2.3.1-beta.29`；
- 记录产物大小与 SHA256。

## Out of Scope

- 打标签、推送、公开 Release、代码签名、自动安装或启动应用；
- 数据库迁移（本版本无）；
- 真实账号验收（由用户安装后观察，避免消耗额度）。

## Existing Behavior

本机运行 `v2.3.1-beta.28`：Google 订阅账号的 429 不换号、不冷却，额度耗尽后仍会被轮换选中。

## Expected Behavior

安装后：额度耗尽的分组模型不再被发送；首次 429 在同一请求内换号并冷却到真实重置时间；
429 后快照自动刷新；全部候选不可用时返回 429 `account_pool_cooling`。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`（版本号与资源）
- `docs/releases/`（版本说明）
- `scripts/set-release-version.ps1`、`scripts/build-release.ps1`（构建与校验链）

## Dependencies

- NSIS、`task`、`wails3`、Go、pnpm —— 上一轮已验证可用。

## Risks

- 构建脚本内部再跑一次 `task --force verify`；构建前后核对 `git status`，避免安装包与提交不一致。
- `release-assets/<version>` 必须为空目录，脚本拒绝覆盖。

---

# Implementation Plan

1. 写版本说明与版本号，同步 Windows 版本资源，提交 `chore(release)`。
2. 运行 `scripts/build-release.ps1`。
3. 校验大小、SHA256 与版本资源。
4. 回填记录、归档、更新 CURRENT。

---

# Progress

- 2026-11-19 11:38 创建记录。

---

# Implementation Result

待填写。

---

# Verification

## Tests

- [ ] 构建脚本内置的 `task --force verify`
- [ ] Windows 版本资源校验
- [ ] 发布资产校验
- [ ] 真实账号验收：未执行（由用户安装后观察）

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

- [ ] 版本说明 `docs/releases/v2.3.1-beta.29.md`
- [ ] CURRENT

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
