# TASK-20261119-004: 构建 v2.3.1-beta.29 本地安装包

## Status

Completed

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

- 2026-11-19 11:38 创建记录，写版本说明与版本号。
- 2026-11-19 11:39 `set-release-version.ps1` 通过；提交 `b18f5f03`（准备构建）。
- 2026-11-19 11:41 首次构建在质量门的 `frontend:test` 处失败（1000 项中 1 项超时）；单跑完整
  前端套件 1000/1000 通过，判定为并行负载下的计时抖动，非本次改动。
- 2026-11-19 11:43 重跑构建 EXIT=0，安装器与发布资产校验通过。

---

# Implementation Result

## Changes

- 新增 `docs/releases/v2.3.1-beta.29.md`（中英版本说明：额度参与选号、429 账号级化、429 后刷新快照、
  分组归类口径与 fail-open 边界、本地更新步骤与验证边界）。
- 版本号：`internal/version/version.go`、`frontend/package.json` → `2.3.1-beta.29`；
  `build/windows/info.json` 的 `ProductVersion` 同步为 `2.3.1-beta.29`。
- 安装包：`release-assets/v2.3.1-beta.29/OmniProxy-Setup-v2.3.1-beta.29-windows-amd64.exe`
  （21,435,843 bytes，SHA256 `2d6ae07d33fb1e9bd6b7f6b23a1551f7bd25b40ea49755753ae625e311b79fd5`），
  附 `.sha256`。
- 数字资源（`build/config.yml` 的 `2.3.1`、`build/windows/nsis/wails_tools.nsh` 的
  `INFO_PRODUCTVERSION "2.3.1"`）与既有值相同，没有产生改动。

## Files Changed

- `docs/releases/v2.3.1-beta.29.md`
- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- 产物目录 `release-assets/v2.3.1-beta.29/`（被 `.gitignore` 忽略，不入库）

## Technical Notes

- 构建脚本用 Windows PowerShell 写 `build/windows/info.json` 会产生 CRLF；已还原为 LF，
  `git status` 对 `build/` 无改动。
- 安装包未签名（按项目 `codeSigningPolicy`），未自动安装、未启动、未打标签、未推送。

---

# Verification

## Tests

- [x] 构建脚本内置的 `task --force verify`（格式、lint、`tsc --noEmit`、Go 全包、前端
  163 文件／1000 项测试、Windows 生产构建）
- [ ] 真实账号验收：本版本不消耗额度，由用户安装后观察下一次额度耗尽时的换号行为

## Quality Checks

- [x] Format／Lint／Type Check／Test／Build
- [x] Windows 版本资源校验：`Windows 应用和安装器版本资源校验通过`
- [x] 发布资产校验：`发布资产校验通过：v2.3.1-beta.29`

## Verification Result

```text
scripts\build-release.ps1 -Version v2.3.1-beta.29 -OutputDirectory release-assets\v2.3.1-beta.29
第 1 次：BUILD_EXIT=1（frontend:test 1/1000 超时，单跑 1000/1000 通过）
第 2 次：BUILD_EXIT=0
安装器：21,435,843 bytes
SHA256：2d6ae07d33fb1e9bd6b7f6b23a1551f7bd25b40ea49755753ae625e311b79fd5
Windows 数字版本：2.3.1.0；NSIS 基础版本：2.3.1
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.3.1-beta.29
```

---

# Documentation

- [x] 版本说明 `docs/releases/v2.3.1-beta.29.md`
- [x] CURRENT
- [ ] 其它：`docs/product/GOOGLE_SUBSCRIPTION.md`、`QUOTA.md`、`PRD.md`、ADR-015 已在
  TASK-20261119-003 更新，本轮无新增

---

# Deviations

- 首次构建因前端测试计时抖动失败，未修改任何测试或门禁阈值；重跑完整套件与构建均通过。
  该抖动在 TASK-20261119-001／002 中已出现过同型（`stepfun.test.tsx` 全量并行超时），
  记录在案但不视为本次改动引起。

---

# Git

## Commit

```text
b18f5f03 chore(release): 准备 v2.3.1-beta.29 本地构建
```

本任务记录与 `docs/plans/CURRENT.md` 的更新随 `chore(release): 记录 v2.3.1-beta.29 构建产物`
一并提交。

---

# Result

`v2.3.1-beta.29` 本地安装包已产出并校验通过，内容为 Google 订阅账号按额度窗口跳过与 429 换号
（TASK-20261119-003）；未签名、未安装、未启动、未打标签、未推送，无数据库迁移。

---

# Completed

2026-11-19 11:46
