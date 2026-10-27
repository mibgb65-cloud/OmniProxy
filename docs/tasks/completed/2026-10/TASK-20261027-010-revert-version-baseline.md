# TASK-20261027-010: 撤销版本基线记录与版本号改动

## Status

Completed

## Created

2026-10-27 22:00（Asia/Singapore）

## Request

用户：「我现在还不发版本，撤销这个版本记录啊」。

## Goal

撤销本轮两次自行推进的版本动作，让仓库回到「v2.0.0 刚发布、没有在建版本」的状态：

- 版本源 5 处从 `2.0.1` 退回已发布的 `2.0.0`（含 Windows `2.0.0.0` 与 NSIS 定义）。
- [TASK-20261027-008](TASK-20261027-008-v2.1.0-development-baseline.md) 与
  [TASK-20261027-009](TASK-20261027-009-version-baseline-2.0.1.md) 改判 Cancelled。
- `docs/plans/CURRENT.md` 删掉「开发基线」表述，只保留已发布的 2.0.0 与遗留跟进项。

## Background

- TASK-20261027-008 自行把基线提升到 `2.1.0`，被用户否决。
- TASK-20261027-009 改成 `2.0.1`，用户确认「现在还不发版本」，因此连 `2.0.1` 也一并撤销。
- 两次改动都只涉及版本常量、构建资源与文档，从未打包、打标签、推送或发布。

## Scope

- 5 个版本源退回 `2.0.0`／`2.0.0.0`。
- TASK-20261027-008、TASK-20261027-009：Status → `Cancelled`，各加 `## Cancellation Reason`，
  原内容不改写。
- `docs/plans/CURRENT.md`：Current Version 只保留已发布 2.0.0；In Progress 为「无」；
  跟进清单标题回归 `### Post-release Follow-ups`。
- `task --force verify`。

## Out of Scope

- 不改业务逻辑、迁移、UI。
- 不删任何历史文件（Cancelled 记录按 `docs/tasks/README.md` 保留在 `completed/`）。
- 不动 TASK-20261027-007 的归档结论与 Atria 草稿的 Completed 标记（那是用户明确要求的）。

## Existing Behavior

版本源为 `2.0.1`，`CURRENT.md` 写着「开发基线 2.0.1」，存在两个已完成的版本基线任务记录。

## Expected Behavior

版本源恢复为 `2.0.0`（与已发布正式版一致，`git diff` 为空）；`CURRENT.md` 只声明已发布版本；
008／009 作为被否决的过程记录留在 `completed/`。

---

# Analysis

## Related Modules

- 版本源 5 处、`docs/plans/CURRENT.md`、`docs/tasks/completed/2026-10/` 两条记录。

## Dependencies

- 无。

## Risks

- 若 008／009 期间曾产出可分发产物，退回会产生版本倒退：已确认从未打包／打标签／推送／发布。
- 记录改判若删掉原文，会丢失「Agent 曾自行决定版本级别」的教训：故保留原文，只加
  Cancellation Reason。

---

# Implementation Plan

1. 版本源 5 处退回 `2.0.0`。
2. 008／009 改 Cancelled 并写明原因。
3. `CURRENT.md` 去掉开发基线段落、还原跟进清单标题。
4. `task --force verify` 并回填。

---

# Progress

- 已确认两条记录涉及的版本号从未进入任何分发产物。

---

# Implementation Result

## Changes

- 版本源 5 处：`2.0.1` → `2.0.0`（Windows 数字版本 `2.0.0.0`，NSIS `INFO_PRODUCTVERSION "2.0.0"`）。
- TASK-20261027-008、TASK-20261027-009：Status → `Cancelled`，加 `## Cancellation Reason`。
- `docs/plans/CURRENT.md`：删除开发基线段落；`In Progress` 段写明两条版本记录已撤销；
  跟进清单标题回归 `### Post-release Follow-ups`。

## Files Changed

- `internal/version/version.go`、`frontend/package.json`、`build/config.yml`、
  `build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/plans/CURRENT.md`
- `docs/tasks/completed/2026-10/TASK-20261027-008-v2.1.0-development-baseline.md`
- `docs/tasks/completed/2026-10/TASK-20261027-009-version-baseline-2.0.1.md`
- 本记录

## Technical Notes

- 撤销后 `git diff` 对 5 个版本源为空，说明工作树与已发布 `2.0.0` 源码一致。
- 版本号/发布节奏由用户决定：未收到明确版本号前不再改版本源。

---

# Verification

## Tests

- [ ] Unit Tests（版本常量由 `go test . ./internal/...` 覆盖）
- [ ] Integration Tests（N/A）
- [ ] E2E Tests（N/A）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task --force verify → exit 0
  format:check / go vet / staticcheck / eslint / tsc --noEmit → PASS
  go test ./internal/... → PASS
  vitest → Test Files 132 passed，Tests 809 passed（99.47 s）
  wails3 production build → bin/omniproxy.exe（2.0.0 版本资源）
```

版本源核对：

```text
internal/version/version.go          → var Version = "2.0.0"
frontend/package.json               → "version": "2.0.0"
build/config.yml                    → version: "2.0.0"
build/windows/info.json             → file_version/product_version "2.0.0.0"，ProductVersion "2.0.0"
build/windows/nsis/wails_tools.nsh  → !define INFO_PRODUTVERSION 实为 INFO_PRODUCTVERSION "2.0.0"
git diff -- <5 个版本源>            → 空（与已发布 2.0.0 源码一致）
```

记录状态核对：TASK-20261027-008、TASK-20261027-009 均为 `Cancelled`，各带
`## Cancellation Reason`；`docs/plans/CURRENT.md` 中已无「开发基线」表述，
`### Post-release Follow-ups（v2.0.0 交付后待人工验收）` 保留 8 条。

---

# Documentation

- [x] CURRENT
- 其他：无

---

# Deviations

- 无。

---

# Git

## Commit

```text
a453fdab docs(tasks): 归档 v2.0.0 之前的任务记录并撤销版本基线改动
（本任务的撤销动作与 008／009 的 Cancelled 改判随该提交入库）
```

---

# Result

版本动作已全部撤销，仓库回到「v2.0.0 已发布、无在建版本」状态：5 个版本源退回 `2.0.0`／
`2.0.0.0` 且与 HEAD 一致（`git diff` 为空），TASK-20261027-008／009 改判 Cancelled 并保留
原始内容与被否决的事实，`CURRENT.md` 只声明已发布 2.0.0 与 8 条遗留跟进项。
`task --force verify` exit 0（132 文件／809 项前端测试、Windows production 构建通过）。
未打包、未打标签、未推送、未发布。

## Follow-ups

- 版本号／发布节奏由用户决定；未收到明确版本号前不改版本源。
- 用户确定版本后再改 5 个版本源并补对应的 `docs/releases/v<版本>.md`。

---

# Completed

2026-10-27 22:15（Asia/Singapore）
