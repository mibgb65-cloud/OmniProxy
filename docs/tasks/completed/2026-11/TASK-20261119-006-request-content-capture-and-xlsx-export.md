# TASK-20261119-006: 记录每次请求正文并支持 XLSX 导出

## Status

Completed

## Created

2026-11-19 16:12

## Request

用户需求（原文）：

> 我现在想给这个软件增加一个功能，需要你增加一个记录每次请求的内容，然后记录使用的模型这样的
>
> 要能够导出为xlsx

经确认的口径：

- 记录范围：只记录**请求正文**（不回看模型输出）。
- 开关：设置页提供开关，**默认关闭**。
- 查看方式：请求详情弹窗新增「内容」区块。
- 导出：请求记录支持导出为 `.xlsx`。

## Goal

1. 在用户主动开启后，网关把每次推理请求的客户端正文按有界大小保存到本机 SQLite。
2. 请求列表继续只展示元数据，详情弹窗按需读取正文；未记录时明确说明。
3. 请求页可按当前筛选条件把请求记录（含实际模型与正文）导出为 XLSX。
4. 默认不记录正文，保持现有隐私承诺；开关、保留上限和清理路径可审计。

## Background

现状（修改前）：

- `request_records` 已保存实际模型 `model`、输入模型 `input_model`、推理档位、账号、Token、耗时、缓存、成本估算等元数据。
- `docs/product/ACTIVITY.md` 与 `PRIVACY.md` 明确承诺**不保存提示词、输出正文、凭据或原始头部**；`ADR-006` 也规定不存请求正文。
- 已有日志 JSONL 导出（`LogsExport` + 原生保存对话框 + 临时文件替换），没有 XLSX 能力，`go.mod` 也没有表格库。
- `go.mod` 未引入 XLSX 库；按 `docs/development/CODING_RULES.md` 的依赖规则，优先用标准库实现最小 XLSX 写入。

`docs/decisions/ADR-006-request-accounting.md` 的隐私边界需要新增 ADR 修订说明；`ACTIVITY.md`、`PRIVACY.md` 必须同步。

## Scope

预计包含：

- 迁移 053：请求保留设置表增加「记录请求正文」开关；新增 `request_contents` 明细表（与请求 ID 一对一）。
- Core `proxy`：在 HTTP、Codex 透明转发、Responses WebSocket（含透明 WebSocket 逐轮）采集客户端正文；有界截断；数据 URL 内联图片折叠；采集失败只记诊断日志，不阻断请求。
- `activity`：保存／读取／清理正文，设置读写，保留上限淘汰时同步清理正文，导出用的流式行迭代。
- `transport/wails`：`RequestContentRead`、`RequestContentSet`、`RequestContentClear`、`RequestsExport` 绑定与原生保存对话框。
- 新包 `internal/infrastructure/xlsx`：标准库实现的最小 XLSX 写入器（流式工作表、无第三方依赖）。
- 设置页开关（默认关闭、隐私说明、清除已保存内容）；请求详情弹窗「内容」区块；请求页「导出 XLSX」按钮。
- 中英文案、单元测试、`task verify`。
- 文档：新 ADR、ACTIVITY、PRIVACY、GATEWAY（如涉及）、CURRENT。

## Out of Scope

- 不记录模型输出／响应正文（用户明确只选请求正文）。
- 不记录原始 HTTP 头部、凭据、接入 Key、User-Agent 原文。
- 不做正文全文搜索、不做云端同步、不做加密数据库（正文随 SQLite 本地存储）。
- 不改动现有 Token／成本／配额统计口径。
- 不引入第三方 XLSX 库。
- 不改变默认行为：开关关闭时正文不落库。

## Existing Behavior

- 正文只在内存中转发，请求结束后不保存。
- 请求详情只展示元数据。
- 日志导出为 JSONL，没有表格导出。

## Expected Behavior

- 设置 → 请求与日志：新增「记录请求正文」开关（默认关闭）与「清除已保存的请求正文」入口。开启后，新请求的客户端正文写入 `request_contents`，每条最多保留 256 KiB，超限标记截断并记录原始字节数；`data:…;base64,` 内联图片折叠为占位文本。
- 请求详情弹窗：有正文时显示格式化（JSON 美化失败则原文）内容、截断提示与复制按钮；未记录 / 历史记录 / 关闭开关采集时显示「未记录请求内容」。
- 请求页：工具栏「导出 XLSX」按钮，按当前搜索与结果筛选导出全部匹配记录（最多 100000 条），包含时间、实际模型、输入模型、档位、服务商、账号、Token、状态、耗时、成本与请求正文列；正文在 Excel 单元格上限内截断并在「内容截断」列标记。
- 关闭开关后不再记录新请求，已保存内容保留到请求明细被保留上限淘汰或用户点击清除。
- 所有采集失败（设置读取失败、写入失败）只进入诊断日志与请求结束日志字段，不改变请求结果。

