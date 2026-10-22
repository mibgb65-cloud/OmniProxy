# TASK-20261023-009: 配额卡片显示本机 Token 用量、原生悬浮提示替换与请求 Agent 归属

## Status

Completed

## Created

2026-10-23 09:10

## Request

用户提供三张截图并指出三项改进：

1. 图一（配额页 River 账号卡片）：显示不了剩余额度，只看得到「暂不支持」和「此账号的额度或余额查询尚未接入」，要求直接显示已经用了的 Token 用量。
2. 图二（账号详情「Token 用量」三块输入／输出／缓存）：鼠标悬浮出现的是系统原生提示框（白底黑框），效果很丑，要求换成本产品优化后的样式。
3. 图三（请求页列表）：需要显示 Token 是被哪个 Agent 用的，例如 Codex、DeepSeek Harness 等当前可以接入的 Agent。

## Goal

- 未接入额度／余额查询的账号卡片直接展示本机留存的已用 Token（输入／输出／缓存／请求数），不再只给一句「尚未接入」。
- 账号详情的 Token 用量数值悬浮提示改用产品统一 Tooltip 组件，去掉原生 `title` 提示框。
- 请求列表与请求详情展示发起该请求的本地 Agent（受管客户端），未知来源明确显示「—」。

## Background

- 配额页对不支持的提供商（River、Atria、普通 OpenAI API Key、自定义提供商）返回 `kind: "unsupported"`，卡片只显示固定提示，没有任何用量信息。
- 本机留存用量已有聚合实现 `activity.AccountTokenUsage`，但目前只有 River 专用入口 `RiverAccountTokenUsage`（账号页使用），身份索引固定为 `builtin_river + api_key`。
- 账号详情 `AccountTokenSummary` 用原生 `title` 属性展示完整数值，WebView2 下显示系统样式提示框。
- 请求记录 `request_records` 没有任何客户端字段；网关侧每个受管 Agent 都有独立接入 Key（`agent_connections`），`Authorize` 目前只返回布尔值，丢弃了「这把 Key 属于哪个客户端」的信息。

## Scope

- `internal/application/activity`：把 River 专用用量入口泛化为按账号读取；`RequestItem` 增加 `client`。
- `internal/application/account`：抽出账号的地区／计费作用域，供网关记账与用量查询共用。
- `internal/application/agentconfig`：接入 Key 识别返回受管客户端标识。
- `internal/core/proxy`：`Authorize` 返回客户端标识并写入 `RequestEvent.Client`（HTTP 与 WebSocket）。
- `migrations/040_request_client.sql`：请求记录新增 `client` 列。
- 前端：配额卡片展示本机用量、Token 用量 Tooltip、请求列表／详情 Agent 列。
- 中英文案、回归测试、产品文档。

## Out of Scope

- 不新增官方额度／余额接口，不把本机 Token 用量当成剩余额度或汇率换算。
- 不新增「按 Agent」用量汇总分组（每日桶键仍为 day/subject/model）。
- 不保存客户端 User-Agent、请求头或凭据到数据库。
- 不改账号页 River 用量卡片的数据来源与保留策略。

## Existing Behavior

- 配额页不支持查询的账号：状态「暂不支持」+ 提示「此账号的额度或余额查询尚未接入」+「尚无成功查询结果」。
- 账号详情 Token 用量：原生 `title` 提示框显示完整数字。
- 请求页：时间／模型／账号／结果／输入 Token／输出 Token／耗时，无 Agent 信息。

## Expected Behavior

- 不支持额度查询的账号卡片显示「本机 Token 用量」：输入、输出、缓存（含 `≥` 未知缓存标记）、请求数；无记录时显示已知零，读取失败显示失败与重试。
- Token 用量数值悬浮显示产品统一圆角 Tooltip，含完整千分位数字。
- 请求列表与详情显示 Agent 名称（Codex、Claude Code、DeepSeek Harness 等）；由其他客户端或直接调用产生、无法归属的请求显示「—」。

---

# Analysis

## Related Modules

- `internal/core/proxy`（鉴权、请求事件）、`internal/application/agentconfig`（接入 Key）、`internal/application/activity`（记账与查询）、`internal/transport/wails`（绑定）
- 前端 `features/quota`、`features/accounts`、`features/activity`、`features/gateway/agent-clients.ts`、`components/ui/tooltip`

## Dependencies

无新增依赖。

## Risks

- 用量身份必须与记账快照完全一致（提供商类型 + 账号 + 可选取地区／计费 + 订阅邮箱／工作区），否则会混入其他账号或漏记。
- 客户端标识写入数据库可能被误认为保存原始头部：只保存受管客户端白名单 ID，不保存 User-Agent。
- `Authorize` 签名变更影响网关核心与全部调用点，需要同步修改测试闭包，不能放宽鉴权语义。
- 历史请求没有客户端字段，必须明确显示未知而不是猜测。

---

# Implementation Plan

