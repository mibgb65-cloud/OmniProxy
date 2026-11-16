# TASK-20261116-015: WorkBuddy AI 订阅通道（浏览器登录 + Credits 额度）

## Status

Completed

## Created

2026-11-16 16:30（Asia/Singapore）

## Request

用户先问「magpie 是不是可以接入 WorkBuddy 国际版」，确认国际版由 magpie 的社区插件
`@magpie-community/opencode-workbuddy-auth` 提供（订阅侧），随后提出：

> 我想接入到我的OmniProxy里面，通过OmniProxy进行模型的使用，你看看有没有什么好的方案

并确认本轮的取舍：

| 决策点 | 用户选择 |
| --- | --- |
| 首版范围 | **直接做完整订阅登录（OAuth + 额度）** |
| 接入版本 | **只接国际版 WorkBuddy AI**（`www.workbuddy.ai`） |
| 额度 | **首版就要 Credits 三窗口** |

随后明确：**「先出计划，别改代码」**。本轮只建立任务记录，不修改产品代码、配置、
迁移、依赖或前端；不构建、不启动、不执行登录、不读取真实账号凭据、不消耗积分。

## Goal

把 WorkBuddy AI（腾讯 CodeBuddy 计划的国际版订阅）作为 OmniProxy 的**原生内置订阅服务商**
接入：用户在账号页用浏览器完成 WorkBuddy AI 登录后，可通过本机网关以
Chat Completions／Responses／Messages 三种协议调用其模型，模型目录来自账号自己的
`/v3/config`，账号卡与配额页展示官方 Credits 三窗口（计划内／额外／合计）。

不依赖 magpie、Node/Bun 插件宿主或独立桥接服务；参考实现只作为协议事实来源。

## Background

### 参考实现（只读，社区插件 0.2.36）

包：`@magpie-community/opencode-workbuddy-auth`（npm，MIT）。仓库源码
`github.com/magpie-community/plugins` → `packages/workbuddy/`。本轮只读检查了 npm
tarball 内的 `index.mjs`（5,003 行）、`README.md` 与仓库文件清单，**未执行其代码**。

插件一个包提供两个 provider：`workbuddy`（国内版，`copilot.tencent.com`）与
`workbuddy-ai`（国际版，`www.workbuddy.ai`）。本任务只接后者。

#### 协议事实（插件 0.2.36；**未经真实账号验证**）

| 能力 | 事实 |
| --- | --- |
| 浏览器登录 | `POST {endpoint}/v2/plugin/auth/state?platform=workbuddy-ai`，头 `X-No-Authorization／X-No-User-Id／X-No-Enterprise-Id／X-No-Department-Info: true`；返回 `{state, authUrl}`。打开 `authUrl` 时追加 `version=<官方桌面版版本>` 与随机 `loginSessionId`；随后 `GET /v2/plugin/auth/token?state=…` 轮询（未完成时业务码 `11217`，1 秒一次，最多 5 分钟）取 `accessToken／refreshToken／expiresAt／refreshExpiresAt／domain／tokenType`；再 `GET /v2/plugin/login/account?state=…`（Bearer + `X-Domain`）取 `uid` 与企业信息 |
| 续期 | `POST {endpoint}/v2/plugin/auth/token/refresh`，头 `X-Refresh-Token`、`X-Auth-Refresh-Source: plugin`、`X-Domain`、企业头；返回同一组 token 字段。同一 refresh token 只允许一次并发续期 |
| 模型目录 | `GET {endpoint}/v3/config`（签名头 + CLI 的 UA + `X-Requested-With`）→ `data.models[]`：`id／name／maxInputTokens／maxOutputTokens／supportsImages／supportsReasoning／reasoning.{supportedEfforts,defaultEffort,canDisableThinking}`，另 `data.modelPromotions`（价格/免费信息）。按账号的套餐发放 |
| 对话 | `POST {endpoint}/v2/chat/completions`，**只接受流式**；请求体按官方客户端 gzip；非 `WorkBuddy/5.9.3 workbuddy-ai/5.9.3 CLI/2.165.0` 的 UA 被 `10085` 拒绝 |
| 必需请求头 | `Authorization: Bearer`、`X-User-Id`、`X-Domain`、`X-Product: SaaS`、`X-IDE-Type: WorkBuddy`、`X-IDE-Name`、`X-IDE-Version`、`X-Requested-With: XMLHttpRequest`、`X-Agent-Intent: craft`、`X-Agent-Purpose: conversation`、`X-Agent-Type: main`、`X-Agent-Runtime: local-agent`、`Accept: application/json`、`X-Stainless-*`（OpenAI SDK 指纹） |
| 会话/请求 id | `X-Conversation-ID`、`X-Conversation-Request-ID`、`X-Conversation-Message-ID`、`X-Request-ID`；后端把一次用户发送按 `X-Conversation-Request-ID` 计一次请求 |
| 额度 | `GET /billing/meter/get-user-resource-summary`；`POST /billing/meter/get-user-resource-paid-packages`（`PackageCodes`、`NeedRenewInfo`）；`POST /billing/meter/get-user-resource-free-packages`（`PackageCodes`、`SlicePeriodStartTime`、`SlicePeriodEndTime`）；企业账号改读 `/billing/meter/get-enterprise-user-usage` |
| Credits 语义 | 计划内基础（`Credits`）+ 其他一切（`Extra credits`：bonus／充值／赠送）+ 合计（`Total credits`，前两者 set aside，只有它停止账号）；可按包到期时间给出 `runsOutAt／runsOut`；无 total 的预付包是 `Prepaid credits` 余额窗口 |
| 已知拒绝码 | `10085` UA 不符、`11128` 系统提示被拒（unapproved channel）、`11159` thinking 签名不符、`11158` 账号无该模型权益、`11217` 登录未完成 |
| 国际版差异 | **没有每日签到**（无 claim 接口）；**没有 `/v1/messages` 路由**（国内版的 Claude 模型才走 Anthropic Messages），因此本任务全部走 Chat |
| 免费模型日窗 | 厂商不返回；插件用 magpie 的 `usage.jsonl` 自行重算「约 1e8 tokens/24h、固定窗口」 |