---

# Analysis

## Related Modules

- `internal/core/proxy`：`handler.go`、`passthrough.go`、`websocket.go`、`passthrough_socket_accounting.go`、`trace.go`、`types.go`、`tracked_http.go`
- `internal/application/activity`：`types.go`、`observer.go`、`queries.go`、`retention.go`、新增正文与导出文件
- `internal/transport/wails`：`app.go`、`activity.go`、新增导出事件绑定
- `migrations/`：新增 053
- `internal/infrastructure/xlsx`：新包
- `frontend/src/features/activity`、`frontend/src/features/settings`、`frontend/src/services`、`frontend/src/i18n/locales`
- 文档：`docs/product/ACTIVITY.md`、`PRIVACY.md`、`docs/decisions/`、`docs/plans/CURRENT.md`

## Dependencies

- Go 标准库 `archive/zip`、`encoding/xml`、`regexp`、`unicode/utf8`：已具备，不新增依赖。
- Wails 原生保存对话框：沿用日志导出模式。
- 无第三方 XLSX 库。

## Risks

- **隐私风险（高）**：正文可能包含提示词、代码、密钥。缓解：默认关闭、显式说明、仅本机、可清除、导出需用户主动选择路径、日志绝不输出正文。
- **存储增长（中）**：开启后数据库可能显著增长。缓解：单条 256 KiB 上限、随请求明细淘汰、可清除；导出单元格按 Excel 上限截断。
- **性能（中）**：采集与正则折叠在请求热路径上。缓解：开关关闭时零成本（只读一次设置）；数据 URL 正则只替换超长 base64；写入与请求事务分离，失败不阻断请求。
- **兼容性（中）**：手写 XLSX 必须被 Excel／LibreOffice 正常打开。缓解：用 bundled openpyxl 实际打开生成文件校验，并加 ZIP/XML 结构测试。
- **统计口径（低）**：正文不进入任何 Token／成本聚合。

## Decisions

- 正文与请求记录分表（`request_contents`），列表查询不读取正文；本任务需要新 ADR 记录隐私边界变化。
- 采集在 Core 完成（协议边界早已持有正文），应用层只负责持久化。
- 采集失败不阻断请求，与「记账失败阻断请求」区分：正文是可选诊断数据。
- XLSX 用标准库写最小 OOXML，避免为单一导出引入大依赖。

---

# Implementation Plan

1. 迁移 053：`request_retention.record_content` 默认 0；`request_contents` 表。
2. `activity`：设置读写、正文写入／读取／清理、保留淘汰同步清理、导出流式迭代。
3. `proxy`：`Options.RecordRequestContent`；`Trace.RecordContent`；四个采集点；有界截断与数据 URL 折叠；诊断日志字段。
4. `gateway` 接线：`s.activity.ContentCaptureEnabled`。
5. `internal/infrastructure/xlsx`：最小 XLSX 写入器 + 测试（含 openpyxl 校验脚本）。
6. `transport/wails`：正文读取／开关／清除、导出绑定与原生保存对话框。
7. 生成 Wails bindings。
8. 前端服务封装、设置开关、详情内容区块、请求页导出按钮、中英文案。
9. Go／前端单元测试。
10. 文档与 ADR；`task verify`；Git 提交并回填任务记录。

---

# Progress

- 2026-11-19 16:12 创建任务记录，完成现有实现与隐私边界分析。
- 2026-11-19 16:20 迁移 053、Core 采集、activity 持久化／导出迭代、Wails 绑定与前端界面完成；`go test` 目标包与受影响前端测试通过。
- 2026-11-19 16:50 生成样例 XLSX 并用 openpyxl 3.1.5 实际打开校验（工作表名、中文、超长单元格截断到 32767）。
- 2026-11-19 17:00 压缩／非 UTF-8 透明转发正文改为不保存；文档与 ADR-028 同步。

---

# Implementation Result

## Changes

实际完成：

- 迁移 053：`request_retention.record_content`（默认 0）与 `request_contents`（请求 ID 一对一、正文、原始／保存字节数、截断标记、采集时间）。
- Core：`Options.RecordRequestContent` 开关读取；`Trace.RecordContent` 单次写入、失败只记录诊断；HTTP 协议入口、Codex 透明 HTTP、Responses WebSocket 每轮、透明 WebSocket 每轮四个采集点；256 KiB 上限与 UTF-8 边界截断；长度 ≥1000 的 Base64 内联数据折叠；压缩／非 UTF-8 正文不保存。
- activity：开关读写、正文写入／读取／清除、保留上限淘汰时同步删除正文、导出用流式行迭代（最多 100000 行）。
- Wails：`RequestContentRead`／`RequestContentSet`／`RequestContentClear`／`RequestsExport`；原生保存对话框、串行化导出、稳定错误码。
- `internal/infrastructure/xlsx`：标准库单工作表 OOXML 写入器，流式行回调、数字／文本单元格、XML 控制字符清理、Excel 32767 字符上限、临时文件替换。
- 前端：请求详情「请求内容」区块（格式化 JSON、复制、大小与截断提示、未记录说明）；请求页「导出 XLSX」与结果提示；设置「记录请求正文」开关与「清除已保存的正文」；中英文案。
- 文档：ADR-028、ACTIVITY、PRIVACY、GATEWAY、FAILOVER、PRD、OVERVIEW、CURRENT。

