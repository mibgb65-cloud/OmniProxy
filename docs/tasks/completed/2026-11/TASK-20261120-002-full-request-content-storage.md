# TASK-20261120-002: 完整保存请求正文（取消 256 KiB 截断）

## Status

Completed

## Created

2026-11-20 08:35

## Request

用户反馈（原文）：

> 不要被 256 KiB 截断

## Goal

1. 请求正文不再按 256 KiB 截断，保存客户端实际发送的完整正文（请求本身已有的 16／32 MiB 边界不变）。
2. 用压缩存储控制数据库体积：正文以 gzip 保存，列表只读元数据与预览。
3. 用独立的正文总预算淘汰最旧正文，避免无限增长；不改变请求明细的保留上限与累计统计。
4. 记录页与详情页正确展示原始大小、压缩后大小与历史截断标记。

## Background

- 迁移 053 的 `request_contents.content` 带有 `CHECK (length(content) BETWEEN 1 AND 262144)`，Core 在采集时按 256 KiB 截断。
- 生产库 413 条正文中 252 条达到上限被截断，原始最大 1.65 MiB，用户看到的内容不完整，因此认为「记录不了」。
- 请求正文在网关侧已有硬上限：协议入口 16 MiB、Codex 透明转发 32 MiB；保存完整正文不需要新的请求体限制。

## Scope

- 迁移 054：重建 `request_contents` 为压缩 BLOB（`data`）+ `format` + `preview`，保留旧 `text` 行并在读取时兼容；新增 `captured_at` 索引。
- Core：移除 256 KiB 截断与 `RequestContentEvent.Truncated`，继续折叠超长 Base64 内联数据、跳过非 UTF-8／压缩透明正文。
- `activity`：gzip BestSpeed 压缩写入、读取解压、采集时生成预览、总预算淘汰（默认 512 MiB 压缩后字节）；保留设置暴露已用／预算。
- 导出：按压缩数据解压后写入 XLSX（Excel 单元格 32767 字符上限仍由 Excel 决定）。
- 记录页：列表使用已存预览；历史截断行继续显示「已截断」。
- 设置页：显示正文占用与预算。
- 测试与文档（ACTIVITY、ADR-028、PRD、CURRENT）。

## Out of Scope

- 不提高单次请求体上限，不保存响应正文。
- 不做正文全文搜索、云同步或正文加密。
- 不改变请求明细保留上限（100–1000000）与累计统计。

## Existing Behavior

正文超过 256 KiB 即被截断，详情只显示前 256 KiB；生产库大量记录不完整。

## Expected Behavior

- 新请求保存完整正文；详情、记录页按需读取完整内容。
- 存储为 gzip 压缩数据，`stored_bytes` 表示压缩后大小，`original_bytes` 表示原始大小。
- 压缩后总占用超过 512 MiB 时自动删除最旧正文，请求明细与累计不受影响。
- 历史已截断记录仍标记「已截断」，读取内容为当时保存的前 256 KiB。

---

# Analysis

## Related Modules

- `migrations/054_compressed_request_contents.sql`
- `internal/core/proxy/request_content.go`、`trace.go`、`tracked_http.go`
- `internal/application/activity/content.go`、`recorded.go`、`export.go`、`types.go`、`retention.go`
- `frontend/src/features/settings/request-retention.tsx`、`records/*`
- 文档与 ADR-028

## Dependencies

- 标准库 `compress/gzip`、`bytes`、`io`；无新增依赖。

## Risks

- 压缩与 JSON 预览解析会增加开启记录时的请求延迟（典型 0.7–1.6 MiB 正文，BestSpeed 预计数十毫秒）；失败组仍不阻断请求。
- 迁移重建表需保留旧数据；迁移在事务中执行，失败会阻止启动并明确报错。
- 总预算淘汰与请求明细淘汰相互独立；正文缺失不影响明细与累计。
- 导出仍受 Excel 单元格上限约束，与存储截断无关，需在界面与文档说明。

---

# Implementation Plan

1. 迁移 054（新表结构 + 旧数据迁移 + 索引）。
2. Core 取消截断；activity 压缩写入／解压读取／预览／预算淘汰；导出读取解压。
3. 前端设置占用显示、记录页大小文案；生成绑定。
4. 单元测试（迁移兼容、大正文完整、压缩大小、预算淘汰、导出）。
5. 文档与 ADR；`task verify`；提交并回填记录。