#### 桌面登录文件

`CodeBuddyExtension/Data/Public/auth/workbuddy-desktop-ai.info`（macOS
`~/Library/Application Support/…`、Windows `%LOCALAPPDATA%\\…`、Linux
`~/.local/share/…`）。**WorkBuddy 5.6+ 用 `$wbEncrypted` 加密该文件**，插件与 OmniProxy
都无法读取，只能走浏览器登录。本任务因此不做桌面文件导入。

### 仓库现状（只读核对，2026-11-16）

- 内置服务商 14 家（`internal/application/provider/catalog.go`），最近两次接入：
  HowToFix（`api_key` 型，迁移 049）与 DuMate（自定义登录态 `dumate_cookie`，迁移 048）。
- 认证形态共三种：`api_key`、`subscription_login`（**Codex 隔离登录专属**，见
  `internal/application/account/subscription.go` 的 `IsCodexSubscriptionProvider` 硬判
  `kind == "openai"`）、`codex_auth_json`，以及 DuMate 的自定义 `dumate_cookie`。
- 订阅登录横跨 9 层，其中三处硬编码到厂商：`internal/transport/wails/subscription.go`
  （`builtin_google` 分流、`builtin_openai` 白名单、`google_` 会话前缀）、
  `account.CheckSubscriptionProvider`、`gateway/routing.go` 的 kind 白名单。
- 迁移最大编号 049（`migrations/049_builtin_howtofix.sql`），新迁移取 **050**。
- 额度分两族：`internal/application/balance`（**只接受 `api_key` 账号**）与订阅侧的独立绑定
  （`SubscriptionQuota`／`GoogleSubscriptionQuota`／`MiniMaxQuota`／`DuMatePoints`）。
- 模型目录版本由视图 `account_catalog_versions` 从 `credential_ref` 派生；**换凭据引用会
  让整份目录失效**，全仓只有 Google 的 `writeGoogleCredentials(preserve=true)` 做了事务内重绑。
- 前端判断服务商能力全部是散落的 `if`／白名单，无注册表；`routing-editor` 两处白名单
  至今漏了 HowToFix（手动路由里显示「适配器待接入」），说明这类白名单没有测试兜底。

## Scope

- 新增内置服务商 `workbuddy_ai`：kind `workbuddy_ai`、id `builtin_workbuddy_ai`，
  默认启用，只有订阅登录一种认证方式。
- 新增适配器包 `internal/provider/workbuddyai/`：固定端点、请求头与 UA 指纹、gzip、
  流式 Chat 转发、`/v3/config` 目录、token 续期、Credits 额度 DTO、错误码映射。
- 新增账号认证类型 `workbuddy_login`（provider 包定义、account 层别名），DPAPI 保存
  access/refresh/expiry/uid/domain/enterprise；**续期在同一事务内重绑目录版本**。
- 新增登录编排 `internal/application/workbuddy/`：`Begin/Read/Current/Cancel/complete`、
  单飞续期、身份核对、额度读取、登出与删除。
- 网关：`selectRoutedTarget` 白名单与认证分支、`forwardWorkBuddy`（Chat 原生透传；
  Responses／Messages 经既有 `agentprotocol.ToChat` → `ReadChat` → `Encode`），
  401 先刷 token 一次再换号；凭据字节用完清零。
- 路由与目录：`presets`／`catalog`／`model_entries` 三处、账号级 `/v3/config` 同步与
  后台自动补目录、stale-only 去重、失败保留旧快照。
- 额度：Credits 三窗口 +（有则）Prepaid credits 余额；独立 Wails 绑定与配额页分区。
- Wails：登录会话 state 与 7 个订阅绑定分支、额度绑定、事件复用
  `subscription:updated`／`modelcatalog:updated`；绑定经 `wails3 generate bindings` 重新生成。
- 前端：服务商标签/图标/文案、账号页登录入口与弹窗按 kind 参数化、额度组件与配额页分类、
  手动路由白名单、接入检查白名单、目录同步白名单、i18n 中英、组件测试。
- 文档：`docs/product/WORKBUDDY.md`、PRD／OVERVIEW／CURRENT、ADR-026。

## Out of Scope

- WorkBuddy 国内版（`copilot.tencent.com`）及其 Claude 模型的 `/v1/messages` 路由。
- 每日签到（国际版无该接口）。
- 桌面登录文件导入（5.6+ 已加密，读不到）。
- API Key 认证方式（`www.workbuddy.ai/profile/keys`）：协议在插件里存在，但不作为首版
  认证方式；后续如需可作为第二认证方式单独立项。
- 免费模型每日 token 窗口：需要按本机记账重算窗口，语义待真机验证，本任务先不做。
- Embeddings／Images／Videos／Files 等无协议位的接口。
- 队列／平衡轮换策略（首版按候选顺序；多账号本身支持）。
- 把 WorkBuddy 的模型与其他服务商同名模型做别名合并（沿用 `workbuddy_ai/<model>` 前缀）。
- 上游模型清单之外的能力推断：上下文、图片、推理档位只按 `/v3/config` 声明。

