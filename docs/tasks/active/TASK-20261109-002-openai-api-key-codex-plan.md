# TASK-20261109-002: OpenAI API Key 接入与 Codex 使用规划

## Status

In Progress

## Created

2026-11-09（Asia/Singapore）

## Request

用户在 https://platform.openai.com/ 拥有 API Key 和 40 美元额度，询问：

1. 如何把该 API Key 接入 OmniProxy，让 Codex 调用时使用 API Key 的额度；
2. 是否有查询剩余余额的接口；
3. 希望先给出实施计划。

## Goal

明确一条可交付的接入路径：OpenAI 按量 API Key 能作为 OmniProxy 的上游转发账号，Codex 能通过本地网关调用并消耗该 Key 的 40 美元额度；同时明确“剩余额度”在官方接口层面的真实可得性，区分官方余额、管理 API 消费数据和本机估算，不用估算值冒充官方余额。

## Background

当前 main `3be2558e`（本地 `v2.2.3-beta.2` 之后）已支持 OpenAI **ChatGPT / Codex 订阅**账号的授权、模型目录、额度、重置卡和 `/codex/v1` 透明转发；OpenAI 的 API Key 认证方式虽然在提供商目录和账号表单中存在，但只保存凭据，尚未接入转发、目录和额度链路。

用户提到的 40 美元是 platform.openai.com 的**按量计费预付额度**，与 ChatGPT 订阅额度是两套计费通道，不能复用 Codex 订阅的 5h／周窗口。

## Scope

本任务先完成规划，包含：

- 官方 Codex API Key 认证方式与配置字段；
- 官方剩余额度接口的可得性核对；
- 当前代码缺口和需要修改的模块；
- 分阶段实施计划、验收条件和风险。

后续实施预计包含：OpenAI API Key 上游适配器、路由资格、模型目录、请求记账、Codex 接入方式和可选的额度展示。

## Out of Scope

- 本轮不修改代码、迁移、界面或客户端配置；
- 不读取、保存或验证用户的真实 Key；
- 不发起会消耗 40 美元额度的模型请求；
- 不实现网页抓取、浏览器 Cookie／Session Key 登录或非公开接口；
- 不把 API Key 账号与 ChatGPT 订阅账号混为同一计费池；
- 不承诺 Codex 的所有 ChatGPT 专有功能（订阅压缩、Fast 档位等）在 API Key 通道可用。

## Existing Behavior

以当前代码为准：

| 位置 | 当前行为 |
| --- | --- |
| `internal/application/provider/catalog.go:36` | OpenAI 声明 `api_key` 与 `subscription_login` 两种认证方式，默认 Base URL 为 `https://api.openai.com/v1`。 |
| `internal/application/account/mutations.go:11` | API Key 账号可正常创建，密钥进受保护凭据存储，SQLite 只保存引用。 |
| `internal/application/routing/catalog.go:121` | OpenAI 账号只有 `codexLogin(a)`（订阅或网关托管 auth.json 且已认证）才具备模型路由资格；API Key 账号被排除。 |
| `internal/application/gateway/routing.go:230` | OpenAI 目标账号必须是 `AuthSubscription`／`AuthCodexFile` 且 `authenticated`，否则返回 `route_account_unavailable`。 |
| `internal/application/gateway/agent.go:34-64` | OpenAI 分支只调用订阅适配器 `chatgpt.Forward`，固定访问 `https://chatgpt.com/backend-api/codex/responses`；没有 API Key 适配器，兜底返回 `adapter_unavailable`。 |
| `internal/application/gateway/passthrough.go:123` | `/codex/v1` 透明转发的候选账号只选订阅登录账号。 |
| `internal/application/modelcatalog/sync.go:54` | OpenAI 只有订阅账号能同步模型；API Key 账号落到 `ErrUnavailable`。 |
| `internal/application/balance/service.go:70` | 余额查询只支持 DeepSeek、阶跃星辰、Kimi；OpenAI 返回 `unsupported`。 |
| `docs/product/QUOTA.md:21` | 普通 OpenAI API Key 已按“无官方额度／余额接口”处理，只展示本机终身 Token 用量。 |

结论：**当前在 UI 里添加 OpenAI API Key 账号不会让 Codex 用上它**——该账号没有模型、不能路由、不会被任何适配器转发。这是规划要补的核心缺口，不是配置问题。

