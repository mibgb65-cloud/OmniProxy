# TASK-20261113-013: 请求明细把进行中的记录固定置顶并加分隔

## Status

Completed

## Created

2026-11-13 11:48（Asia/Singapore）

## Request

> 可不可以处理一下这个，就是默认在进行中的这个请求记录，就一直置顶在最上面，下面的是
> 已经完成的记录，我觉得这样更加直观，可以在中间做个分隔

（截图里两条“进行中 · 未报告”的记录夹在已完成记录之间。）

## Goal

- 请求页里进行中（`pending`）的记录始终排在最前，已结束的记录排在其后。
- 两组之间有一条可见分隔，让“还没结束”和“已结束”一眼可分。
- 分页、筛选、5 秒自动刷新与详情打开行为不变。

## Background

- 列表查询在 `internal/application/activity/queries.go`，固定按
  `r.started_at DESC, r.id DESC` 排序，因此一条进行中的长请求会被随后完成的请求挤到中间。
- 页面按 50 条分页并每 5 秒刷新；请求明细表格是 `frontend/src/features/activity/request-table.tsx`。

## Scope

- `internal/application/activity/queries.go`：排序把 `pending` 提到最前。
- `frontend/src/features/activity/request-table.tsx`：进行中一组在上、已结束一组在下，
  两组之间渲染分隔行。
- i18n：新增分隔行文案（中／英）。
- 前端与 Go 测试。

## Out of Scope

- 不改仪表盘「最近请求」的排序（那是最新时间线，不是工作台）。
- 不改筛选、分页大小、保留上限与刷新间隔。

## Existing Behavior

进行中的记录按开始时间混在已完成记录之间。

## Expected Behavior

进行中的记录固定在最上面，其后是“已结束”分隔行与已完成记录。

---

# Analysis

## Related Modules

- `internal/application/activity/queries.go`
- `frontend/src/features/activity/request-table.tsx`、`requests-view.tsx`
- `frontend/src/i18n/locales/activity-{zh,en}.ts`

## Dependencies

无。

## Risks

- 排序变化会让分页边界随“某条请求刚好结束”而移动：页面本来就每 5 秒整体刷新，
  行序会变，这是实时列表的固有行为；不新增去重或游标逻辑。
- 分组在前端也会做一次（保持各组内原顺序），即使后端顺序被改动，视觉分组也稳定。

---

# Implementation Plan

1. 创建本记录。
2. 后端排序加 `CASE WHEN r.state='pending' THEN 0 ELSE 1 END`。
3. 表格按 `pending` 分组渲染，并在两组之间加分隔行。
4. 补测试（分组顺序、分隔行只在两组都存在时出现）。
5. `task verify`，提交并构建本地 Beta。

---

# Progress

- 2026-11-13 11:48：创建记录。

---

# Implementation Result

## Changes

- `internal/application/activity/queries.go`：请求列表排序改为
  `CASE WHEN r.state='pending' THEN 0 ELSE 1 END, r.started_at DESC, r.id DESC`；
  进行中的记录跨分页也排在最前。
- `frontend/src/features/activity/request-table.tsx`：表格按 `state === "pending"` 分组渲染
  （组内保持接口顺序），两组之间插入一条 `已结束` 分隔行；只有一组时不显示分隔行。
- i18n：新增 `activity.finishedGroup`（已结束／Finished）。
- 测试：`request-table-groups.test.tsx`（置顶、分隔行位置、单组不显示分隔行）、
  `queries_order_test.go`（pending 置顶、已结束组仍按时间倒序、状态筛选不受影响）。

## Files Changed

- `internal/application/activity/queries.go`、`queries_order_test.go`（新增）
- `frontend/src/features/activity/request-table.tsx`、`request-table-groups.test.tsx`（新增）
- `frontend/src/i18n/locales/activity-{zh,en}.ts`

## Technical Notes

- 前端也做一次分组：即使后端顺序变化（例如换成游标分页），视觉分组仍然成立。
- 只影响请求页；仪表盘「最近请求」保持纯时间线。

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

```text
go test ./internal/application/activity/ -count=1          → ok
vitest src/features/activity                               → 7 文件 / 26 项通过
task --force verify → VERIFY_EXIT=0（149 文件 / 904 项前端测试，全部 Go 包与 Windows production 构建）
```

---

# Documentation

- [x] CURRENT

---

# Git

## Commit

见归档提交。未打标签、未推送。

---

# Result

请求页里进行中的记录固定在最上面，其后一条「已结束」分隔行，再是已完成记录；
分页、筛选、5 秒刷新与详情行为不变。

---

# Completed

2026-11-13 12:00