## Existing Behavior

- 没有 `workbuddy_ai` kind；账号页无法新增 WorkBuddy AI 账号，网关也没有该分支。
- 订阅登录只有 Codex（`builtin_openai`）与 Google（`builtin_google`）两条并行实现，
  `SubscriptionBegin` 按 provider id 硬编码分流，新 kind 不加分支只会落进 Codex 流程。
- `balance` 服务对非 `api_key` 账号直接返回 `unsupported`，订阅额度必须走独立绑定。
- 受管客户端（Claude Code／Codex／DSH／OpenCode 等）的目录由「已配置的路由」投影生成，
  新增服务商后建一条路由即可出现，不需要为每个客户端单独接线。

## Expected Behavior

- 账号页可为 WorkBuddy AI 新增账号：浏览器登录 → 显示 uid/企业/Plan → 刷新、重新登录、退出。
- 登录完成后后台自动同步 `/v3/config` 目录；未同步前该服务商没有可路由模型。
- 建好路由后：Chat 原生；Responses／Messages 经网关转换（不宣称上游原生）；
  流式与非流式客户端都可用（上游始终流式，非流式由网关聚合）。
- 账号卡与配额页展示 Credits（计划内）、Extra credits、Total credits 的已用/总量/百分比、
  计划周期与最早到期时间；字段缺失显示未知，不补 0。
- 401 先尝试续期一次，仍失败才触发同池换号；402/429 语义按既有策略，不放大重试。
- 上游凭据只存 DPAPI，SQLite 只存 `credential_ref` 与可展示身份；日志与前端不回显 token。

---

# Analysis

## Related Modules

后端：

- 新增 `internal/provider/workbuddyai/`：`models.go`（Kind／Name／DefaultBaseURL／静态模型清单）、
  `credentials.go`、`auth.go`（sign-in state／poll／refresh）、`client.go`（头与 UA、gzip、
  Chat 转发、401 一次重试）、`discovery.go`（`/v3/config`）、`quota.go`（Credits 三窗口）、
  `errors.go`（固定错误码）。
- 新增 `internal/application/workbuddy/`：`service.go`（会话状态机、state 读取、busy/关闭）、
  `login.go`（Begin／poll／complete）、`operations.go`（Use／UseAt／Models／Refresh／Logout／Delete）、
  `quota.go`、`quota_cache.go`（持久化快照，照 `googlesubscription/quota_cache.go`）。
- 新增 `internal/application/account/workbuddy.go`：`CheckWorkBuddyProvider`、
  `SaveWorkBuddyAccount`、`FindWorkBuddyAccount`、`UseWorkBuddyCredentials`、
  `RotateWorkBuddyCredentials`（事务内重绑 `account_models` 与额度快照）、`Logout...`。
- `internal/application/account/{update.go,mutations.go}`：改名分支与删除守卫。
- `migrations/050_builtin_workbuddy_ai.sql`（只追加 `providers` 一行）+ 额度快照表迁移
  `051_workbuddy_quota_snapshots.sql`（照 `038_google_subscription_quota_snapshots.sql`）。
- `internal/application/provider/catalog.go` 与两处硬编码计数测试。
- `internal/application/routing/{presets.go,catalog.go,model_entries.go}` + 新 `workbuddy.go`。
- `internal/application/gateway/{routing.go,service.go,agent.go,key_access.go,subscription_quota.go}`
  + 新 `workbuddy.go`；新增 `WorkBuddySubscriptionAccess` 与 `SetWorkBuddySubscription`。
- `internal/application/modelcatalog/{service.go,sync.go,auto.go}` + 新 `workbuddy_auto.go`。
- `internal/transport/wails/{subscription.go,app.go,gateway.go,accounts.go}`、
  新 `workbuddy_subscription.go`／`workbuddy_quota.go`。

前端：

- `frontend/src/features/providers/{provider-labels.ts,provider-brand-icon.tsx}`、
  `frontend/public/provider-icons/` + `SOURCE.md`。
- `frontend/src/features/accounts/{add-account-dialog.tsx,subscription-login-dialog.tsx,
  subscription-quota.tsx,provider-accounts-view.tsx,account-details-card.tsx,account-details-list.tsx}`
  + 新 `workbuddy-credits.tsx`。
- `frontend/src/features/quota/{quota-model.ts,quota-resource.tsx}`。
- `frontend/src/features/routing/{model-catalog.tsx,routing-editor.tsx,routing-editor-fields.tsx}`。
- `frontend/src/features/gateway/use-gateway-preparation.ts`。
- `frontend/src/services/{subscription.ts,workbuddy.ts}`、`frontend/src/i18n/locales/{zh,en}.ts`。
- `frontend/bindings/**`（生成，不手改）。

文档：

- 新增 `docs/product/WORKBUDDY.md`；`docs/product/PRD.md`、`docs/architecture/OVERVIEW.md`、
  `docs/plans/CURRENT.md`；新增 `docs/decisions/ADR-026-workbuddy-ai-subscription.md`。

## Dependencies

- 不新增第三方依赖：`net/http`、`compress/gzip`、`crypto/rand`、`encoding/json`。
- 复用既有：`internal/core/proxy`、`internal/core/agentprotocol`、`internal/core/modelcatalog`、
  `internal/core/netproxy`、`internal/infrastructure/credential`（DPAPI）、`internal/application/subscription`
  的 `Input/State` DTO。