1. `account.Item` 增加地区／计费作用域方法，网关 `snapshotTarget` 与会用量查询共用。
2. `activity`：`RiverTokenUsage` 泛化为 `AccountUsage`（按账号 + 提供商类型构造身份）；`accountTokenUsage` 复用同一实现；`RequestEvent.Client` 校验并落库。
3. `agentconfig`：新增 `Client(ctx, key)` 返回受管客户端标识，`Authorize` 在其上返回布尔值。
4. `core/proxy`：`Options.Authorize` 改为返回 `(client, ok)`，HTTP 与 WebSocket 都把客户端写入 `RequestEvent`。
5. 迁移 040 增加 `request_records.client`，查询层返回 `client`，搜索包含该列。
6. 前端：请求列表／详情显示 Agent；配额卡片显示本机用量；Token 用量改统一 Tooltip。
7. 补 Go／前端回归测试、更新产品文档、`task verify`。

---

# Progress

- 2026-10-23 09:10 建立任务记录，确认配额 `unsupported` 分支、账号详情原生 `title` 与请求记录缺少客户端字段的现状。
- 2026-10-23 09:40 后端完成：`account.Item.ProviderScope` 统一地区／计费取值；`activity.AccountUsage` 泛化 River 专用用量入口；`Authorize` 返回受管客户端标识并写入 `RequestEvent.Client`；迁移 040 增加 `request_records.client_id`。
- 2026-10-23 10:20 前端完成：请求列表／详情 Agent 列、配额卡片本机用量、Token 用量统一 Tooltip；绑定重新生成（`-ts -i`）。
- 2026-10-23 10:50 修正两处回归：迁移 040 列名避免与旧库探测的 `client` 列冲突（改为 `client_id`）；`accounting_test` 显式应用迁移 040。
- 2026-10-23 11:10 `task verify` exit 0（123 文件 / 731 项前端测试、Go 测试、Windows production 构建）。

---

# Implementation Result

## Changes

已实现：

- `internal/application/account/provider_scope.go`（新增）：`Item.ProviderScope()` 返回 StepFun／MiniMax／MiMo／Kimi 选项决定的地区与计费方式，网关记账（`snapshotTarget`）与用量查询共用同一取值，避免同一账号被拆到不同主体。
- `internal/application/activity/account_token_usage.go`：`RiverTokenUsage` 泛化为 `AccountUsage`——按提供商配置读取 `providers.kind`，再用与记账一致的身份索引（提供商类型 + 账号 + 地区／计费 + 订阅邮箱／工作区）统计本机留存用量；`accountTokenUsage`（Codex 订阅）复用同一实现；保留已知零、未知明细与进行中／未完整计数。
- `internal/transport/wails/activity.go`：`RiverAccountTokenUsage` 改为 `AccountTokenUsage(id)`，仍只读账号元数据与本机记录。
- `internal/application/agentconfig/keys.go`：新增 `Client(ctx, key) (string, bool)`，遍历全部接入引用（不提前退出）返回受管客户端标识；`Authorize` 在其上返回布尔值，鉴权语义不变。
- `internal/core/proxy`：`Options.Authorize` 改为返回 `(client, ok)`；HTTP 与 WebSocket 每帧鉴权后把客户端写入 `RequestEvent.Client`，随首个事件落库。
- `migrations/040_request_client.sql`：`request_records.client_id`（`''` 或 1–32 字符），历史记录保持未知。
- `internal/application/activity/queries.go`、`types.go`：`RequestItem.client` 返回并纳入关键字搜索。
- 前端：请求列表新增「Agent」列、请求详情新增「Agent」行（`agentTitleKey`，未知显示「—」）；搜索占位符补充 Agent；配额页 `unsupported` 账号改为「尚未接入 + 本机 Token 用量（输入／输出／缓存／请求数）+ 非官方额度说明」；`AccountTokenSummary` 改用统一 `Tooltip` 并以 `aria-label` 提供完整数字。
- `features/accounts/river-token-usage.tsx` 泛化为 `account-usage-panel.tsx`（`AccountUsagePanel`），账号详情与配额页共用；`river.usage.*` 的加载／失败／重试／不完整文案迁到 `activity.accountUsage*`，River 页面说明保留。
- 中英文案：`activity.{agent,accountUsage*,search}`、`quota.{unsupportedHint,unsupportedUsageNote}`。

## Files Changed

- `migrations/040_request_client.sql`（新增）
- `internal/application/account/provider_scope.go`（新增）
- `internal/application/activity/{account_token_usage.go,observer.go,queries.go,types.go,account_usage_test.go,request_client_test.go}`
- `internal/application/agentconfig/{keys.go,keys_test.go}`
- `internal/application/gateway/{trace.go,accounting_test.go}`
- `internal/core/proxy/{types.go,handler.go,websocket.go,trace.go,client_identity_test.go}` 及鉴权闭包相关测试
- `internal/transport/wails/{activity.go,gateway.go,account_token_usage_test.go}`
- `frontend/src/features/accounts/{account-usage-panel.tsx,account-token-summary.tsx,account-details-card.tsx,account-details-list.tsx}` + 测试
- `frontend/src/features/activity/{request-table.tsx,request-dialog.tsx,request-agent.test.tsx,requests-pagination.test.tsx}`
- `frontend/src/features/quota/{quota-resource.tsx,quota-test-helpers.tsx,quota-view.test.tsx,quota-cache.test.tsx}`
- `frontend/src/features/gateway/agent-clients.ts`、`frontend/src/services/activity.ts`、`frontend/src/i18n/locales/*`
- `frontend/bindings/...`（重新生成，仅新增 `AccountTokenUsage` 与 `RequestItem.client`）
- `docs/product/{ACTIVITY,QUOTA,GATEWAY}.md`、`docs/plans/CURRENT.md`

