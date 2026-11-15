# TASK-20261115-011: HowToFix 内置服务商接入

## Status

In Progress

## Created

2026-11-15 20:40（Asia/Singapore）

## Request

用户给出 <https://howtofix.id/docs>，要求评估并给出接入方案：

> 你看看怎么接入这个平台到我的OmniProxy

本轮只做只读调研与方案记录，不修改产品代码、配置、迁移或依赖。

## Goal

判定 HowToFix 能否作为 OmniProxy 的可调用通道接入，并给出可执行的实现计划与明确边界：
用户添加一个 `sk-` API Key 后，能在本机网关路由里选到该账号的模型，Claude Code／Codex／
DSH 等受管客户端按既有目录机制自动可用；不臆造上游能力，不宣称未经真实 Key 验证的协议。

## Background

### 官方事实（只读抓取，2026-11-15）

文档页 `/docs` 是 SPA 空壳，正文由 `GET https://howtofix.id/api/handbook/docs` 返回
（HTML，约 22.5 KB）；`/api/handbook/overview`、`/api/handbook/cli` 同理，支持 `?lang=id`。
页面 JSON-LD 的 `sameAs` 指向 `github.com/QuantumNous/new-api`，`/api/status` 返回
`version = v1.0.0-rc.38`、`quota_per_unit = 500000`、`server_address = https://api.howtofix.id`，
错误体与 `/api/pricing` 结构均为 new-api（One API 系）形状。

| 项 | 值 | 证据 |
| --- | --- | --- |
| 服务商 | HowToFix，运营方 NetroIndonesia（印尼），支持 support@howtofix.id | handbook §联系 |
| OpenAI Base URL | `https://api.howtofix.id/v1`（**已含 `/v1`**） | handbook §1／§4 |
| 认证 | `Authorization: Bearer sk-...`；Anthropic 形状路由同时接受 `x-api-key` + `anthropic-version: 2023-06-01` | handbook §4；`POST /v1/messages` 伪造 Key 返回 `Invalid token` 而非缺认证 |
| 目录 | `GET /v1/models`、`GET /v1/models/:model` | handbook §5 |
| 协议端点（实测路由存在，伪造 Key 一律 401） | `/v1/chat/completions`、`/v1/completions`、`/v1/messages`、`/v1/messages/count_tokens`、`/v1/responses`（含 `/v1/responses/compact`）、`/v1/embeddings`、`/v1/rerank`、`/v1/images/*`、`/v1/audio/*`、`/v1/moderations`、`/v1/videos`、`/v1/alpha/search`、`/v1beta/models` | 逐路径探测 |
| 文档声明的原生路由 | overview 原文："**Native provider routes** — Anthropic `/v1/messages` and Gemini routes are served directly" | handbook overview |
| 501 占位 | `/v1/files`、`/v1/fine-tunes`、`/v1/images/variations` → `501 api_not_implemented` | handbook §5 |
| 限流 | 20 RPM（标准客户端）／120 RPM（howtofixcli），429 携带 `Retry-After`；503 为维护 | handbook §8 |
| 计费 | 严格预付余额，按 input／output／cache read 分别计价，完成后扣费；Free Credit 与 Pro Credit 两层 | handbook §6 |
| 余额接口 | 文档「Programmatic Billing API」：`GET /v1/dashboard/billing/subscription`（hard limit／current spend／active balance，USD）、`GET /v1/dashboard/billing/usage`；两者均可直接用 `sk-` Key | handbook §6；实测未带 Key 返回 401（路由已注册），字段形状**未用真实 Key 验证** |
| 公开模型目录 | `GET /api/pricing` 返回 12 条，全部 `supported_endpoint_types = ["openai"]` | 实测 |
| 已知上下文 | 仅 `deepseek-v4.1-flash`／`deepseek-v4.1-flash-1017`／`glm-5.3-flash` = 1M，`grok-4.6` = 500K | `/api/pricing` 描述与 tags |
| 分组限制 | `hy4` 仅 `vip` 分组；其余模型在 `cli/free/vip/default` | `/api/pricing` `enable_groups` |
| CORS／追踪 | 预检 204、`access-control-allow-origin: *`；响应头 `x-request-id`、Cloudflare | 实测 |