- 参考实现为 MIT：协议移植需在 `frontend/public/provider-icons/SOURCE.md` 或
  `THIRD_PARTY` 类说明中保留来源声明（照 DuMate 先例）。
- 上游版本与 UA 指纹固定在插件记录的版本（WorkBuddy AI 5.9.3 / CLI 2.165.0），
  需在文档中标注「随上游发版可能失效」。N/A 以外无其他依赖。

## Risks

- **上游身份对齐**：UA 与客户端指纹是硬要求（否则 `10085`），属于「按官方客户端形态发请求」。
  与 ADR-013（Google 打散 system prompt）同级风险，需 ADR 记录并明确不承诺账号安全。
- **协议非官方契约**：所有端点来自社区插件与内部接口，随时可能变；失败必须给出可操作错误码，
  不静默降级。
- **续期导致目录失效**（高危）：凭据引用变化会让 `account_catalog_versions` 变，
  `account_models` 未同步重绑时模型从路由与客户端目录消失。必须照 Google `preserve` 写法，
  并有「续期后仍可路由」的回归测试。
- **认证语义错配**（高危）：复用 `subscription_login` 会让 Codex 的额度/重置卡/清理逻辑
  把 WorkBuddy 账号当成自己的；必须使用自定义 `workbuddy_login` 常量。
- **Wails 分派漏改**：7 个绑定方法 + 会话前缀 + AppService 字段 + shutdown 漏一处，
  表现为登录不可用、会话找不到或后台授权无法取消。
- **额度语义**：Credits 三个窗口的「set aside / 只有 Total 停止账号」是插件对官方面板的解读，
  首版只用于展示，**不参与调度**（不按额度跳过账号），避免解读错误影响可用性。
- **只接受流式**：非流式客户端需要网关聚合，长回复会占用内存；需有上限与取消处理。
- **同名模型**：`glm-5.3`、`deepseek-v4.1-flash`、`kimi-k3` 等与其他服务商同名，
  默认路由必须按 `workbuddy_ai/` 前缀区分，不做自动合并。
- **真实副作用**：登录与模型调用会消耗真实积分；真机验证必须由用户授权并记录消耗。
- **模型清单陈旧**：静态清单只作 fallback，运行时应以 `/v3/config` 为准；两者不一致时以账号为准。

---

# Implementation Plan

按依赖顺序分阶段，每阶段可独立提交与验证。

## Phase 0：决策记录与骨架

1. 新增 `docs/decisions/ADR-026-workbuddy-ai-subscription.md`：登录凭据所有权（网关拥有、
   DPAPI 保存）、请求形态对齐与风险、额度只读不调度、国际版单通道。
2. 迁移 050（providers 一行）与 051（额度快照表），照 048/049 与 038 的写法。

## Phase 1：适配器 `internal/provider/workbuddyai/`

3. `models.go`：`Kind = "workbuddy_ai"`、`Name = "WorkBuddy AI"`、
   `DefaultBaseURL = "https://www.workbuddy.ai"`、静态模型清单（来自插件 `AI_MODELS`，
   标注「参考实现、未真机验证」，仅作 fallback）。
4. `credentials.go`：`Credentials{AccessToken,RefreshToken,ExpiresAt,RefreshExpiresAt,Domain,
   TokenType,UID,EnterpriseID,Plan}` + `Valid()` + `Marshal/Parse`（有界 ≤1 MiB，拒绝控制字符）。
5. `auth.go`：`SignInState`／`PollToken`／`LoginAccount`／`Refresh`；UA 与固定头集中一处。
6. `client.go`：隔离 `http.Client`（禁重定向、`netproxy` 出站代理）、`Forward` 只发 Chat、
   `stream:true`、gzip、会话 id、401 一次重试、`Response` 原样交给核心。
7. `discovery.go`：`GET /v3/config` 有界解析（1 MiB／1000 条），映射上下文/输出/图片/推理档位；
   空目录或非 200 失败，保留调用方旧快照。
8. `quota.go`：`ReadCredits` 读 resource summary（+ paid/free packages，企业走 enterprise meter），
   输出 `Credits{Credits, Extra, Total, Prepaid?}`，比例字段用指针，nil=未知。
9. `errors.go`：`Failure(code)`／`ErrorCode`，映射 10085／11128／11159／11158／11217／401／429。

## Phase 2：账号与凭据生命周期

10. `account/workbuddy.go`：Check／Save／Find／Use／Rotate；`RotateWorkBuddyCredentials`
    在同一事务里写新 ref、更新 `account_models.account_version`、更新额度快照版本。
11. `account/update.go`：登录型账号只允许改名；`mutations.go`：删除走 `DeleteManaged`
    （先停用授权、清密文、失败保留引用）。

## Phase 3：登录编排 `internal/application/workbuddy/`

12. `service.go` + `login.go`：单会话状态机（`starting→waiting→exchanging→verifying→saving→
    completed/failed/cancelled`）、5 分钟上限、取消、关闭；会话 ID 前缀 `workbuddy_`。
13. `operations.go`：`Use/UseAt`（5 分钟内到期先续期、单飞）、`Models`（调 discovery 并写目录）、
    `Refresh`（只读身份，不强制续期）、`Logout`、`Delete`。
14. `quota.go` + `quota_cache.go`：`ReadQuota`／`ReadDisplayQuota`／`CachedQuota`，
    快照只存公开字段（照 ADR-019）。

## Phase 4：路由、网关、目录