## Technical Notes

- 列名用 `client_id` 而非 `client`：旧库探测把 `request_records` 中的 `client` 视为 Beta.1 历史统计列，直接命名为 `client` 会让已升级数据库被误判为未知结构。
- Agent 归属来自本次请求实际通过的独立接入 Key，不解析 User-Agent、不保存 Key 或原始头部；WebSocket 每帧重新鉴定，Key 停用后旧连接立即失效。
- 本机 Token 用量不是官方剩余额度：配额卡片同时显示固定说明，避免被读成余额。

---

# Verification

## Tests

- [x] Unit Tests
      - Go：`activity`（账号隔离、未知缓存、部分用量、空历史已知零、无法构造身份时拒绝）、`agentconfig`（每把 Key 只归属自己的客户端并拒绝未知 Key）、`core/proxy`（鉴权通过的客户端写入首个事件、未通过鉴权不产生记录）
      - 前端：`request-agent.test.tsx`（Agent 列显示 Codex／DeepSeek Harness Web，未知与自定义客户端显示「—」）、`account-usage-panel.test.tsx`、`quota-view.test.tsx`（未接入额度的账号显示本机 Token）、`subscription-quota-usd.test.tsx`、`account-token-summary.test.tsx`
- [x] Integration Tests（`task test`：Go 全部包 + 前端 123 文件 / 731 项通过）
- [ ] E2E Tests（N/A，无 E2E 框架）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify
PASS（format:check → lint → typecheck → test → build，exit 0；前端 123 文件 / 731 项测试通过，Go 全部包通过，Windows production 构建产出 bin/omniproxy.exe）
```

绑定：`wails3 generate bindings -ts -i` 仅新增 `AccountTokenUsage` 与 `RequestItem.client`，无其他差异。

视觉核对（真实构建产物 CSS + 组件 DOM，非原生窗口）：`tmp/ui-preview/agent-usage.png` 显示请求列表的 Agent 列（Codex CLI / Desktop、DeepSeek Harness Web、Claude Code Desktop、未知为「—」）与配额卡片的本机用量（输入 300.77M / 输出 984.71K / 缓存 ≥ 103.57M + 非官方额度说明）。原生 Wails 窗口下的 Tooltip 悬浮与真实客户端请求归属未在本任务内验收。

---

# Documentation

- [ ] PRD
- [ ] Architecture
- [ ] Tech Stack
- [ ] Coding Rules
- [ ] Testing
- [ ] Build
- [ ] ADR
- [x] CURRENT

更新：`docs/product/ACTIVITY.md`（请求 Agent 归属、迁移 040、账号本机用量）、`docs/product/QUOTA.md`（未接入额度账号显示本机 Token）、`docs/product/GATEWAY.md`（接入 Key 同时决定 Agent 归属）、`docs/plans/CURRENT.md`（本任务条目）。

---

# Deviations

- 原计划请求记录列名 `client`；实际使用 `client_id`。原因：旧库探测把同名 `client` 判为 Beta.1 历史统计列，会导致已升级数据库被拒绝打开。
- 原计划只给 River 账号展示本机用量；实际泛化为任意账号（含 Atria、普通 OpenAI API Key、自定义提供商），避免同类账号继续只显示一句「尚未接入」。
- 原计划配额卡片顶部保留原提示；实际改写为「改为显示本机记录的已用 Token」，并新增「不表示官方剩余额度或账单」一行。

---

# Git

## Commit

`211a98f` — feat(usage): 配额显示本机 Token 用量并为请求标注 Agent（57 个文件，含迁移 040、Go 核心／应用／传输、绑定、前端与产品文档）。本记录归档与 docs/plans/CURRENT.md 状态更新随后续 docs(tasks) 提交记录。

---

# Result

三项反馈均已实现：配额页对没有官方额度／余额接口的账号直接显示本机已用 Token；账号详情的 Token 用量悬浮改为产品统一 Tooltip；请求列表与详情按独立接入 Key 显示发起请求的 Agent（Codex、Claude Code／Desktop、DeepSeek Harness 等，未知显示「—」）。完整质量门通过；原生窗口 Tooltip 与真实客户端归属待用户验收。

---

# Completed

2026-10-23 归档：实现提交 `211a98f`，完整质量门 exit 0。原生窗口 Tooltip 悬浮与真实客户端 Agent 归属仍待用户验收。