文档称 `howtofix.id/v1` 不是 API host、返回 404，实测 apex 的 `/v1/*` 同样返回 401
（只有 `GET /v1` 空路径 404）。接入仍固定使用 `api.howtofix.id`，不依赖文档与实测的这处出入。

### 仓库现状（只读核对）

- 内置服务商共 13 个（`internal/application/provider/catalog.go:35-51`），其中 `oxionai`
  是「仅 API Key + 按账号目录 + 只透传 Chat」的最小形态（迁移 043），与本平台同构。
- 界面上的「自定义服务商」保存为 `kind=openai-compatible`／`source=custom`，
  **只能配置，不能转发**：`internal/application/routing/presets.go` 的 `supportedProvider`
  与 `internal/application/gateway/routing.go:226` 的 kind 白名单都不含该 kind，
  `modelcatalog/sync.go` 也没有它的目录同步分支。`docs/architecture/OVERVIEW.md:481`
  把这列为「待接适配器」。因此接入 HowToFix **不能**靠自定义服务商绕过，必须新增内置 kind。
  （同一结论已在已取消的 `TASK-20261029-010-subq-provider` 记录中确认。）
- 迁移最大编号为 048；编号按文件名字符串排序且只追加，新迁移取 049。

## Scope

拟纳入实现（待确认后执行）：

- 新增内置服务商 `howtofix`（`builtin_howtofix`），仅 `api_key` 认证，默认启用。
- 新增 `internal/provider/howtofix/`：固定端点、Bearer 转发、账号级 `GET /v1/models` 目录同步。
- Chat Completions 原生透传；Responses／Messages 由网关既有 `agentprotocol` 转换后转发并按原协议回包。
- 路由资格、网关分派、模型目录同步、前端标签／图标／文案、迁移 049 及对应测试。
- 产品文档 `docs/product/HOWTOFIX.md` 与 PRD／OVERVIEW／CURRENT 同步。

## Out of Scope

- 本轮任何产品代码、迁移、前端或依赖改动（Status 保持 `Planned`）。
- Embeddings／Rerank／Images／Audio／Videos／Moderations（网关没有对应协议位）。
- 把 Anthropic `/v1/messages`、Gemini `/v1beta` 当成原生透传（文档声明未经真实 Key 验证）。
- 推理档位（`reasoning_effort`）发布：上游只有 `Reasoning` 标记，没有档位枚举。
- 余额／额度面板（接口存在但字段未验证，见 D3）。
- 签到／分组切换／控制台会话接口（需要 apex 会话态，不是 `sk-` Key 能力）。
- 放开自定义 `openai-compatible` 提供商的路由（独立议题，需要 ADR）。

## Existing Behavior

- 内置服务商才能成为路由目标；`openai-compatible` 自定义配置建了也路由不到。
- 没有余额接口的服务商（River／Atria／普通 OpenAI API Key／自定义）在账号卡与配额页
  自动降级为本机留存的 Token 用量，不需要额外代码。
- 受管客户端（Claude Code／Codex／DSH 等）的模型目录由「已配置的路由」投影生成，
  不存在「接入服务商后还要再为 agent 接一次」的环节。

## Expected Behavior

- 用户添加 HowToFix API Key 账号后，可在账号页「同步官方模型」按该 Key 所属分组取到模型；
  建一条路由即可在网关与各受管客户端选择使用。
- 未同步前该服务商没有任何可路由模型（与 Oxion AI 一致），页面与文档明确说明。
- 转发的 Chat 请求原样透传，`usage` 按上游值记账；Responses／Messages 为网关转换，
  不宣称上游原生协议，转换路径的用量保持未知。
- 401／403／429 沿用既有错误与故障切换语义；402 预付余额不足原样回传客户端（见 D4）。

---

# Analysis

## Related Modules

后端：

- 新增 `internal/provider/howtofix/`：`models.go`（`Kind`、`DefaultBaseURL`、空 `Models()`）、
  `client.go`（`Forward`，只发 `POST /chat/completions`）、`discovery.go`（`DiscoverModels`）。
- `internal/application/provider/catalog.go`：import + `definitions` 追加一条
  `DiscoveryOnly: true` 的定义。
- `migrations/049_builtin_howtofix.sql`：只追加一行 `providers`，不重置已有开关与凭据。
- `internal/application/gateway/`：`service.go` 增加 client 字段与构造、`agent.go` 增加分派、
  `howtofix.go` 新增 `forwardHowToFix`、`routing.go` 两处 kind 白名单。