15. `provider/catalog.go`：`{Kind, Name, DefaultBaseURL, Models: 静态清单, AuthMethods:
    []string{workbuddyai.AuthMethod}, DiscoveryOnly: true}`；改两处硬编码计数测试（14 → 15）
    并为新 kind 增加例外分支（照 dumate）。
16. `routing/{presets,catalog,model_entries}.go` + 新 `routing/workbuddy.go`：
    `supportedProvider`、`eligible`（登录型且 `authenticated`）、`targetSupports`、
    `hasPresetAccount`。目录仍声明网关支持的三种客户端协议（Chat 原生，Responses／Messages
    由网关转换，与 HowToFix／DuMate 一致）；上游只有 Chat 是适配器事实，**不**据此收窄目录协议，
    但 `ModelMetadata` 不猜测上游推理档位。
17. `gateway`：`service.go` 字段与构造、`routing.go` kind 白名单与认证分支、
    `agent.go` 分派、新 `workbuddy.go`（凭据门 + Chat 透传 + 跨协议转换 + 用量入账）、
    `key_access.go` 的 `useWorkBuddyToken`、`subscription_quota.go` **不加**额度跳过。
18. `modelcatalog`：`SetWorkBuddySubscription`、`sync.go` 分支、`auto.go` 的
    `subscriptionReady` 增加新 auth_method、新 `workbuddy_auto.go`（stale-only）。

## Phase 5：Wails 绑定

19. `app.go` 新增 `workbuddyAuth` 字段与启动/关闭、`gateway.go` 安装
    `SetWorkBuddySubscription`、`accounts.go` 删除分支。
20. 新 `workbuddy_subscription.go`（会话 state、`workbuddySession(id)`、打开浏览器限官方
    HTTPS 域、shutdown）、`workbuddy_quota.go`（Quota／CachedQuota）。
21. `subscription.go` 的 7 个方法各加 `workbuddy_ai` 分支；事件复用既有两类。
22. `wails3 generate bindings` 重新生成 `frontend/bindings/**`（不手改）。

## Phase 6：前端

23. `provider-labels.ts`、`provider-brand-icon.tsx`、`provider-icons/*.png` + `SOURCE.md`、
    `i18n/locales/{zh,en}.ts`（`providers.kind.workbuddy_ai` 与额度文案）。
24. `add-account-dialog.tsx` 的 kind 数组；`subscription-login-dialog.tsx` 把 google 二分重构为
    按 kind 的「品牌 + 文案 key + 是否支持设备码」配置；`subscription-quota.tsx` 增加分支 +
    新 `workbuddy-credits.tsx`（未知显示 `—`，不显示 0）。
25. `provider-accounts-view.tsx` 两处 `balanceSupported`/提示链、`account-details-card.tsx` /
    `account-details-list.tsx` 分支、`quota-model.ts` 分类与状态、`quota-resource.tsx` 渲染。
26. `routing/model-catalog.tsx`、`routing-editor.tsx`、`routing-editor-fields.tsx`、
    `use-gateway-preparation.ts`（同时补 HowToFix 漏项，作为回归）。
27. `services/workbuddy.ts` 封装与错误码白名单、`services/subscription.ts` 的 provider 参数化。

## Phase 7：测试与验收

28. 适配器契约测试（loopback 假上游）：头与 UA、gzip、只发 Chat、401 一次重试、错误码映射、
    目录解析边界、额度 DTO（nil 不补 0）。
29. 账号/凭据测试：身份核对、重新登录保持身份、**续期后目录仍可路由**、删除清理失败保留 ref。
30. 登录流程测试：假 sign-in 服务（state／poll／account）、超时、取消、busy、身份不一致。
31. 路由/网关测试：未同步不可路由 → 同步后可路由 → 停用/改认证/停用服务商均拒；三协议落点；
    非流式客户端由聚合得到完整回包；用量入账；凭据清零。
32. 迁移测试：050/051 只追加、幂等、不重置开关与 `credential_ref`。
33. 前端测试：标签、登录入口、额度组件（未知 ≠ 0）、配额分类、路由白名单可提交。
34. `task verify`（format → lint → typecheck → test → build），记录结果。
35. 真机验证（需用户授权，会产生真实消耗）：登录、目录、一次 Chat、额度读数；记录消耗与结果。

## Phase 8：文档与收尾

36. `docs/product/WORKBUDDY.md`（账号、登录、协议边界、额度语义、错误码、未验证项、风险）。
37. PRD（内置服务商数量与清单）、OVERVIEW（服务商段与网关适配列表）、CURRENT 看板。
38. 回填 Implementation Result／Verification／Deviations，提交并把 commit 写入任务记录。

## 待确认项

- **Q1 免费模型日窗**：首版不做（本计划取值）；若要求同期做，需要基于本机 `usage_accounts`
  重算固定 24h 窗口，并接受「语义未经官方确认」的标注。
- **Q2 额度是否参与调度**：计划取值「不参与」，只展示。若要求按 Total credits 跳过账号，
  需要额外接入 `subscription_quota.go` 并扩大回归面。
- **Q3 上游版本指纹更新**：首版固定 5.9.3／2.165.0；是否需要「跟随已安装桌面版」读取版本，
  还是接受发版后手工更新（计划取值：手工，文档标注）。
- **Q4 模型清单**：首版静态 fallback 用插件 `AI_MODELS`（16 条）。是否需要先真机拉一次
  `/v3/config` 再定清单（计划取值：静态 fallback + 运行时空目录失败保留旧快照）。

---

# Progress