## Files Changed

主要修改文件：

- `migrations/053_request_contents.sql`
- `internal/core/proxy/request_content.go`、`handler.go`、`passthrough.go`、`websocket.go`、`passthrough_socket_accounting.go`、`trace.go`、`types.go`
- `internal/application/activity/content.go`、`export.go`、`retention.go`、`types.go`
- `internal/infrastructure/xlsx/xlsx.go`、`cell.go`
- `internal/transport/wails/activity.go`、`activity_export.go`、`activity_export_events.go`、`app.go`、`gateway.go`
- `frontend/src/features/activity/request-content.tsx`、`requests-view.tsx`、`request-dialog.tsx`
- `frontend/src/features/settings/request-retention.tsx`
- `frontend/src/services/activity.ts`、`frontend/src/i18n/locales/activity-zh.ts`、`activity-en.ts`
- `docs/decisions/ADR-028-request-content-capture.md` 及相关产品文档

## Technical Notes

- 正文保存的是客户端规整后、路由改写前的字节；透明转发与 WebSocket 保存实际发送的正文／帧。
- 正文写入与用量记账分离：失败只写 `content_unavailable`／`content_setting_unavailable` 与结束日志 `request_content_failed`，不改变请求结果。
- XLSX 不引入第三方依赖；单次导出有 100000 行上限，正文单元格按 Excel 上限截断并由「正文截断」列标记。
- 已有正文随请求明细淘汰；关闭开关只停止新增，手动清除只删除正文。

---

# Verification

## Tests

- [x] Unit Tests：Core 正文采集／截断／折叠／开关；activity 保存／读取／清除／淘汰／导出；xlsx 写入与单元格上限；Wails 导出与正文绑定。
- [x] Integration Tests：迁移 053 全量应用；Wails 原生对话框注入后实际写出 XLSX 并读取工作表；openpyxl 3.1.5 实际打开样例文件。
- [ ] E2E Tests：不适用（桌面端实机目视检查未执行）。

## Quality Checks

- [x] Format
- [x] Lint（2 条既有 `react-refresh` 警告，0 error，与本任务无关）
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify  EXIT=0
- format:check: PASS（gofmt + prettier）
- lint: PASS（go vet、staticcheck、eslint 0 error）
- typecheck: PASS
- test: PASS（Go 全部包；前端 163 文件 / 1002 项测试）
- build: PASS（Windows production，bin/omniproxy.exe）
- XLSX 兼容性: openpyxl 3.1.5 打开样例通过（工作表名「请求记录」、中文、32767 字符截断）
```

---

# Documentation

检查是否更新：

- [x] PRD
- [x] Architecture
- [ ] Tech Stack（无技术栈变化；未新增依赖）
- [ ] Coding Rules
- [ ] Testing
- [ ] Build
- [x] ADR（ADR-028）
- [x] CURRENT

---

# Deviations

- 原计划「非法 UTF-8 以替换字符处理」，实际改为**压缩／非 UTF-8 的透明转发正文不保存**：替换字符会把二进制写成无意义乱码，解压又会引入压缩炸弹风险；该限制已写入 ACTIVITY 与 ADR-028。
- XLSX 采用标准库自实现单工作表写入器（计划内），未引入 excelize 等依赖；已用 openpyxl 实际打开验证兼容性。
- 导出上限 100000 行与单元格 32767 字符截断为计划内边界，前端以「正文截断」列与导出提示说明。

---

# Git

## Commit

```text
3b412e739b3ec6239af0a55f270e745e6163cbaa feat(activity): 可选记录请求正文并导出请求记录 XLSX
```

---

# Result

新增默认关闭的请求正文记录：用户开启后按请求保存客户端正文（每条最多 256 KiB，内联图片折叠、超限截断标记），请求详情可回看与复制；正文随请求明细淘汰并可手动清除。请求页可按当前筛选把模型、Token、成本与正文导出为 XLSX（最多 100000 行，单元格按 Excel 上限截断）。响应正文、凭据与原始头部仍不保存，采集失败不阻断请求。`task verify` EXIT=0。

---

# Completed

2026-11-19 17:12
