# TASK-20261025-007: Codex 额度改走官方 HTTP 接口并保留上次成功结果

## Status

Completed

## Created

2026-10-25 11:14

## Request

原始需求（用户原话）：

> 额度查询失败，请稍后刷新。老是在查询的时候出现这个提示，你看看 v1 版本的时候是怎么处理的，有没有什么可以借鉴的，我不希望看到这个信息每次刷新都出现。

补充：

> 我记得这个 codex 的额度查询不是有官方的接口吗？你参考一下这个项目的处理：`E:\githubresource\cc-switch`

> 你把 task 写出来，然后开始做优化吧

## Goal

用户刷新额度时不再每次都看到「额度查询失败，请稍后刷新。」：

1. Codex 订阅额度优先走官方 HTTP 额度接口（不启动官方子进程），只有必要时才回退现有 stdio 链路；
2. 瞬时失败（网络／超时／5xx／429）在最近一次成功后的一段时间内继续显示上次结果，不弹错误；
3. 确定性失败（未授权、身份变化、凭据不可用）仍然立即透出，不被旧数据掩盖；
4. 额度快照写入失败不再否决已经读到的额度。

## Background

- 现状：`SubscriptionQuota` → `application/subscription.ReadDisplayQuota` → 每次 `infrastructure/codex.Open` 启动 `codex app-server` 子进程（本机实测 5.5–5.9 s，其中初始化 3.5–3.8 s），再 `Account` + `account/rateLimits/read`。
- 生产日志 `%APPDATA%\OmniProxy\v2\logs\omniproxy.log`（2026-10-19 ～ 10-25）共 54 次 `subscription quota query failed`：`quota_timeout` 42（`rate_limits` 10、`queue` 10、`refresh` 9、`open` 8、`account` 4）、`quota_busy` 7、`rate_limits/quota_unavailable` 5、`quota_not_authenticated` 1。其中 `stage=refresh` 只存在于 10-19 那版源码（当时 `Account(true)` 强制续期），当前源码已改为 `Account(false)`。
- 同一数据库 `subscription_quota_snapshots` 中两个 OpenAI 订阅账号都存有 10-25T02:48 的最后成功快照，说明失败时**数据是有的**，但界面选择显示错误。
- v1（`E:\go\new\OmniProxy`）与 cc-switch（`E:\githubresource\cc-switch`）都用官方 HTTP 端点 `GET https://chatgpt.com/backend-api/wham/usage`：v1 配置项 `codexUsageEndpoint` 默认即此地址；cc-switch `services/subscription.rs:984` 同端点，超时 15 s，重置卡并行查询，解析失败只丢该字段。v1 的失败语义是“失败不写 Usage、错误只进日志/健康记录、卡片用静态引导文案”；cc-switch 是 keep-last-good（10 分钟窗口）+ 瞬时/确定性失败分流。

## Scope

本任务包含：

- `internal/infrastructure/codex` 增加官方 HTTP 额度读取（`wham/usage`），含超时、错误分级与响应校验。
- `application/subscription.ReadQuota` 增加 HTTP 快路径：本地隔离授权材料 → HTTPS 读取；`ErrRefreshRequired`／传输失败／协议失败回退既有 stdio 链路。
- 快照写入失败降级为告警，仍返回已读到的额度。
- 前端额度展示 keep-last-good：瞬时失败在 10 分钟窗口内保留上次成功结果并标注“已保留上次获取的信息”，确定性失败立即透出。
- 相关测试与文档。

## Out of Scope

- 不改账号 `authState` 判定规则、不改额度驱动的调度／自动激活语义。
- 不新增数据库迁移、不加依赖、不改 v1 仓库。
- 不实现定时轮询、额度历史、低额度提醒。
- 不替换重置卡（`ConsumeReset`）路径。

## Existing Behavior

- 每次用户刷新（或额度页挂载超过 `staleTime`）都为每个订阅账号启动一次官方子进程；任一环节失败即返回 `subscription: quota_*`，前端 `query.isError` → 卡片渲染 `role="alert"` 红字，即使保留着上次成功数字。
- `fromCache` 仅在冷启动读盘或旧快照回填时为真，实时查询失败时永不置位，因此“已保留上次获取的信息”不会出现。
- `QuotaCache.Save` 失败会让一次成功的额度读取整体变成 `quota_cache_unavailable`。

## Expected Behavior

- 额度读取在正常网络下由一次 HTTPS 请求完成，本机不再为每次刷新启动子进程；仅在需要续期或 HTTP 路径不可用时才走 stdio。
- 瞬时失败且上次成功在 10 分钟内：卡片继续显示上次百分比与重置时间，并标注“已保留上次获取的信息”，不出现红色错误；超过窗口或确定性失败才显示错误。
- 快照写入失败时额度照常返回，只在日志留一条稳定码。

---

# Analysis

## Related Modules