- `internal/application/modelcatalog/`：`service.go` 注入 client、`sync.go` 增加同步分支。
- `internal/application/routing/`：`presets.go` 的 `supportedProvider`／`presetModels`、
  `catalog.go` 的 `eligible`。
- 硬编码计数：`main_test.go:37`（13 → 14）、
  `internal/application/provider/service_test.go:27`（`TestCatalogContainsThirteenProviderKinds`）。

前端：

- `frontend/src/features/providers/provider-labels.ts`、`provider-brand-icon.tsx`、
  `frontend/src/i18n/locales/{zh,en}.ts`、`frontend/public/provider-icons/howtofix.png` 与 `SOURCE.md`。

文档：`docs/product/HOWTOFIX.md`（新建）、`docs/product/PRD.md`、`docs/architecture/OVERVIEW.md`、
`docs/plans/CURRENT.md`。

## Dependencies

无新增第三方依赖。复用 `internal/core/proxy`、`internal/core/agentprotocol`、
`internal/core/modelcatalog`、`internal/core/netproxy` 与既有 DPAPI 凭据存储。

## Risks

- **模型元数据大面积缺失**：公开 12 个模型里只有 4 个能确认上下文，2 个带能力 tags。
  目录同步只能按 `GET /v1/models` 实际返回的字段映射，缺字段保持未知，不得臆造。
- **分组差异**：`hy4` 仅 `vip` 分组，账号能看到的目录取决于所属分组，因此必须按账号同步，
  不做静态预设。
- **原生协议声明不可验证**：文档称 Anthropic／Gemini 路由是原生直出，但没有真实 Key 无法验证；
  首版一律按 OpenAI Chat 接入，Responses／Messages 走网关转换，避免把转换结果说成原生。
- **`sk-` 与 OpenAI Key 撞形**：账号自动命名取 Key 前若干字符，可能看起来像 OpenAI Key；
  `KeyPrefix` 只作表单提示，不加强制校验。
- **限流 20 RPM**：高频客户端会更快触发 429；沿用既有 `Retry-After` 处理，不额外放大重试。
- **计费不可逆**：真实 Key 的端到端验证会产生实际扣费，必须由用户授权后执行。
- **上游稳定性**：new-api 定制发行版，端点与模型随时可能变动；以官方 handbook 与实测为准。

---

# Implementation Plan

## Phase 1：适配器与内置注册

1. `internal/provider/howtofix/models.go`：`Kind = "howtofix"`、
   `DefaultBaseURL = "https://api.howtofix.id/v1"`、`KeyPrefix = "sk-"`、`Models()` 返回空切片。
2. `client.go`：隔离 transport、禁重定向、只接受 `chat` 协议；`endpointURL` 只允许 HTTPS 或
   本机回环 HTTP，Base URL 已含 `/v1` 时不再追加版本前缀；Key 校验为空／超长／非可打印 ASCII 拒绝；
   `Authorization` 用完即删，流式请求带 `Accept: text/event-stream`。
3. `discovery.go`：`GET {base}/models`，1 MiB／1000 条有界解析，按 `type=chat` 过滤，
   `context_window` 与 `capabilities` 有则映射、无则留空；空目录／非 JSON／非 200 统一失败。
4. `migrations/049_builtin_howtofix.sql` + `internal/application/provider/catalog.go`：
   `{Kind, Name: "HowToFix", DefaultBaseURL, AuthMethods: []string{AuthAPIKey}, DiscoveryOnly: true}`。

## Phase 2：路由与网关

5. `routing/presets.go`：`supportedProvider` 加入 `howtofix.Kind`；`presetModels` 显式返回空。
6. `routing/catalog.go`：`eligible` 增加 `api_key` 分支。
7. `gateway/routing.go`：kind 白名单与 `api_key` 校验分支加入 `howtofix`。
8. `gateway/service.go` + `agent.go` + 新 `howtofix.go`：Chat 原样透传并 `ObserveResponse(..., "chat")`；
   Responses／Messages 先 `agentprotocol.ToChat`，非 200 直接回传，200 用 `ReadChat` + `Encode`
   按原协议与流式标志回包；不注入 `stream_options`／`reasoning_effort`。