- 2026-11-16：完成只读调研（社区插件 0.2.36 源码、README；OmniProxy 的 provider/account/
  subscription/gateway/routing/modelcatalog/balance/Wails/前端接入面），产出本方案。
  用户确认「完整订阅登录 + 只接国际版 + 首版含 Credits 三窗口」并要求先出计划。
  本轮未修改任何产品代码、配置、迁移或依赖；Status 保持 `Planned`。
- 2026-11-16：用户授权实施。按 Phase 0–8 完成适配器、账号与凭据生命周期、登录编排、
  路由/网关/目录、Wails 绑定（重新生成）、前端、迁移 050／051、文档与 ADR-026；
  各层分项测试随实现落地。
- 2026-11-16 18:56：`task verify` exit 0（format／lint／typecheck／test／build 全绿；
  前端 156 files／961 tests）。此前一次 verify 失败于既有的端口争用偶发测试
  `TestGatewayStartupAvoidsOccupiedPort`，隔离复跑 3 轮通过，已在 Verification 记录。
  真机验证按计划未执行（消耗真实积分，需用户授权）。

---

# Implementation Result

## Changes

- 新增 `internal/provider/workbuddyai/`：协议适配器（Kind／静态展示清单、凭据、浏览器登录
  与轮询、续期、`/v3/config` 目录、`/billing/meter` Credits 三窗口、固定错误码），
  并附契约测试（签名头与 UA、gzip、只发 Chat、流式要求、401/429/业务码映射、目录与额度解析边界）。
- 新增 `internal/application/workbuddy/`：登录会话状态机（`Begin/Read/Current/Cancel/Close`）、
  提前 5 分钟单飞续期、目录读取、刷新/登出/删除、Credits 额度与保留期内存快照。
- 新增 `internal/application/account/workbuddy.go`：`workbuddy_login` 认证类型、
  受保护凭据（令牌 + 待清理旧引用）、按 uid 查找/复用账号、只改名、删除守卫；
  `RotateWorkBuddyCredentials` 在同一事务内重绑 `account_models` 的授权版本，
  普通续期不会让目录失效。
- 新增迁移 `050_builtin_workbuddy_ai.sql`（第十五个内置服务商，默认启用）与
  `051_workbuddy_quota_snapshots.sql`（额度快照表，为后续持久化预留，本期未写入）。
- 内置注册与路由：`provider/catalog.go` 增加 `workbuddy_ai`（登录型、DiscoveryOnly）；
  `routing/{presets,catalog,model_entries}.go` 增加资格、预设与目录分支；
  `gateway/{service,routing,agent,key_access}.go` 增加白名单、认证分支、分派与凭据门；
  新增 `gateway/workbuddy.go`：上游始终流式，Chat 客户端转 SSE 原样透传，
  非流式 Chat 与 Responses／Messages 由 `mergeChatStream` 聚合后按原协议编码回包。
- 目录同步：`modelcatalog/{service,sync,auto}.go` 增加订阅分支、`SetWorkBuddySubscription`、
  `StartWorkBuddyAutoSync` 与 `workbuddyMetadata`（未知字段保持未知、档位词表有界）。
- Wails 绑定：新增 `workbuddy_subscription.go`（登录会话、官方 HTTPS 打开、额度绑定、目录同步、
  关闭），`subscription.go` 七个方法按会话前缀／账号归属分派，`app.go`／`gateway.go`／
  `accounts.go` 接入启动、网关安装与删除；`frontend/bindings/**` 由 `wails3 generate bindings` 重新生成。
- 前端：服务商标签/订阅文案/官方图标与 `SOURCE.md`；添加账号入口与登录弹窗按 kind 参数化
  （WorkBuddy 只走浏览器登录）；新增 `services/workbuddy.ts`、`use-workbuddy-quota.ts`、
  `workbuddy-credits.tsx`（三窗口 + 预付余额，未知显示 `—`、`aside` 说明、刷新与缓存回退）；
  账号卡/列表订阅分支、配额页 `workbuddy` 资源类型、路由编辑器与目录同步白名单
  （并补上既有遗漏的 `howtofix`）、接入检查白名单；中英文案与 4 个新测试。
- 文档：新增 `docs/product/WORKBUDDY.md`、`docs/decisions/ADR-026-workbuddy-ai-subscription.md`；
  更新 PRD、架构 OVERVIEW、CURRENT 看板。

## Files Changed

主要文件：

- `internal/provider/workbuddyai/{workbuddyai,credentials,auth,client,discovery,quota,errors}.go`
- `internal/application/workbuddy/{service,login,operations,quota}.go`
- `internal/application/account/{workbuddy.go,update.go,mutations.go}`
- `internal/application/gateway/{workbuddy.go,service.go,routing.go,agent.go,key_access.go}`
- `internal/application/modelcatalog/{workbuddy.go,workbuddy_auto.go,service.go,sync.go,auto.go}`
- `internal/application/routing/{presets.go,catalog.go,model_entries.go}`
- `internal/application/provider/{catalog.go,service_test.go,workbuddy_migration_test.go}`、`main_test.go`
- `internal/transport/wails/{workbuddy_subscription.go,subscription.go,app.go,gateway.go,accounts.go,google_model_sync.go}`
- `migrations/{050_builtin_workbuddy_ai.sql,051_workbuddy_quota_snapshots.sql}`
- `frontend/src/{features/providers,features/accounts,features/quota,features/routing,features/gateway,services,i18n,public}` 相关文件
- `frontend/bindings/**`（生成）
- `docs/product/WORKBUDDY.md`、`docs/decisions/ADR-026-workbuddy-ai-subscription.md`、
  `docs/product/PRD.md`、`docs/architecture/OVERVIEW.md`、`docs/plans/CURRENT.md`

