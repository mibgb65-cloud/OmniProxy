# TASK-20261120-001: 新增「记录」侧边栏页面

## Status

Completed

## Created

2026-11-20 08:25

## Request

用户反馈（原文）：

> 好像记录不了啊，你要不侧边栏新开一个栏，就是记录栏，里面显示有记录到请求的记录

## Goal

1. 在侧边栏新增「记录」入口，集中展示已保存请求正文的请求。
2. 列表显示时间、实际模型、客户端、账号、正文大小与截断状态，并提供可读预览。
3. 明确区分「记录已关闭」「已开启但还没有新记录」「已有历史记录」三种状态。
4. 从记录列表可直接打开对应请求详情查看完整正文。

## Background

2026-11-20 08:20 只读核查生产数据库（`D:\OmniPProxy-data\omniproxy.db`，仅查结构、计数与大小，不读取正文）：

- 迁移 053 已执行，`request_retention.record_content = 1`，用户已开启记录。
- `request_contents` 共 413 条，最新一条与核查时的请求时间一致；最近 2 小时的 5 条请求全部有正文。
- 413 条中 252 条达到 256 KiB 上限并被截断，原始大小最大 1.65 MiB；其余完整。
- 现有入口只在「请求」列表 → 详情弹窗 →「请求内容」，且请求列表混有大量开启记录前的历史请求，用户难以分辨哪些有正文。

结论：记录功能实际生效，问题在于缺少集中入口，且大量正文被截断、详情中不够醒目。

## Scope

- `activity`：新增已记录正文的分页查询（搜索 ID／模型／账号／Agent、总数、分页），列表只返回预览与大小元数据，完整正文仍按需读取。
- 预览提取：从 Chat `messages` 与 Responses `input` 中提取最近的非 system 文本；无法解析时回退为折叠后的原文前缀。
- Wails 绑定 `RecordedRequestsRead`。
- 前端：`/records` 路由、侧边栏「记录」入口、中英文案、列表卡片（时间／模型／客户端／大小／截断／预览）、搜索与分页、空状态与关闭状态说明、跳转请求详情。
- 测试与文档（ACTIVITY、PRD、侧边栏说明、CURRENT）。

## Out of Scope

- 不提高 256 KiB 正文上限，不改变已有截断口径（本任务只把截断状态在记录页明确展示）；该限制随后由 TASK-20261120-002 取消。
- 不做正文全文搜索（预览与元数据搜索，不检索完整正文）。
- 不新增迁移（读取现有 `request_contents`）。
- 不构建新安装包（另立构建任务）。

## Existing Behavior

- 记录开启后正文确实写入 `request_contents`。
- 只能通过「请求」列表逐条点开详情查看，无法一眼看出哪些请求有正文，也无法按「有正文」集中浏览。

## Expected Behavior

- 侧边栏「请求」与「日志」之间新增「记录」，进入 `/records`。
- 页面顶部说明当前记录开关状态；关闭时提示去「设置 → 请求与日志」开启，并说明只影响新请求。
- 已开启但无记录时提示等待新请求；有记录时按保存时间倒序展示，显示预览、模型、客户端、账号、状态、原始／保存大小与「已截断」标记。
- 搜索支持请求 ID、模型、账号与 Agent；分页每页 20 条；点击记录打开对应请求详情。

---

# Analysis

## Related Modules

- `internal/application/activity`：新增记录查询与预览提取。
- `internal/transport/wails`：新增绑定。
- `frontend/src/features/records`（新）、`frontend/src/app/routes/records.tsx`、侧边栏、i18n。
- `docs/product/ACTIVITY.md`、`docs/product/PRD.md`、`docs/plans/CURRENT.md`。

## Dependencies

无新增依赖；标准库 `encoding/json` 与现有 TanStack Query／Radix Dialog。

## Risks

- 预览解析 256 KiB JSON 的 CPU 成本：每页 20 条、最多约 5 MiB 解析，按 10 秒轮询，本地 SQLite 可接受；不使用全文搜索。
- 历史记录在开启前不存在正文，记录页只展示有正文的请求，不制造空记录。
- 路由树由 Vite 插件生成，新增路由后需先构建前端再 typecheck。