- `internal/infrastructure/codex`（新增 HTTP 额度读取，复用 `RateLimits` 模型与校验函数）
- `internal/application/subscription`（`quota.go` 快路径与快照降级、`service.go` 注入点）
- `internal/transport/wails/subscription.go`（装配与告警回调）
- `frontend/src/features/accounts`（保留策略与卡片展示）
- `frontend/src/features/quota`、`frontend/src/features/dashboard`（共用同一策略）
- 文档：`docs/product/SUBSCRIPTION.md`、`docs/product/QUOTA.md`、`docs/architecture/OVERVIEW.md`、`docs/plans/CURRENT.md`

## Dependencies

无新增依赖（标准库 `net/http` + 现有 i18n）。

## Risks

- `wham/usage` 是官方客户端的反代订阅端点，非公开文档化 API：形状或可用性可能变化。缓解：任何解析／协议失败都回退现有 stdio 链路，不改变权限判断。
- 快路径不打开官方组件，因此失去 `Account(false)` 的线上身份核验；缓解：`UseFreshToken` 已在本地核对受保护材料的账号 ID 与邮箱，且账号版本在写快照时由 SQL 条件再次校验。
- keep-last-good 可能短暂掩盖真实故障；缓解：只对瞬时码生效、窗口 10 分钟、且确定性码会清空留存并立即透出。

---

# Implementation Plan

1. `infrastructure/codex/usage.go`：`UsageClient.Read`（`GET wham/usage`，15 s 超时，`Bearer` + `ChatGPT-Account-Id`），错误分级 `ErrUsageUnauthorized` / `ErrUsageTransient` / `ErrProtocol`，返回既有 `RateLimits`。
2. `application/subscription/quota.go`：抽出 `publishQuota`（记忆 + 快照 + 显示合并）；新增 `readQuotaHTTP` 快路径；`QuotaCache.Save` 失败改为告警。
3. `application/subscription/service.go`：新增 `UsageSource` 注入点与 `SetWarn`。
4. `transport/wails/subscription.go`：装配 `codex.NewUsageClient(nil)` 与 WARN 回调。
5. 前端 `features/accounts/quota-retention.ts`：纯函数保留策略 + 瞬时/确定性码分类；接入 `use-subscription-quota`、`use-quota-overview`、订阅额度卡片与配额卡片。
6. 测试：Go 侧 HTTP 解析／错误分级／回退／快照降级；前端保留策略与卡片行为。
7. `task verify`，回填本记录并更新文档。

---

# Progress

- 2026-10-25 11:14：完成根因分析（对比 v1、cc-switch、生产日志与数据库快照），建立本记录。
- 2026-10-25 11:20：HTTP 额度读取（`infrastructure/codex/usage.go`）与 `readQuota` 快路径／快照降级完成，定向 Go 测试通过。
- 2026-10-25 11:28：前端 keep-last-good（`quota-retention.ts`）接入账号卡片、配额页与首页，定向测试与 typecheck 通过。
- 2026-10-25 11:36：`task verify` exit 0（129 文件／780 项前端测试）。

---

# Implementation Result

## Changes

实际完成：

- 新增官方 HTTP 额度读取：`GET https://chatgpt.com/backend-api/wham/usage`，`Bearer` + `ChatGPT-Account-Id` + `User-Agent: codex-cli`，15 秒上限，响应上限 256 KiB；把 `rate_limit.primary_window/secondary_window` 的 `used_percent`／`limit_window_seconds`／`reset_at` 换算成既有的 `RateLimits` 快照（秒 → 分钟，取整并夹到 0–100），缺少使用比例或窗口时长的窗口不猜、直接跳过。
- 错误分级：401／403 → `ErrUsageUnauthorized`（确定性），429／其它非 2xx／传输失败／读体失败 → `ErrUsageTransient`，JSON 或形状不可用、没有任何可用窗口 → `ErrProtocol`。
- `ReadQuota` 增加 HTTP 快路径：先在同一账号门内用 `UseFreshToken` 读取本地隔离授权材料（已核对账号 ID 与邮箱），再请求额度接口；成功即返回，不再启动官方子进程。`ErrRefreshRequired`、`ErrUsageTransient`、`ErrProtocol`、材料不可用一律回退既有 stdio 链路（含续期）；`ErrUsageUnauthorized` 直接返回 `quota_not_authenticated`（阶段 `usage`），不回退、不改变授权状态。
- 抽出 `publishQuota`（记忆观察 + 保存显示快照 + 显示合并），HTTP 与 stdio 两条路径共用；`QuotaCache.Save` 失败改为 `SetWarn` 告警回调，仍返回本次已读到的额度，不再把一次成功读取变成 `quota_cache_unavailable`。
- Wails 装配：`SetUsageSource(codex.NewUsageClient(nil))` 与 `SetWarn(...)` → 结构化 WARN 日志（只含稳定码）。
- 前端 keep-last-good：新增 `quota-retention.ts`（10 分钟窗口；瞬时码为 `quota_transient`／`quota_unavailable`／`quota_timeout`／`quota_busy`／`quota_refresh_failed`／`operation_timeout`，其余为确定性）。账号卡片、配额页、首页资源卡片统一按此策略：窗口内继续显示上次结果并标注“已保留上次获取的信息”，确定性失败立即透出且不保留旧值。