## Technical Notes

- 上游只接受流式：网关把每个请求改成 `stream: true` 并注入 `stream_options.include_usage`；
  Chat 流式原样透传，非流式或跨协议在进程内聚合成完整快照（文本、工具调用分片、finish、
  usage），再交给 `agentprotocol.ReadChat`／`Encode`，不把半个工具调用交给客户端。
- Credits 语义：`Credits` 与 `Extra credits` 都是 `aside`，只有 `Total credits` 决定账号可用性；
  首版只展示，不参与调度。字段缺失保持未知，界面显示 `—`。
- 续期与目录：令牌临近到期才续期（5 分钟余量、单飞）；续期在同一事务重绑
  `account_models.account_version`，避免「刷新令牌后模型静默消失」的既有坑。
- 未使用 `subscription_login`：该认证类型是 Codex 隔离登录代号（ADR-024），
  WorkBuddy 使用独立的 `workbuddy_login`，避免 Codex 的额度与清理逻辑误认账号。
- 上游版本指纹固定在 WorkBuddy AI 5.9.3／CLI 2.165.0，随官方发版可能需要更新常量。


---

# Verification

## 质量门（`task verify`，2026-11-16 18:42–18:56 Asia/Singapore）

```text
task verify   →   exit 0
```

| 阶段 | 结果 |
| --- | --- |
| format check | `gofmt -l main.go internal` 无输出；`prettier --check .` → All matched files use Prettier code style |
| lint | `go vet . ./internal/...`、`staticcheck . ./internal/...` 无告警；`eslint .` 0 error（仅 `reset-credit-list.tsx` 两条既有 warning） |
| typecheck | `tsc --noEmit` exit 0 |
| test（后端） | `go test . ./internal/...` 全部 `ok`，含新增包 `internal/provider/workbuddyai`、`internal/application/workbuddy`、`internal/application/gateway`（WorkBuddy 三条）、`internal/transport/wails`（20.9s） |
| test（前端） | `pnpm test` → 156 files / 961 tests passed（基线 155 files / 948 tests，本次 +1 文件 +13 测试） |
| build | `wails3 generate bindings` 重新生成（与工作树一致）+ `tsc && vite build`（2788 modules，8.52s）+ `go build -tags production` → `bin/omniproxy.exe` |

首次 `task verify` 曾在最后一步失败于 `internal/transport/wails` 的
`TestGatewayStartupAvoidsOccupiedPort`（`gateway: port_unavailable`）。该测试与本次改动无关
（启动期端口探测与其他包测试进程的临时监听争用），隔离复跑 3 轮全部通过：

```text
go test ./internal/transport/wails -run 'TestGatewayStartupAvoidsOccupiedPort|TestSetProxyPortStillAvoidsForeignOccupiedPort' -count=3
→ --- PASS ×6, PASS, ok  omniproxy/internal/transport/wails  1.634s
```

随后的完整 `task verify` 该包 20.9s 通过，认定为环境级偶发，不作为本次改动缺陷。

## 分项自测（实现期间逐层验证）

| 范围 | 命令 | 结果 |
| --- | --- | --- |
| 适配器契约 | `go test ./internal/provider/workbuddyai/ -count=1` | 通过（签名头与 UA、gzip 请求体、只发 Chat、`stream:true`、401 一次重试、10085/11128/11159/11158/11217 映射、目录与额度解析边界、nil 不补 0） |
| 登录编排 | `go test ./internal/application/workbuddy/ -count=1` | 通过（state／poll／account 假上游、超时、取消、busy、身份不一致） |
| 网关 | `go test ./internal/application/gateway/ -count=1` | 通过（含 `workbuddy_test.go` 三条：流式透传、非流式聚合、跨协议编码） |
| 内置注册与迁移 | `go test ./internal/application/provider/ ./-count=1` | 通过（`TestCatalogContainsFifteenProviderKinds`、`workbuddy_migration_test.go`） |
| 绑定一致性 | `wails3 generate bindings -clean=true -ts -i` | exit 0，140 methods / 7 events；`git status frontend/bindings` 与提交前一致 |

## 未验证（需用户授权）

- **真机验证未做**：浏览器登录、`/v3/config` 真实目录、一次真实 Chat 调用、Credits 四窗口读数
  都会产生真实账号副作用或消耗积分，本任务只做到假上游契约测试。协议事实仍以社区插件
  0.2.36 为依据，`docs/product/WORKBUDDY.md` 已逐条标注「未经真实账号验证」。

---

# Documentation

- [x] PRD（内置服务商数量与清单、WorkBuddy AI 条目）
- [x] Architecture（新增 WorkBuddy AI 订阅段、外部依赖与适配器清单）
- [x] Tech Stack（未新增依赖，无需变更）
- [x] Coding Rules（无需变更）
- [x] Testing（沿用既有测试约定，无需变更）
- [x] Build（绑定生成流程未变，无需变更）
- [x] ADR（新增 ADR-026）
- [x] CURRENT（已实现条目；完成后移入 completed/）
- [x] 产品说明：新增 `docs/product/WORKBUDDY.md`

---

# Deviations

与 Implementation Plan 的实际差异（全部为收敛或等价实现，无范围扩张）：