---

# Implementation Plan

1. 建任务记录，确认数据库现状（已完成）。
2. `activity`：`RecordedQuery`／`RecordedPage`／`RecordedRequests` 与预览提取 + 单元测试。
3. Wails 绑定与生成的 TypeScript。
4. 前端路由、侧边栏、记录页、i18n、测试。
5. 文档与 CURRENT；`task verify`；提交并回填记录。

---

# Progress

- 2026-11-20 08:25 创建记录；只读核查确认记录已生效、缺少集中入口。
- 2026-11-20 08:30 记录查询、预览提取、Wails 绑定、侧边栏入口、`/records` 页面与测试完成；等待完整质量门。

---

# Implementation Result

## Changes

- `activity.RecordedRequests`：按保存时间倒序分页查询有正文的请求，支持请求 ID／模型／账号／Agent 搜索；返回原始／存储大小、历史截断标记与预览，不返回完整正文。
- `contentPreview`：从 Chat `messages` 与 Responses `input` 提取最近的非 system 文本；跳过 system／developer，无法解析时回退原文前缀，最多 320 字符。
- Wails 绑定 `RecordedRequestsRead`。
- 前端 `/records`：侧边栏「请求」与「日志」之间新增「记录」；页面展示记录开关状态、关闭／等待／历史说明，记录卡片显示时间、结果、模型、客户端、账号、原始／压缩后大小、历史截断标记与预览；支持搜索、每页 20 条分页，点击卡片跳转对应请求详情。
- 中英文案、记录页测试；`ACTIVITY.md`、`PRD.md` 同步。

## Files Changed

- `internal/application/activity/recorded.go`、`content_preview.go`、`recorded.go` 测试与预览测试
- `internal/transport/wails/activity.go`
- `frontend/src/features/records/records-view.tsx`、`record-card.tsx`、`records-view.test.tsx`
- `frontend/src/app/routes/records.tsx`、`frontend/src/app/routeTree.gen.ts`
- `frontend/src/components/common/app-sidebar.tsx`、`frontend/src/services/activity.ts`
- `frontend/src/i18n/locales/records-zh.ts`、`records-en.ts`、`zh.ts`、`en.ts`
- `docs/product/ACTIVITY.md`、`docs/product/PRD.md`

## Technical Notes

- 预览在读取时由正文计算（迁移 054 之后新记录直接读取采集时已存的预览），列表不加载完整正文。
- 本任务最初只展示 256 KiB 截断；用户随后要求取消截断，由 TASK-20261120-002 改为完整压缩存储，本页按新口径显示原始／压缩后大小。

---

# Verification

## Tests

- [x] Unit Tests：记录查询、搜索、分页、预览提取（Chat／Responses／字符串输入／回退）；记录页渲染、空状态与搜索分页；侧边栏导航。
- [x] Integration Tests：Wails 绑定生成；`task verify` 全量 Go 与前端测试。
- [ ] E2E Tests：桌面窗口实机目视检查未执行。

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
- Go 全部包通过
- Windows production 构建通过
- 记录页测试：3 项；侧边栏测试 8 项
```

---

# Documentation

- [x] PRD
- [x] Architecture（OVERVIEW 同步记录页）
- [x] ADR（ADR-028 在 TASK-20261120-002 中补充完整正文修订）
- [x] CURRENT（随提交更新）

---

# Deviations

- 原计划只在记录页展示 256 KiB 截断状态；用户随后要求取消截断，TASK-20261120-002 改为完整压缩存储，本页按新口径显示原始／压缩后大小。

---

# Git

## Commit

```text
99da7c5fadad120a0e1bd6d6fa7570bdb62526dc feat(records): 新增记录页并完整保存请求正文
```

本任务与 TASK-20261120-002 共用同一实现提交；任务记录与 CURRENT 随 `docs(tasks)` 提交。

---

# Result

侧边栏新增「记录」：集中展示已保存正文的请求，支持搜索、分页、预览与跳转详情；关闭／等待／历史状态有明确说明。记录功能经只读核查确认一直生效，问题在于缺少入口与正文截断，后者由 TASK-20261120-002 一并解决。

---

# Completed

2026-11-20 09:02