## Expected Behavior

规划落地后：

1. 添加 OpenAI API Key 账号后，可同步该 Key 可见的模型，并在路由页选择；
2. Codex 通过 OmniProxy 发起的 Responses 请求可以使用该账号，按 API 计费消耗 40 美元额度；
3. 请求记录、Token 用量、错误分类和账号冷却按现有口径工作；
4. API Key 账号与 ChatGPT 订阅账号保持独立计费池，失败时不跨计费通道切换；
5. 额度区域明确显示“官方剩余额度不可查询”；如启用管理 API，只显示官方消费，不冒充余额；
6. 日志、数据库、前端和客户端配置中不出现完整 Key。

---

# Analysis

## 官方事实（核对于 2026-11-09）

### Codex 使用 API Key

- 官方认证文档明确：设置环境变量 `OPENAI_API_KEY` 即可按量计费使用 Codex；该 Key 至少需要 **Responses API 的写权限**。同时提供 `preferred_auth_method = "apikey"`，在同时存在 ChatGPT 授权时强制走 API Key。
  - [Codex Authentication](https://github.com/openai/codex/blob/fb5dfe3396d311d4873931a0a4ab26bf52936a9d/docs/authentication.md)
- 官方配置参考确认自定义提供商的字段：`model_providers.<id>.base_url`、`env_key`、`wire_api = "responses"`、`requires_openai_auth`；内置 `openai`、`ollama`、`lmstudio` 的提供商 ID 不可覆盖。配置参考另提供 `forced_login_method = "chatgpt | api"`。
  - [Codex Configuration Reference](https://learn.chatgpt.com/docs/config-file/config-reference)
- 因此“Codex 直接使用 API Key”本身是官方支持的，不需要 OmniProxy 也能完成；接入 OmniProxy 的价值是统一账号、路由、记账、故障切换和客户端管理。

### 剩余额度查询

- OpenAI **没有**公开受支持的“查询预付剩余额度”API。官方社区中 OpenAI Support 在 2026-09 明确表示：预付剩余额度与月度预算不同，会把“受支持的余额 API 和低余额提醒”转给产品团队，但没有时间表。
  - [Why is there no API for account balance?](https://community.openai.com/t/why-is-there-no-api-for-account-balance/577584)
- 旧版 `GET https://api.openai.com/v1/dashboard/billing/credit_grants` **只接受浏览器 Session Key**；用普通 `sk-...` API Key 调用会返回 403（`must be made with a session key`），不属于当前公开 API 参考。
  - 同上社区帖第 6–7 楼。
- 官方 Administration API 需要 **Admin API Key**（`sk-admin-...`），且 Admin Key 不能用于普通推理。它提供组织级的用量与费用查询（如 `/v1/organization/usage/completions`、`/v1/organization/costs`），能回答“花了多少”，**不能回答“还剩多少预付额度”**。
  - [Administration Overview](https://developers.openai.com/api/reference/administration/overview)
  - 端点与分桶实践参考（第三方实现，仅作字段核对）：[CodexBar OpenAI provider](https://github.com/steipete/CodexBar/blob/c61e01e774c449b06324a1cc260af7c77cf17d47/docs/openai.md)
- 结论：
  - **官方剩余余额接口：不存在（截至核对日）。**
  - 可选替代：本机按 Token 估算消费；或用户额外提供 Admin Key 读取官方消费／费用；真实剩余额度仍引导到 platform.openai.com 账单页。
  - 禁止使用 Session Key、抓取 dashboard、复用网页 Cookie 或未公开 RPC 来实现“余额”。

## 现有扩展点

| 模块 | 预计改动 |
| --- | --- |
| `internal/provider/openai/`（新增） | API Key 上游客户端：原生 Responses（JSON／SSE）、`GET /models`、错误分类、超时与大小限制。 |
| `internal/application/gateway/` | 目标资格、API Key 分支、用量观察、失败切换策略；保持订阅与按量两个池。 |
| `internal/application/routing/` | `eligible`、`codexLogin` 之外的 OpenAI API Key 资格；模型目录与别名解析。 |
| `internal/application/modelcatalog/` | OpenAI API Key 的 `GET /v1/models` 同步；已知模型的上下文／能力元数据只使用已核对值。 |
| `internal/application/agentconfig/` | Codex 的 API Key 接入模式（自定义提供商或固定前缀），备份／恢复与冲突检查。 |
| `internal/application/balance/`（可选） | 管理 API 消费查询与本机估算，独立于官方余额语义。 |
| `frontend` | 账号认证方式展示、Codex 接入模式选择、额度／消费说明；不新增伪造余额。 |
| `migrations/` | 若保存“初始额度／充值记录／Admin Key 引用”，新增迁移 046；仅 P1 不需要迁移。 |

## 方案比较

### 方案 A：不改代码，Codex 直连 OpenAI API

用户在 `%USERPROFILE%\.codex\config.toml` 设置：

```toml
preferred_auth_method = "apikey"
# 新版若只认 forced_login_method，则用：
# forced_login_method = "api"
```

并设置用户环境变量：

```powershell
setx OPENAI_API_KEY "sk-..."
```

新开终端后 `codex`，用 `/status` 确认认证方式为 API Key。

- 优点：5 分钟可用，官方支持，直接消耗 40 美元额度。
- 缺点：绕过 OmniProxy，没有统一记账、路由、故障切换和多账号；每个客户端各自配置环境变量。
- 适用：先验证 Key 权限和额度是否可用，不作为最终产品方案。

### 方案 B：OmniProxy 原生支持 OpenAI API Key（推荐）

新增按量上游适配器，并让 Codex 通过本地网关调用。Codex 侧有两种接法：

- **B1：扩展 `/codex/v1` 透明转发**，让 API Key 账号进入同一入口。
  - 优点：保持现有 Codex 一键配置（只写 `openai_base_url`，保留 ChatGPT 登录身份），用户无感。
  - 风险：透明转发当前固定上游 `chatgpt.com/backend-api/codex`，且透传 `OpenAI-Beta`、`ChatGPT-Account-ID`、压缩标记等订阅协议；API 通道的路径、头部、压缩和 WebSocket 能力不同，需要逐项验证，失败时容易把协议差异误报成账号故障。
- **B2：新增独立的 Codex“API Key 模式”**，写自定义提供商指向网关。
  - 典型配置：`model_provider = "omniproxy-openai"`，`base_url = "http://127.0.0.1:<port>/v1"`，`wire_api = "responses"`，`requires_openai_auth = false`，用固定前缀或接入 Key 认证。
  - 优点：协议边界清晰，复用现有 `/v1/responses` 适配器、记账、路由和故障切换；不触碰订阅透明转发的语义。
  - 缺点：Codex 的 `model_provider` 身份会改变；需要备份／恢复和冲突检查，已有会话不自动切换。

**推荐：P1 先做 B2**，把“API Key 上游 + 普通 `/v1` 网关”打通；**B1 作为 P3 可选增强**，在真实协议验证通过后再做。这样先把“能用、能记账、能切号”做对，再追求“Codex 完全无感”。

## Risks

- **协议差异**：Codex 在 ChatGPT 登录模式下可能发送订阅专有字段、压缩标记或 WebSocket；API 通道不一定接受。B2 用自定义提供商可先规避；B1 必须用真实 Codex 做契约测试。
- **模型元数据**：`GET /v1/models` 只返回 ID、创建时间、归属等，不返回上下文窗口、工具／图片能力；不能猜，未知就保持未知，客户端目录只声明已核对能力。
- **额度语义**：40 美元是预付额度，可能因过期、组织／项目限制、Key 权限或模型权限不可用；不能把“添加成功”当作“额度可用”。
- **计费池隔离**：同一 OpenAI 提供商下同时存在订阅与按量账号时，若允许互相故障切换，会把两种计费混在一起；必须按 `AuthMethod` 隔离（现有 `SameAccountPool` 已具备该基础）。
- **额度展示**：没有官方余额接口，任何“剩余 $X”都只能是本机估算；必须标注 `≈`、说明不含其他客户端消费，不能用于调度或“余额不足”判断。
- **测试成本**：真实请求会消耗用户的 40 美元额度；验收应使用最小请求并事先说明。

---

# Implementation Plan

## P0：先用官方方式验证（不写代码）

1. 在 platform.openai.com 确认 Key 权限至少包含 Responses API 写权限，确认 40 美元额度属于当前组织／项目。
2. 按方案 A 配置 Codex，`/status` 确认走 API Key，发一条最小请求确认额度扣减。
3. 记录实际可用的模型 ID、错误表现和计费页剩余额度，作为后续验收基线。

P0 成功只证明 Key 可用，不代表 OmniProxy 已接入。

## P1：OpenAI API Key 上游与 Codex 接入

1. **新增 `internal/provider/openai`**
   - `Forward`：`POST {baseURL}/responses`，Bearer 认证；保留 JSON／SSE，按现有 `proxy.Response` 消费；禁止重定向，设置响应头上限、请求期限和响应大小上限。
   - `DiscoverModels`：`GET {baseURL}/models`，有界解析 ID；不把展示名当 ID。
   - 错误分类：`invalid_api_key`／401／403 → 认证失败；`insufficient_quota`／429 → 明确额度不足（不按普通限流重试）；`rate_limit_exceeded` → 可重试限流；5xx → 上游不可用。
   - 不在适配器内强制改写 `store`、`stream`、工具或推理字段；是否强制 `store=false` 需单独评审（涉及上游留存与隐私说明）。
2. **网关资格与转发**
   - `routing.eligible` 与 `gateway.selectRoutedTarget` 允许 `kind=openai && AuthMethod=api_key`；不要求订阅 `auth_state`。
   - `gateway.forwardAgent` 增加 API Key 分支，调用新适配器；Chat／Messages 首版可返回明确 `unsupported_protocol`，或复用现有跨协议转换（二选一，实施前确认）。
   - 保留 `SameAccountPool` 的按认证方式隔离：订阅池与按量池不互相切换。
   - 接入 `proxy.ObserveResponse` 记账；核对 Responses `usage` 的输入、输出、缓存字段，不重复计缓存。
3. **模型目录**
   - `modelcatalog.Sync` 增加 OpenAI API Key 分支；同步失败保留旧快照，换 Key 失效旧目录。
   - 已知模型的上下文／能力使用已核对元数据；`/models` 未声明的字段保持未知，不生成 1M 或图片能力。
   - 路由页、Codex 目录和客户端目录保持同一模型 ID。
4. **Codex 接入（B2）**
   - 在 Codex 接入详情增加“认证方式：ChatGPT 订阅 / OpenAI API Key”选择；默认保持现有订阅行为。
   - API Key 模式写入独立自定义提供商（不复用内置 `openai` ID），`wire_api="responses"`、`requires_openai_auth=false`；凭据使用现有受保护接入 Key 或固定前缀，不把上游 Key 写入 Codex 配置。
   - 复用现有预览、加密备份、恢复、外部修改拒绝和模型目录逻辑；切换模式前明确提示会改变 `model_provider`。
   - 当前配置文件里的 `model_provider`、`profile`、`model_providers.<id>.auth` 冲突继续按既有规则拒绝，不擅自覆盖。
5. **UI 与文档**
   - 账号页 OpenAI 同时展示“订阅”和“API Key（按量）”两种账号；API Key 账号不显示订阅额度窗口。
   - 额度页对 OpenAI API Key 显示“官方剩余额度不可查询”，保留本机终身 Token；提供 platform.openai.com 账单入口。
   - 更新 `docs/product/SUBSCRIPTION.md`、`LOCAL_AGENTS.md`、`QUOTA.md`、`GATEWAY.md`；需要时新增 ADR（API Key 计费通道与 Codex 身份切换）。
6. **测试**
   - 适配器契约：JSON／SSE、usage、401／403／429 `insufficient_quota`／5xx、超时、重定向、超大响应、无 usage。
   - 路由：API Key 账号可选中；订阅池与按量池不互相切换；固定账号仍优先。
   - 目录：同步成功／失败／换 Key 失效；未知元数据不伪造。
   - Codex 配置：预览、应用、备份、恢复、外部修改冲突；不写入上游 Key。
   - 真实验收：用户同意后用最小请求验证一次，记录请求明细与官方消费页对照。

## P2：额度／消费展示（可选）

1. **明确官方边界**：不实现“官方剩余余额”；额度卡片显示“官方未提供查询接口”，链接到 platform.openai.com。
2. **本机估算（默认）**
   - 以用户填写的初始额度（如 40 USD）和充值记录为基线，按本机请求的实际模型与 `internal/core/tokenprice` 参考价累计；
   - 显示 `≈ 已用` 与 `≈ 估算剩余`，注明不含其他客户端、缓存未知时按区间、不是官方余额；
   - 估算只用于展示，不参与调度、冷却或“余额不足”判断。
3. **管理 API（可选，需用户额外提供 Admin Key）**
   - 独立凭据类型，不与推理 Key 混用；Admin Key 不能推理。
   - 只读 `GET /v1/organization/usage/completions` 与 `GET /v1/organization/costs`，按日分桶，支持项目范围；
   - 展示“官方消费”而不是“剩余额度”；权限不足时明确提示需要组织 Admin Key，不回退到 Session Key 或网页抓取。
4. 不实现自动充值、余额不足自动停用或按余额调度。

## P3：`/codex/v1` 原生支持（可选）

- 为透明转发增加按账号选择上游基址与认证方式：订阅账号走 `chatgpt.com/backend-api/codex`，API Key 账号走 `api.openai.com/v1`。
- 逐项验证路径、头部、`store`、压缩、WebSocket、错误分类和用量观察；无法对齐的端点返回明确错误，不伪装成功。
- 只有 P3 通过真实 Codex 契约测试后，才把 API Key 账号放入 `/codex/v1` 候选池；否则保持 B2。

---

# 验收标准

1. 添加 OpenAI API Key 账号后可同步模型，路由页可选，Codex 经网关完成一次真实 Responses 请求。
2. 请求明细记录模型、账号、Token 和结果；401／403、429 额度不足、5xx 有稳定错误码，不把额度不足当普通限流。
3. 订阅账号与 API Key 账号不跨计费池故障切换；固定账号仍优先。
4. 额度区域不出现伪造的官方余额；本机估算有 `≈` 和边界说明；无 Admin Key 时不请求管理接口。
5. Key 不出现在日志、SQLite 明文、前端返回值或 Codex 配置中；沿用 DPAPI 与中央 Redactor。
6. `task verify` 通过；真实请求验收前明确告知会消耗额度并取得用户同意。

---

# 待确认

1. Codex 接入采用 **B2 自定义提供商**（推荐，协议边界清晰），还是必须保持内置 `openai` 身份、走 B1 `/codex/v1`？
2. 40 美元额度所在组织是否可创建 **Admin API Key**；若不能，P2 只做本机估算＋官方入口。
3. 首版协议范围：只支持 Responses（Codex 主路径），还是同时支持 Chat／Messages 跨协议？
4. 是否接受 API Key 模式改变 Codex 的 `model_provider`（会退出内置 openai 身份，可用加密备份恢复）？
5. 是否允许用用户的真实 Key 做一次最小计费验收（会产生少量费用）。

---

# Progress

- 2026-11-09：完成官方 Codex 认证、余额接口与当前代码路径核对；形成 P0–P3 计划。未修改代码。
- 2026-11-09：用户确认 Codex 身份不变、余额暂不做。转入 B1 实施：`/codex/v1` 透明转发按账号选择 OpenAI API Key 上游，同时补齐普通 `/v1` 的 API Key 转发、模型目录与路由资格。

# Implementation Result

## Changes

- 新增 `internal/provider/openai`：原生 Responses／`/responses/compact` 转发、`GET /v1/models` 目录发现、Bearer 认证、非对话型号过滤、固定端点与重定向拒绝。
- `/codex/v1` 透明转发：按目标账号切换上游基址与凭据；API Key 账号使用 `api.openai.com/v1` + `Authorization: Bearer`，不发送 `ChatGPT-Account-ID`；订阅账号保持原 `chatgpt.com/backend-api/codex` 路径。无模型握手的兜底取号优先订阅池，没有订阅账号时使用 API Key 池。
- 普通 `/v1`：新增 OpenAI API Key 分支，Responses 原样转发，Chat／Messages 经既有协议转换后回包；`/responses/compact` 同样走 API Key 端点。
- 路由与目录：OpenAI API Key 账号可参与模型路由；`modelcatalog.Sync` 通过 `/v1/models` 建立账号目录，未知能力保持为空；前端模型同步下拉框允许 OpenAI API Key 账号。
- 计费隔离：订阅与 API Key 仍按 `AuthMethod` 分池，不跨池故障切换；订阅额度观察只对订阅账号生效。
- 额度错误：透明转发识别 OpenAI `insufficient_quota` 为执行前额度拒绝，按受控 429 收尾。
- 文档：`LOCAL_AGENTS.md` 新增按量账号说明，`GATEWAY.md` 更新手动接入与透明转发边界。

## Files Changed

- `internal/provider/openai/`（新增）
- `internal/application/gateway/{agent,compaction,key_access,passthrough,passthrough_token,routing,service,subscription_quota}.go`
- `internal/application/modelcatalog/{service,sync}.go`
- `internal/application/routing/catalog.go`
- `internal/core/proxy/quota_rejection.go`
- `frontend/src/features/routing/model-catalog.tsx`
- `docs/product/{LOCAL_AGENTS,GATEWAY}.md`
- 测试：`internal/provider/openai/client_test.go`、`internal/application/gateway/openai_apikey_test.go`、`internal/application/modelcatalog/openai_apikey_test.go`

## Technical Notes

- Codex 客户端配置未改动：继续使用内置 `openai`、原官方登录和 `openai_base_url`，因此身份、会话与一键配置流程不变。
- API Key 账号的模型目录只包含 `/v1/models` 中可进入对话／Codex 的型号；上下文窗口、图片、工具档位未由该接口公布，保持未知。
- API Key 与订阅账号的计费、冷却和凭据不混用；固定账号仍优先。
- 未实现余额查询；官方不存在可用的剩余预付额度接口。管理 API 消费查询与本机估算留待后续。

# Verification

- [x] 核对 Codex 官方认证文档与配置参考
- [x] 核对 OpenAI 余额／管理 API 公开边界
- [x] 核对当前 main 的 OpenAI 账号、路由、适配器、目录、余额代码路径
- [x] 新增 OpenAI API Key 适配器、`/codex/v1` 与 `/v1` 转发、目录同步、路由资格
- [x] `go test . ./internal/... -count=1` 全部通过
- [x] `go vet . ./internal/...`、`staticcheck . ./internal/...` 通过
- [x] `pnpm typecheck`、Prettier 检查通过
- [x] 前端完整测试 146 文件／891 项通过（`vitest run --testTimeout=15000`；默认 5 秒超时在并行负载下有 4 项异步超时，单独重跑均通过）
- [x] `task build` 生产构建 exit 0（Wails v3.1.0 bindings、Vite production、Windows amd64）
- [ ] 真实 Key 验证（按用户要求留待用户测试）

## Verification Result

`task verify` 默认运行已完成格式、lint、typecheck 与 Go 测试，但在默认并行度下前端测试出现 4 项 5 秒异步超时并在全量运行中长时间不结束；终止后按同一质量门逐项复验：

```text
gofmt -l main.go internal          PASS (empty)
go vet . ./internal/...            PASS
staticcheck . ./internal/...       PASS
pnpm typecheck                     PASS
go test . ./internal/...           PASS
vitest run --testTimeout=15000     146 files / 891 tests PASS
task build                         exit 0
```

4 个超时文件单独重跑均通过（36/36），未发现与本改动相关的断言失败。

# Documentation

已更新：

- `docs/product/LOCAL_AGENTS.md`
- `docs/product/GATEWAY.md`
- `docs/product/SUBSCRIPTION.md`
- `docs/plans/CURRENT.md`

不涉及 PRD、架构、技术栈、Coding Rules、Testing、Build 或 ADR；本实现复用 ADR-023 透明转发与 ADR-024 网关持有登录边界，未新增架构决策。

# Deviations

- 原计划推荐 B2 自定义提供商；用户明确要求 Codex 身份不变，改为 B1：扩展 `/codex/v1` 透明转发，按账号切换 API Key 上游。
- 原计划 P2 余额／消费展示本轮不做，按用户要求留待后续。
- 普通 `/v1` 同时支持 Responses 与 Chat／Messages 转换，超出 Codex 单路径的最小范围，但复用既有转换，未新增依赖。

# Git

待提交。

# Result

已实现 OpenAI 按量 API Key 作为可路由上游：账号可同步模型，Codex 保持内置 `openai` 身份通过 `/codex/v1` 使用 API Key 额度，普通 `/v1` 也可调用；订阅与按量分池。余额查询未实现，真实计费与模型权限待用户实机验证。