## Files Changed

新增：

- `internal/infrastructure/codex/usage.go`、`usage_test.go`
- `internal/application/subscription/quota_http.go`、`quota_http_test.go`
- `frontend/src/features/accounts/quota-retention.ts`、`quota-retention.test.ts`

修改：

- `internal/application/subscription/quota.go`、`service.go`
- `internal/transport/wails/subscription.go`
- `frontend/src/features/accounts/use-subscription-quota.ts`、`subscription-quota.tsx`、`subscription-quota.test.tsx`、`subscription-quota-refresh.test.tsx`
- `frontend/src/features/quota/use-quota-overview.ts`、`quota-resource.tsx`、`quota-view.test.tsx`
- `frontend/src/features/dashboard/use-dashboard-resources.ts`
- `docs/product/SUBSCRIPTION.md`、`docs/product/QUOTA.md`、`docs/architecture/OVERVIEW.md`、`docs/plans/CURRENT.md`

## Technical Notes

- 快路径不再有 `Account(false)` 的线上身份核验；替代证据是 `UseFreshToken` 在受保护存储内核对账号 ID 与邮箱，加上写快照时 SQL 的 `updated_at/auth_method/auth_email/auth_account_id` 条件，账号改动后旧结果无法写回。
- `quota_unavailable` 归为“瞬时”：生产日志中用户看到的那句提示正是该码对应的 `rate_limits/request_failed`；确定性失败已有独立码，窗口到期后错误仍照实显示。
- 重置卡统计仍只来自 stdio 协议（HTTP 接口只用于窗口），本次未改重置卡链路。

---

# Verification

## Tests

- [x] Unit Tests（Go：`internal/infrastructure/codex`、`internal/application/subscription` 定向与整包）
- [x] Integration Tests（`internal/transport/wails`、`internal/application/quotaactivation` 整包回归）
- [ ] E2E Tests（不适用：仓库尚无 Playwright 流程）
- [x] Frontend（`vitest`：129 文件／780 项）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
go test ./internal/...            PASS（全部包）
go vet ./internal/...             PASS
gofmt -l internal/                无输出
task format:check                 exit 0
task lint                         exit 0
task typecheck                    exit 0
task test                         前端 129 文件 / 780 项通过
task build                        exit 0（bin/omniproxy.exe 22,901,760 bytes）
task verify                       exit 0
```

未执行：真实账号联调（需要用真实订阅账号验证官方 HTTP 接口在当前账号上的返回形状）。本次验证未打开 `codex` CLI、未修改生产数据库与安装。

---

# Documentation

- [x] PRD — 无需修改（额度能力描述未变）
- [x] Architecture — `OVERVIEW.md` 订阅额度查询链路改为“HTTP 优先、stdio 回退”
- [x] Tech Stack — 无新增依赖，无需修改
- [x] Coding Rules — 无需修改
- [x] Testing — 无需修改
- [x] Build — 无需修改
- [x] ADR — 不新增：沿用 ADR-019 的“实时观测与显示留存分离”，本次补齐接口与展示策略
- [x] CURRENT — 已移入 Recently Completed

---

# Deviations

1. 原计划前端瞬时码集合不含 `quota_unavailable`，实现时改为包含。
   原因：生产日志显示用户反复看到的那句提示正是 `rate_limits/quota_unavailable`；确定性失败已由 `quota_not_authenticated`／`identity_changed`／`invalid_account` 单独表示，因此把它归入瞬时，并保留 10 分钟窗口到期后照实透出。
2. 两个既有前端测试（`subscription-quota-refresh.test.tsx`、`quota-view.test.tsx`）把失败样例从 `quota_unavailable` 换成 `quota_not_authenticated`。
   原因：它们验证的是“旧快照保留 + 错误透出”，而瞬时码现在按设计不再透出错误；断言意图不变，另有新增用例覆盖“瞬时失败不报错”。
3. 计划中的“失败退避”未实现。
   原因：HTTP 快路径已把单次查询从约 5.5 秒降到一次 HTTPS，既有后台观察已有 5 分钟去重；UI 侧冷却会与用户主动刷新冲突，留待另立任务。

---

# Git

## Commit

```text
5ce4bfcfe6398d113ec18c4761f37b3ab4934b16 fix(subscription): 额度改走官方 HTTP 接口并保留上次成功结果
```

仅本地提交，未推送。本记录的归档提交见紧接着的 `docs(task)` commit。

---

# Result

Codex 订阅额度查询从“每次启动官方子进程”改为“官方 HTTP 额度接口优先、必要时回退 stdio”，单次查询不再依赖 3.5–3.8 秒的子进程握手；显示快照写入失败不再否决已读到的额度；前端对瞬时失败改为 10 分钟 keep-last-good。用户刷新时不会再每次看到“额度查询失败，请稍后刷新。”，确定性失败（未授权、身份变化）仍立即透出。`task verify` exit 0。

---

# Completed

2026-10-25 11:45