9. `modelcatalog/service.go` + `sync.go`：注入 client，新增 `howtofix` + `api_key` 同步分支。

## Phase 3：前端

10. `provider-labels.ts`、`zh.ts`／`en.ts` 增加 `providers.kind.howtofix`；
    `provider-brand-icon.tsx` 增加图标条目；下载官方 logo 到
    `frontend/public/provider-icons/howtofix.png`，在 `SOURCE.md` 记录来源、尺寸、SHA-256 与授权说明。

## Phase 4：测试与质量门

11. 适配器契约测试：路径与 Bearer 头、拒绝非 Chat 协议、拒绝空／非 ASCII Key、拒绝明文远端
    HTTP／允许回环、SSE 与 JSON 透传、目录过滤与能力映射、异常响应。
12. 迁移测试：只追加、幂等、不重置已有开关与 `credential_ref`。
13. 路由／网关集成测试：未同步不可路由 → 同步后可路由；Responses／Messages 转换往返与流式回包；
    停用账号／服务商、改用非 `api_key` 认证均被拒。
14. 前端定向测试 + `task verify`（format → lint → typecheck → test → build）。

## Phase 5：文档

15. 新建 `docs/product/HOWTOFIX.md`：接入参数、目录规则、协议边界（Chat 原生／其余转换）、
    能力未知项、限流、计费与余额现状、明确未验证项。
16. `PRD.md`（内置服务商数量与列表）、`OVERVIEW.md`（提供商段落与网关适配列表）、
    `CURRENT.md` 看板；完成后回填 Implementation Result／Verification 并提交。

## 决策点（Needs Confirmation）

- **D1 协议范围**：推荐首版按 Oxion 形态——Chat 原生 + Responses／Messages 网关转换，
  不发布推理档位；即使文档称 Anthropic 路由为原生也不透传。
- **D2 模型目录**：推荐 `DiscoveryOnly`，不内置 12 条静态预设（分组差异 + 元数据缺失）。
- **D3 余额面板**：文档提供 `/v1/dashboard/billing/subscription` 与 `/usage`，但字段未验证。
  推荐首版不做（自动降级为本机 Token 用量），或由用户提供一次性测试 Key 后再作为第二阶段接入。
- **D4 402 语义**：预付余额不足时是否触发同池换号。推荐首版原样回传，不加入 `failover_policy`。
- **D5 品牌图标**：需下载官方 logo 并记录来源与 SHA-256；若不便获取，先用通用图标并在文档注明。

---

# Progress

- 2026-11-15：完成只读调研——抓取官方 handbook 正文（`/api/handbook/docs`）、
  `/api/pricing` 与 `/api/status`，逐路径探测端点、认证与错误形状；
  核对仓库内置服务商、路由白名单、网关分派、目录同步与前端清单，
  确认「自定义服务商不可路由，必须新增内置 kind」，产出本方案并留待确认。
- 2026-11-15：用户确认现在实现、余额面板要一起做、可提供真实 Key 验证。
  按 `builtin_oxionai` 形态落地：新增 `internal/provider/howtofix`（目录／Chat／余额）、
  迁移 049、路由与网关分派、模型目录同步、余额分派、前端标签图标与余额面板，并更新硬编码计数。
- 2026-11-15：真实 Key 复核改变了一处设计——`/dashboard/billing/usage` 在该部署中
  始终返回 `total_usage: 0`，不反映真实支出；`soft_limit_usd`／`system_hard_limit_usd`
  与 `hard_limit_usd` 同值。因此余额只读取并展示 `hard_limit_usd`（剩余美元），
  删掉「额度上限／已用」两项，避免展示恒为零的误导数字。

---

# Implementation Result

## Changes

- 新增 `internal/provider/howtofix/`：
  - `models.go`：`Kind = "howtofix"`、`DefaultBaseURL = "https://api.howtofix.id/v1"`、
    `KeyPrefix = "sk-"`、`Models()` 返回空（目录按账号分组发放，无预设）。
  - `client.go`：隔离传输、禁重定向、只接受 `chat` 协议；Base URL 已含 `/v1`，
    只拼 `/chat/completions`；Key 非空／≤8192／可打印 ASCII；`Authorization` 用完即删。
  - `discovery.go`：`GET /models`，1 MiB／1000 条有界解析，`type` 存在时只收 `chat`，
    `context_window`／`capabilities` 有则映射、无则保持未知；空目录／非 JSON／非 200 一律失败。
  - `balance.go`：`GET /dashboard/billing/subscription` 读取 `hard_limit_usd` 作为剩余美元余额，
    两位小数只用于展示；缺失字段按 `invalid_response` 失败，不当作零。