---

# Progress

- 2026-11-20 08:35 创建记录。
- 2026-11-20 08:40 迁移 054、Core 取消截断、gzip 压缩／预算淘汰／预览、导出解压、设置占用显示与迁移兼容测试完成；等待完整质量门。

---

# Implementation Result

## Changes

- 迁移 054：重建 `request_contents` 为 `data BLOB` + `format`（`gzip`／`text`）+ `preview`，旧明文行原样迁移（`format='text'`，保留历史 `truncated`），新增 `captured_at` 索引。
- Core：移除 256 KiB 截断与 `RequestContentEvent.Truncated`；仍折叠超长 Base64 内联数据、跳过压缩或非 UTF-8 透明正文；完整正文不再截断。
- activity：`packContent` 优先 gzip（BestSpeed），压缩反而更大时按明文保存；采集时生成预览；`RequestContent` 按 format 解压读取；gzip 压缩后总占用超过 512 MiB 时删除最旧正文；`Retention` 暴露 `contentBytes`／`contentBudget`。
- 导出：读取并解压完整正文后写 XLSX（Excel 单元格上限仍为 32767 字符）。
- 记录页：列表使用存储的预览，历史明文行按需解压生成预览；历史截断标记保留。
- 设置页：显示正文压缩后占用／预算。
- 测试：迁移兼容（旧明文→新表）、大正文完整读取、压缩率、预算淘汰、历史行预览、导出完整正文。
- 文档：ACTIVITY、ADR-028（Revision）、PRD、PRIVACY、OVERVIEW。

## Files Changed

- `migrations/054_compressed_request_contents.sql`
- `internal/core/proxy/request_content.go`、`request_content_test.go`
- `internal/application/activity/content.go`、`recorded.go`、`export.go`、`retention.go`、`types.go` 及测试
- `frontend/src/features/settings/request-retention.tsx`、i18n、记录页文案
- `docs/decisions/ADR-028-request-content-capture.md`、`docs/product/ACTIVITY.md`、`PRD.md`、`PRIVACY.md`、`docs/architecture/OVERVIEW.md`

## Technical Notes

- `stored_bytes` 为实际入库大小（多数为压缩后）；压缩无收益的小正文按 `text` 保存，避免 gzip 头开销导致超出请求体上限。
- 预算与请求明细保留上限相互独立：正文先受预算约束，请求明细仍按 100–1000000 条保留；累计统计不受影响。
- 迁移前的 256 KiB 截断行无法恢复完整内容，仍显示历史截断标记。

---

# Verification

## Tests

- [x] Unit Tests：完整正文（超过旧上限）、gzip 压缩与明文回退、512 MiB 预算淘汰、旧明文行迁移与预览、导出解压、迁移 054 兼容。
- [x] Integration Tests：全量迁移在临时数据库执行；`task verify` 全量 Go 与前端测试。
- [ ] E2E Tests：本机生产库迁移与桌面窗口实机检查由用户安装后执行。

## Quality Checks

- [x] Format
- [x] Lint（2 条既有 `react-refresh` 警告，0 error）
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify  EXIT=0
- 前端 164 文件 / 1005 项测试通过
- Go 全部包通过（含迁移 054 与完整正文存储测试）
- Windows production 构建通过
```

---

# Documentation

- [x] ADR-028（Revision：完整正文、gzip、512 MiB 预算）
- [x] ACTIVITY
- [x] PRD
- [x] PRIVACY、Architecture OVERVIEW
- [x] CURRENT（随提交更新）

---

# Deviations

- 压缩无收益的小正文按明文（`text`）保存，而不是强制 gzip；避免 gzip 头开销导致入库体积超过请求体上限。

---

# Git

## Commit

```text
99da7c5fadad120a0e1bd6d6fa7570bdb62526dc feat(records): 新增记录页并完整保存请求正文
```

本任务与 TASK-20261120-001 共用同一实现提交；任务记录与 CURRENT 随 `docs(tasks)` 提交。

---

# Result

请求正文不再按 256 KiB 截断：新请求完整保存并以 gzip 压缩，单条无压缩收益时按明文保存；压缩后总占用超过 512 MiB 时删除最旧正文。迁移 054 原地迁移旧明文行并保留历史截断标记；记录页与详情显示完整正文，设置页显示占用与预算。`task verify` EXIT=0。

---

# Completed

2026-11-20 09:02