| 计划 | 实际 | 原因 |
| --- | --- | --- |
| `provider/workbuddyai/models.go` 单独放常量与静态清单 | 合并进 `workbuddyai.go` | 该文件需要 `Kind／Name／DefaultBaseURL／AuthMethod／AppVersion／CLIVersion` 与 `Models/DisplayModels/ModelName/SupportsModel/ModelMetadata` 一处维护，拆开只增加跳转 |
| `provider/catalog.go` 写 `Models: 静态清单` | `Models: nil` + `DiscoveryOnly: true`，静态清单只经适配器 `Models()` 暴露 | 把静态清单写进 catalog 会让 `routing` 预设发布 `workbuddy_ai/*` 模型，`TestDefaultRoutesAreDerivedWithoutCreatingRules` 失败；目录必须以账号 `/v3/config` 为准 |
| `workbuddy/quota_cache.go` 持久化快照（照 `googlesubscription/quota_cache.go`） | 内存快照（`quota.go` 的保留期缓存），迁移 051 建表但本期不写入 | 额度只做展示、不参与调度，重启后一次拉取即可恢复；持久化留作后续需求，表已就位避免再加迁移 |
| `account.RotateWorkBuddyCredentials` 同时重绑额度快照版本 | 只重绑 `account_models.account_version` | 同上：没有持久化额度快照可重绑 |
| `transport/wails/workbuddy_quota.go` 独立文件 | 额度绑定并入 `workbuddy_subscription.go` | 同属一个订阅服务，共用一个 `workbuddySubscriptionService` |
| `modelcatalog/workbuddy_auto.go` | 另加 `modelcatalog/workbuddy.go`（`WorkBuddySubscription` 接口与 `workbuddyMetadata`） | 与 `google.go`／`dumate` 的既有分层保持一致 |
| `provider-icons/*.png` | `provider-icons/workbuddy-ai.svg` | 现有图标机制以 svg 为主，官方来源只有 svg |
| 新增绑定方法 | 未新增 `WorkBuddyCredits` 之外的方法；会话与额度复用订阅绑定 + 2 个额度方法 | 7 个既有订阅方法按 `workbuddy_` 前缀分派即可，避免绑定面扩张 |
| Phase 7 第 35 步「真机验证」 | 未执行 | 需用户授权且消耗真实积分；已在 Verification 明确标注 |

计划外但必要的顺带改动（均在本任务接线的同一批白名单里，且有测试覆盖）：

- `routing-editor.tsx`／`routing-editor-fields.tsx` 补上既有遗漏的 `howtofix`（计划中即列为回归项）。
- `use-gateway-preparation.ts`、`provider-accounts-view.tsx` 的接入检查／余额提示链排除 `workbuddy_ai`，
  否则会提示「该服务商不支持余额」这类误导文案。
- `account-edit-form.tsx`、`delete-account-dialog.tsx`、`account-provider-card.tsx` 的认证方式分支：
  后端 `update.go` 对 `AuthWorkBuddyLogin` 带 key 会返回 `invalid_input`，前端不能显示 API Key 替换。
- `internal/application/account/mutations.go` 的受管账号守卫加入 `AuthWorkBuddyLogin`，
  保证删除走「先停用授权、再清密文、失败保留引用」的既有路径。

---

# Git

## Commit

Not committed by Agent.

---

# Result

## 达成

WorkBuddy AI（国际版）已成为 OmniProxy 的第十五个内置服务商，形态与 Codex／Google 订阅同级，
不依赖 magpie、Node 插件宿主或任何桥接进程：

- **账号**：账号页可按 `workbuddy_ai` 新增账号，唯一认证方式为浏览器授权登录；登录态
  （access／refresh／到期／uid／domain／企业）只进 DPAPI，SQLite 只有 `credential_ref` 与
  可展示身份。支持刷新、重新登录、退出、删除。
- **目录**：登录后后台自动同步账号自己的 `/v3/config`，模型名、上下文、输出上限、图片与
  推理档位按上游声明建模；失败保留旧快照，普通续期不再让目录失效。
- **网关**：`workbuddy_ai/*` 路由在 Chat Completions 上原生流式透传，非流式 Chat 与
  Responses／Messages 由网关聚合后按原协议编码；401 先单飞续期一次再换号。
- **额度**：账号卡与配额页展示 Credits（计划内）、Extra credits、Total credits 与
  （有则）Prepaid credits，字段缺失显示未知而不是 0。额度只展示，不参与调度。
- **质量门**：`task verify` exit 0（format／lint／typecheck／test／build 全绿；
  后端 `go test . ./internal/...` 全通过，前端 156 files / 961 tests 通过）。

## 未达成／后续

- 真机验证（登录 + 真实目录 + 一次 Chat + 额度读数）未做，会消耗真实积分，需用户授权后单独执行。
- 免费模型每日 token 窗口（Q1）、额度参与调度（Q2）、跟随桌面版更新上游版本指纹（Q3）、
  以真机目录替换静态 fallback（Q4）仍按计划保持未做，待真实账号证据。
- 上游版本指纹固定在 WorkBuddy AI 5.9.3／CLI 2.165.0，官方发版后可能需要更新常量
  （`internal/provider/workbuddyai/workbuddyai.go`）。

## 风险状态

ADR-026 记录的三个风险均已按计划处置：请求形态对齐（UA／签名头／gzip）集中在适配器一处并
有契约测试；额度语义只读不调度；续期重绑目录版本有回归测试。协议非官方契约这一点无法消除，
已在 `docs/product/WORKBUDDY.md` 逐条标注验证状态。

---

# Completed

2026-11-16 18:56（Asia/Singapore）