- 新增 `migrations/049_builtin_howtofix.sql`：只追加 `builtin_howtofix` 一行，默认启用。
- `internal/application/provider/catalog.go`：`definitions` 追加 `DiscoveryOnly: true` 定义；
  连带更新 `service_test.go`、`main_test.go` 的硬编码计数（13 → 14）与 kind 清单。
- `internal/application/routing/`：`presets.go` 的 `supportedProvider`／`presetModels`、
  `catalog.go` 的 `eligible` 增加 `howtofix` + `api_key` 分支。
- `internal/application/gateway/`：`service.go` 增加 client 字段与构造、`agent.go` 分派、
  新增 `howtofix.go`（Chat 原生透传；Responses／Messages 经 `agentprotocol` 转换后按原协议回包）、
  `routing.go` 两处 kind 白名单与 `api_key` 校验。
- `internal/application/modelcatalog/`：`service.go` 注入 client、`sync.go` 增加同步分支。
- `internal/application/balance/`：`service.go` 的 `Info` 增加 `howtofix` 字段、
  新增 `howtofix.go` 读取分支（错误码映射与 kimi／stepfun 一致）。
- 前端：`provider-labels.ts`、`provider-brand-icon.tsx`、
  `frontend/public/provider-icons/howtofix.png`（官方 logo，256×256，附 `SOURCE.md` 记录）、
  `i18n/locales/{en,zh}.ts`、`account-balance.tsx` + 新 `howtofix-balance.tsx`、
  `provider-accounts-view.tsx`（余额白名单）、`use-gateway-preparation.ts`（接入检查白名单）。
- Wails 绑定由 `wails3 generate bindings` 重新生成（新增
  `frontend/bindings/omniproxy/internal/provider/howtofix/`，`balance/models.ts` 增加字段），未手改。

## Files Changed

- `internal/provider/howtofix/{models,client,discovery,balance}.go` 与三个 `*_test.go`
- `migrations/049_builtin_howtofix.sql`
- `internal/application/provider/{catalog.go,service_test.go,howtofix_migration_test.go}`、`main_test.go`
- `internal/application/routing/{presets.go,catalog.go,howtofix_test.go}`
- `internal/application/gateway/{service.go,agent.go,routing.go,howtofix.go,howtofix_test.go}`
- `internal/application/modelcatalog/{service.go,sync.go}`
- `internal/application/balance/{service.go,howtofix.go,howtofix_test.go}`
- `frontend/src/features/providers/{provider-labels.ts,provider-brand-icon.tsx,provider-labels.test.ts}`
- `frontend/src/features/accounts/{account-balance.tsx,howtofix-balance.tsx,howtofix-balance.test.tsx,provider-accounts-view.tsx}`
- `frontend/src/features/gateway/use-gateway-preparation.ts`
- `frontend/src/i18n/locales/{en,zh}.ts`、`frontend/public/provider-icons/{howtofix.png,SOURCE.md}`
- `frontend/bindings/**`（生成）
- `docs/product/HOWTOFIX.md`（新建）、`docs/product/PRD.md`、`docs/architecture/OVERVIEW.md`、`docs/plans/CURRENT.md`

## Technical Notes

- 目录按账号分组发放：实测同一 `GET /v1/models` 只返回该 Key 分组可见的 12 个模型
  （`hy4` 属 `vip` 分组，未出现在测试账号的目录里），因此坚持 `DiscoveryOnly`、不内置预设。
- `/v1/models` 只返回 `id`／`object`／`created`／`owned_by`／`supported_endpoint_types`，
  没有名称、上下文与能力字段，因此同步后的模型能力一律保持未知，不臆造 tools／image。
- 余额只取 `hard_limit_usd`：实测 `soft_limit_usd`／`system_hard_limit_usd` 与它同值，
  `total_usage` 恒为 0，展示「额度上限／已用」会给出误导数字，故只保留「剩余额度」。
- 402（预付余额不足）原样回传，不纳入 `failover_policy` 的同池换号，避免用其他 Key 承担支出。
- 不注入 `stream_options`／`reasoning_effort`，也不发布推理档位：上游只有 `Reasoning` 标记，
  没有该适配器验证过的档位枚举。
