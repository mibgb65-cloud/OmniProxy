# TASK-20261027-009: 开发基线更正为 v2.0.1

## Status

Cancelled

## Created

2026-10-27 21:35（Asia/Singapore）

## Request

用户：「你不要随意给我提升开发基线，打算的下个版本是 v2.0.1」。

## Goal

把上一轮擅自提升的 `2.1.0` 开发基线更正为 `2.0.1`，并让文档、任务记录与版本源一致。

## Background

- [TASK-20261027-008](TASK-20261027-008-v2.1.0-development-baseline.md) 未经用户确认就把版本
  基线从已发布的 `2.0.0` 提升到 `2.1.0`（minor），属于 Agent 自行决定产品版本策略。
- 用户明确下个版本是 `v2.0.1`（patch）。`2.0.0` 刚发布，跟进项以修复与验收为主，
  patch 版本线符合实际范围。
- 版本号属于发布决策，必须由用户确定；后续不再自行判断版本级别。

## Scope

- 版本源 5 处由 `2.1.0` 改为 `2.0.1`（Windows 数字版本 `2.0.1.0`）。
- `docs/plans/CURRENT.md`：开发基线、阶段范围标题改为 2.0.1。
- [TASK-20261027-008](TASK-20261027-008-v2.1.0-development-baseline.md) 追加更正说明
  （不改写其原始结论，只标注作废部分）。
- `task --force verify`。

## Out of Scope

- 不打标签、不打包、不推送、不发布、不创建 `docs/releases/v2.0.1.md`。
- 不改业务逻辑、迁移、UI。
- 不修 `CURRENT.md` 的 Known Issues / Next Steps 陈旧段落，不修既有归档记录的失效链接。

## Existing Behavior

版本源为 `2.1.0`，与用户确定的 `v2.0.1` 计划不一致。

## Expected Behavior

版本源为 `2.0.1`／`2.0.1.0`，文档表述为「已发布 2.0.0 / 开发基线 2.0.1」。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/config.yml`、
  `build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`

## Dependencies

- 无。

## Risks

- 版本号回退方向（minor → patch）只影响资源与常量，不涉及数据或协议；
  已安装 2.0.0 的用户升级到 2.0.1 属正常 patch 升级路径。
- 若 `2.1.0` 曾被打包或打标签，回退会造成版本倒退——已确认本轮从未打包、打标签或推送。

---

# Implementation Plan

1. 五个版本源 `2.1.0` → `2.0.1`（`2.1.0.0` → `2.0.1.0`）。
2. `CURRENT.md` 相关表述改为 2.0.1。
3. TASK-20261027-008 追加 Correction 段。
4. `task --force verify` 并回填。

---

# Progress

- 已确认本轮 2.1.0 从未打包、打标签、推送或发布，可安全更正。

---

# Implementation Result

## Changes

- 版本源 5 处：`2.1.0` → `2.0.1`；Windows 数字版本 `2.0.1.0`。
- `docs/plans/CURRENT.md`：开发基线改 `2.0.1`，阶段范围标题改 `v2.0.1 阶段范围`。
- [TASK-20261027-008](TASK-20261027-008-v2.1.0-development-baseline.md)：追加
  `# Correction (2026-10-27)`，声明 `2.1.0` 判断作废、以本任务为准。

## Files Changed

- `internal/version/version.go`
- `frontend/package.json`
- `build/config.yml`
- `build/windows/info.json`
- `build/windows/nsis/wails_tools.nsh`
- `docs/plans/CURRENT.md`
- `docs/tasks/completed/2026-10/TASK-20261027-008-v2.1.0-development-baseline.md`
- 本记录

## Technical Notes

- `2.0.1` 满足 `set-release-version.ps1` 的完整 SemVer 与 Windows 数字版本要求。
- 版本级别由用户决定：后续是否需要 minor／major 先问，不自行提升。

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
第 1 次 task --force verify → exit 201
  format / lint / typecheck / go test 全部通过；
  vitest 132 文件 / 809 项中 1 项失败：
  src/features/providers/providers-view.test.tsx:128
  （await screen.findByRole("button", { name: "切换OpenAI提供商的启用状态" })，
   页面停在该测试桩的「正在加载提供商…」；该轮 Duration 111.75 s）

定向复跑 pnpm vitest run src/features/providers/providers-view.test.tsx → exit 0（10/10，8.47 s）
第 2 次 task --force verify → exit 0
  Test Files 132 passed，Tests 809 passed，Duration 98.42 s
  Windows production 构建通过（bin/omniproxy.exe，含 2.0.1 版本资源）
```

结论：首轮失败是该用例 1000 ms 默认等待窗口在全量并发下的时序抖动，不是本次版本改动引起
（本次只改版本常量、构建资源与文档；该页面与版本号无关）。已用两次验证记录在案，
不据此修改测试或放宽断言。

版本源核对：`internal/version/version.go`、`frontend/package.json`、`build/config.yml`、
`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh` 中 `2.0.1`／`2.0.1.0` 一致，
无 `2.1.0` 残留。

`docs/plans/CURRENT.md` 链接解析：189 条，失效 0（009 归档后复核）。

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
（本记录作为被否决的过程记录随该提交入库；2.0.1 基线未落地，版本源保持 2.0.0）
```

---

# Result

开发基线已更正为 **2.0.1**：版本源 5 处统一为 `2.0.1`／Windows 数字版本 `2.0.1.0`／
NSIS `INFO_PRODUCTVERSION "2.0.1"`；`docs/plans/CURRENT.md` 的开发基线与阶段范围标题改为
2.0.1；原 TASK-20261027-008 追加 `# Correction (2026-10-27)`，声明其 `2.1.0` 结论作废，
不回写原结论。`task --force verify` 第二次运行 exit 0（132 文件／809 项前端测试、
Windows production 构建通过）；首次运行 1 项时序抖动已记录。未打包、未打标签、未推送、未发布。

## Follow-ups

- 版本级别（patch／minor／major）由用户决定，后续不再自行提升。
- `providers-view.test.tsx` 的 `findByRole` 依赖默认 1000 ms 等待窗口，在慢速全量运行中
  可能抖动；是否需要加完成信号屏障另立任务，未在本次改动测试。

---

# Completed

2026-10-27 21:50（Asia/Singapore）

---

# Cancellation (2026-10-27)

用户「我现在还不发版本，撤销这个版本记录啊」。

## Cancellation Reason

- 本任务把基线改成 `2.0.1` 是为承接用户「下个版本是 v2.0.1」的说明，但用户随后明确
  目前不发布新版本，因此该基线也不保留。
- 版本源已由 [TASK-20261027-010](TASK-20261027-010-revert-version-baseline.md) 退回
  `2.0.0`；`2.0.1` 从未打包、打标签、推送或发布，撤销无副作用。
- 保留的记录价值：`2.0.1` 的用户口径与「版本级别由用户决定」的约束。