- 受管客户端（Claude Code／Codex／DSH／OpenCode 等）目录来自已配置路由，建好路由后自动出现，
  本任务不需要为任何客户端单独接线。

---

# Verification

## Tests

- `internal/provider/howtofix`：Bearer 头与固定路径、拒绝非 Chat 协议、拒绝空／非 ASCII Key、
  拒绝明文远端 HTTP／查询串／用户信息／片段、允许回环 HTTP、SSE 与 JSON 透传；
  目录过滤（`type` 非 chat 被排除、重复 ID 去重、非法 ID 失败、空目录失败、401 失败）；
  余额成功、缺字段／非数字／非 JSON／空响应失败、状态码映射、缺 Key 失败。
- `internal/application/provider`：迁移 049 只追加、幂等、不重置开关与 `credential_ref`；
  Catalog 与内置清单计数改为 14。
- `internal/application/routing`：未同步前不可路由 → 同步后可路由 → 别名解析 →
  停用账号／改认证方式／停用服务商均被拒 → 不能路由到别的服务商；目录元数据随路由发布。
- `internal/application/gateway`：三种协议都落到 `/v1/chat/completions`，模型别名改写，
  不发送 `stream_options`／`reasoning_effort`，Chat 原生用量入账、转换路径用量保持未知，
  凭据字节清零。
- `internal/application/balance`：余额来自账号自己的 Key、零余额不算可用、
  401／5xx 映射为固定错误码、回调结束后凭据清零、只请求订阅端点。
- 前端：`providers.kind.howtofix` 标签（中英）、`HowToFixBalanceValues` 的
  中文／英文标签、未知金额显示 `—` 且不出现 0 或 `US$—`、账号卡余额分支集成。

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
EXIT=0
```

`format:check`（gofmt＋prettier）→ `lint`（go vet＋staticcheck＋eslint）→
`typecheck`（tsc --noEmit）→ `test`（`go test . ./internal/...` 全部包通过；
前端 155 文件／940 项测试）→ `build`（Windows amd64 production 构建，
含重新生成绑定与 syso）。日志见 `tmp-verify-howtofix.log`（未入库）。

### 真实 Key 复核（用户提供的账号 Key，2026-11-15）

用本适配器直接调用真实上游（Key 只作为环境变量，未写入任何文件或日志）：

- `DiscoverModels`：12 个模型，名称回退到 ID，能力全部未知（上游不返回名称／上下文／能力）。
- `Forward`：`deepSeek-v4.1-flash-1017-sg` + `max_tokens: 1` 返回 200、`application/json`、
  417 字节，`usage` 为 31 输入／1 输出 Token。
- `ReadBalance`：`currency=USD`、`remaining=50.00`；同一次请求后上游
  `hard_limit_usd` 从 `50` 变为 `49.999998`，确认该字段就是剩余美元余额、扣费粒度为上游整数额度单位。
- `/v1/dashboard/billing/usage` 始终返回 `total_usage: 0`，因此不展示「已用」。

未验证：Anthropic `/v1/messages` 与 Gemini 路由是否为原生直出（本适配器按 Chat 转发，
转换路径已由网关测试覆盖）；长上下文、工具调用与图片在真实模型上的行为；
真实桌面窗口中的余额面板视觉。

---

# Documentation

- [x] Task Record（本文件）
- [x] PRD
- [x] Architecture
- [ ] Tech Stack（未变更技术栈或依赖）
- [x] ADR（判断：不需要，属既有适配器模式的同类扩展）
- [x] CURRENT

---

# Deviations

None.

---

# Git

Not committed by Agent.

---

# Result

结论：HowToFix 是 new-api 系的 OpenAI 兼容网关，可按 Oxion AI 的最小形态新增第十四个内置
服务商接入——API Key 账号、按账号同步 `/v1/models`、Chat 原生透传、Responses／Messages 网关转换，
迁移 049，前端标签与图标。现有「自定义服务商」通道不能承载它（路由白名单不含该 kind）。
模型元数据大面积缺失、原生 Anthropic 路由与余额接口字段均未用真实 Key 验证，
这些不确定性在方案里被明确限定而不是猜测。

---

# Completed

（未完成）
