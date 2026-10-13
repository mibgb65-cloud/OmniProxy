# TASK-20261014-002: Gemini API、用量、限流和额度接入规划

## Status

Completed

## Created

2026-10-14（Asia/Singapore）

## Request

用户要求参考 https://aistudio.google.com/docs/api-key，先规划 Gemini API 额度等能力接入；随后明确范围为“API 调用 + 用量、限流和额度”。

## Goal

形成基于 Google 官方文档及当前 main 实现的分阶段接入计划，区分 API Key 可完成的推理能力、项目限流与需要额外授权的官方资源查询。

## Scope

官方认证、模型及协议、用量与限流、配额／费用查询边界；现有扩展点、建议方案、验收条件和待确认事项。

## Out of Scope

本轮不实现代码、迁移或界面，不读取真实 Key、登录 Google Cloud、调用真实推理、修改账号／云项目／客户端配置。

# Analysis

基线：main `c6ff233`，已经包含 Atria 与 GPT 长上下文。用户给出的 AI Studio 页面为动态界面，公开检索正文有限；已进一步核对 Google 官方开发文档及 Cloud REST 参考，未登录或读取私人项目。

## 官方能力和边界（核实于 2026-10-14）

| 能力 | 已确认事实与规划含义 |
| --- | --- |
| API Key | 每个 Key 关联 Cloud 项目；新 AI Studio Key 默认为绑定服务账号的 auth key，旧 standard key 的限制状态影响可用性。沿用 `api_key` 凭证保存，不按外观猜 Key 类型，不把绑定身份等同于 Cloud 管理授权。[Key 文档](https://ai.google.dev/gemini-api/docs/api-key) |
| 模型发现 | `GET https://generativelanguage.googleapis.com/v1beta/models` 支持分页，返回真实 ID、名称、输入／输出 Token 上限和生成方法；卡片中的人类可读模型名称不能当作请求 ID。[Models REST](https://ai.google.dev/api/models) |
| 兼容调用 | Google 提供 `/v1beta/openai/chat/completions` 和 OpenAI 风格模型目录，使用 Bearer Key，支持文本、工具和 SSE；未据此确认原生 OpenAI Responses 或 Anthropic Messages。[兼容接口](https://ai.google.dev/gemini-api/docs/openai) |
| 原生调用 | Interactions 已 GA，是官方新项目推荐路径；`generateContent` 保持支持。Interactions 默认保存请求，网关方案应显式 `store=false`，不能盲目映射跨账号续接 ID。[Interactions](https://ai.google.dev/gemini-api/docs/interactions-overview) |
| 请求额度 | RPM、输入 TPM、RPD 等限制按项目及模型／指标共享，同项目多 Key 不增加额度；RPD 在太平洋时间午夜重置，具体限制以项目当前结果为准，不硬编码免费档表。[限流](https://ai.google.dev/gemini-api/docs/rate-limits) |
| 计费资源 | 预付余额、项目消费上限、计费账号消费上限是不同资源；预付余额耗尽可导致关联项目同时停用。余额与交易管理目前在 AI Studio Billing，费用数据有延迟，不能用估算消费计算真实余额。[Billing](https://ai.google.dev/gemini-api/docs/billing) |
| 官方配额读取 | Service Usage 可读取配额指标／限额，要求 OAuth scope 和 `serviceusage.quotas.get`；该接口不是当前消耗计数器。[consumerQuotaMetrics.list](https://docs.cloud.google.com/service-usage/docs/reference/rest/v1beta1/services.consumerQuotaMetrics/list) |
| 官方用量读取 | Monitoring `projects.timeSeries.list` 要求 Cloud／Monitoring OAuth scopes；官方指标表存在 Gemini 项目用量／配额指标，但模型、方法、档位、窗口覆盖及采样延迟需对真实项目验证。[timeSeries.list](https://docs.cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.timeSeries/list)、[指标表](https://docs.cloud.google.com/monitoring/api/metrics_gcp_d_h#generativelanguage) |
| 费用读取 | Cloud Billing 导出到 BigQuery 可分析费用，但需要先启用导出、授权，存在延迟和查询／存储费用；不作为首版强制依赖，也不能由此取得 AI Studio 预付余额。[费用导出](https://docs.cloud.google.com/billing/docs/how-to/export-data-bigquery) |
| 工具续接 | Gemini 思考步骤／签名属于模型上下文；跨协议转换必须保留关联语义，不能把它伪装成 Anthropic 签名或静默丢弃。[Thinking](https://ai.google.dev/gemini-api/docs/thinking)、[Function calling](https://ai.google.dev/gemini-api/docs/function-calling) |
| 错误分类 | 401／403 是凭证／权限问题，402 `payment_required` 是支付资源不足；429 分短期限流与日配额耗尽，不能统一显示余额为零。[API errors](https://ai.google.dev/gemini-api/docs/api-errors) |

**结论**：当前公开文档未确认“仅传入 Gemini 推理 Key，即可读取完整实时剩余额度、档位、预付余额”的统一接口。采用可分阶段交付的方案：Key 完成调用和本机观察；额外只读 Cloud 授权完成经过验证的项目指标查询；官方不提供或未验证的数据明确未知。

## 现有实现与扩展点

| 模块 | 当前事实／预计影响 |
| --- | --- |
| `application/provider/catalog.go`、`application/account/` | 已有 `builtin_google`／`kind=google`、原生 v1beta Base URL 和 API Key 账号／DPAPI；不用新增第十一家卡片或新增推理 OAuth。 |
| `provider/`、`application/gateway/{service,agent,routing}.go` | 尚无 Google 实际适配器，目标资格限定 DeepSeek／订阅 OpenAI／Atria；新增明确 Google 分支，禁止落入其他提供商。 |
| `application/modelcatalog/`、`application/routing/` | 添加 Google 官方账号目录、账号版本失效和 Google 路由资格；复用现有同步失败保留、别名及优先级，不把展示预设当作已验证访问权限。 |
| `core/modelcatalog`、`core/agentprotocol` | 当前只有通用 `ContextWindow`，应区分 Google 输入与输出上限；现有转换结果仅保存调用 ID／名字／参数／类型，会损失供应商扩展与签名。 |
| `core/proxy/usage.go`、`application/activity/` | 支持 Chat／Responses／Messages 的上游 usage 及累计统计；可复用兼容 Chat，原生 Interactions 须新增解析并核对输入、输出、思考及缓存口径。 |
| `frontend/features/quota` | 资源类型只有 subscription／balance／unsupported，Google 当前未支持；新增项目限额／用量类型，不能套 Codex 5h／周百分比或 DeepSeek 金额 DTO。 |
| `frontend/features/routing`、`gateway/use-gateway-preparation.ts` | 提供商选项、同步筛选和接入检查存在白名单，需要随实际可转发能力扩展。 |
| `transport/wails`、`frontend/services` | 新增项目绑定、Cloud 连接及快照查询绑定，前后端共享缓存和公开 DTO；现有账号接口只读元数据，不额外暴露 Key。 |

## 方案比较与推荐

1. 只增加额度卡片：缺少转发和可靠数据源，不算完整接入。
2. 只用 API Key 加静态免费档表：能完成调用，不能得出项目真实剩余，也会把共享额度重复计算，不推荐。
3. **推荐分层实现**：首个闭环用官方 Chat 兼容接口接入现有网关；单独设计项目级资源和只读 Cloud 查询；随后按官方推荐 Interactions 解决完整 Agent 工具／续接／流式桥接。保留原生 v1beta 配置根路径，不把既有用户 Base URL 整体改成 `/openai`。

不增加大型 SDK、不更换技术栈。原生 Gemini CLI、Vertex AI、Gemini 消费订阅、图片／音频／视频、Embedding、Live／Batch／Deep Research 不作为首个文本闭环的默认范围。

# Implementation Plan

1. 核对 API Key、项目限流、计费与管理 API 授权能力。
2. 对照现有接口、目录、用量观察和配额 UI，比较方案并明确最小阶段。
3. 写入接口与数据模型建议、实施步骤、验证标准、权限及未知边界。
4. 检查文档和链接，记录规划结果并提交、归档。

## 后续实施阶段（本轮不执行）

### P1：API 调用、真实模型和本机用量

- 新增 `internal/provider/google/`，复用 `account.UseAPIKey`；原生模型 GET 使用 `x-goog-api-key`，兼容 Chat 使用 Bearer；禁止 Key 出现在查询字符串、URL、日志、客户端目录。
- 从现有 `/v1beta` 根拼接 `/models` 与 `/openai/chat/completions`，精确校验路径／模型 ID、JSON 边界、输出限制、超时／取消、认证头及重定向；仅接受用户明确配置的上游地址，不允许正文指定地址。
- 模型发现有界分页，保存当前账号授权版本；仅公布适用的文本生成模型。按官方字段区分输入／输出上限，不相加、不盲目承诺 1M 变体；未知模态／工具／思考档位继续未知。
- Chat JSON／SSE 原生透传并观察最终 usage；核对兼容接口最终 usage 获取条件。保留思考／缓存分项，不重复加进上游总数，不从字符数推算官方计费用量。
- 接入真实路由、公开目录、接入检查、请求详情与用量页。P1 只声明已验证的 Chat 协议，不预先让 Codex／Claude 工具模型成为可选完整 Agent 目标。
- Key 添加／模型刷新不偷偷发送推理来“激活额度”；真实调用验收使用用户已保存 Key、最少短文本和极短回复，事先明确会消耗额度。

### P2：项目绑定、限流和资源卡片

- 增加独立的 Google 项目绑定：账号 ID → 项目 ID／项目号、绑定来源与验证状态。不要复用订阅工作区的 `auth_account_id`，也不从 Key 字符串推断项目或用 Key 明文做分组键。
- 未绑定的 Key 显示项目未知，不能默认当作独立额度池；用户填写项目只算待验证映射。更换 Key 清除／重验绑定，修改账号名字不使项目快照失效。
- 限流身份使用项目＋模型＋指标／窗口；项目全部 Key 共享 RPM／TPM／RPD。已有路由优先级不变，首版传播错误和经过校验的 Retry-After，不新增无声重试／同项目换 Key 绕过。
- 解析公开错误结构中的限额／维度／RetryInfo（若存在），只保存校验后的分类和值；兼容／原生错误格式分别覆盖。不记录原始错误正文，不因一次 429 判定所有资源耗尽。
- RPD 重置使用 `America/Los_Angeles` 处理夏令时；只对已确认每日指标给重置时刻，计时结束标记待刷新而不是自动补满。
- 配额页新增项目资源卡片，同项目多账号关联展示但汇总只计算一次；分别显示本机用量、官方已用／限额、未知项、短期限流和支付状态。消费上限与预付状态按计费账号或项目实际范围分组，不能每 Key 重复相加。
- 无额外 Cloud 授权时，本机请求／Token 和已观察错误照常可用，官方完整用量／剩余／预付金额为未知，提供 AI Studio Usage／Billing 入口。

### P3：官方项目配额读取（可选只读授权）

- 先做无写操作的接口验证：真实项目的 `consumerQuotaMetrics`、Monitoring 指标是否覆盖该 Key 类型、当前模型、Chat／Interactions 调用及免费／付费档位；验证数据存在性、维度和延迟后再冻结 DTO。
- 设计独立 Google Cloud 只读连接，推理账号无需因此改成 OAuth。推荐桌面 OAuth＋PKCE／受控 loopback 回调；OAuth 客户端配置、scope 和 IAM 权限需实施前确认。现有 Codex 登录服务不能直接冒充 Google OAuth。
- 最小授权以 Service Usage 配额读取和 Monitoring 读取为目标；不要请求 Editor／付款写入权限，不自动启用 Cloud API、链接计费账号、调高配额或开 BigQuery 导出。无法授权时 API 调用仍可用。
- Service Usage 提供限额，Monitoring 提供用量；严格对齐项目、模型、地域、指标、单位、时间窗口和档位。GAUGE／DELTA／累计值分别处理，分钟限流数据有延迟时不展示“实时剩余百分比”；只对语义一致的数据计算带时间戳的剩余。
- Auth key 不进入普通服务账号使用指标的文档说明需纳入验证，不把通用 IAM 计数器当作 Gemini 全量调用；`generate_content_*` 命名的指标也不能直接假定包含 Interactions。
- Cloud OAuth／刷新令牌走现有受保护凭据存储，DB 仅保存引用；快照可持久化公开限额／计数／来源／有效时间，授权版本、项目或计费绑定变化使缓存失效。初版只手动刷新、有界串行查询，失败保留旧快照并标记过期。

### P4：完整 Agent 协议和费用扩展

- 增加 Interactions 文本／工具／SSE 适配，显式 `store=false`；核对 `function_call`、`function_result`、thought steps／签名和终态 usage。原生与兼容路径共用项目资源范围，不能彼此重复记账。
- 对 Responses／Messages 做增量桥接并保留工具历史／不透明签名；先证明完整两轮调用，再开放客户端目录和一键配置。若需要有界签名状态缓存，应写 ADR 明确隔离、TTL、容量、删除和当前无持久化语义的变化；做不到时明确不支持，不能静默丢字段。
- 模型原生输入／输出上限与客户端有效窗口分别建模，使用短输入及代表性长输入实测后再启用客户端大窗口，保留 GPT／DeepSeek 现有行为和原会话身份。
- 真实预付余额查询接口仍为待确认；没有公开受支持接口时保留 AI Studio Billing 入口，不抓私人网页 Cookie／未公开 RPC，不用本地估算费用填余额。
- 可选费用导出只读取用户已配置的 BigQuery 数据，限制查询日期／字段／扫描量／超时并显示数据延迟与币种；存储／查询可能收费，另行明确启用。本机计费估算仅标“估算”，版本化价格，不作真实余额或额度判断依据。

## 建议数据模型（待接口验证后冻结）

- `GoogleProjectBinding`：账号、项目 ID／号、来源、验证状态、Key 版本；多个 Key 指向同一个项目实体。
- `GoogleCloudConnection`：公开授权主体、项目范围、读取能力和 `credential_ref`，不混入推理 Key 类型。
- `GoogleQuotaSnapshot`：项目／模型／指标／单位、窗口起止和重置区、limit／used／remaining 可空、来源、observedAt／fetchedAt、数据延迟与授权版本。
- `GoogleBillingStatus`：计费账号或项目范围、已确认的支付状态／消费上限；预付金额未知时为空，不自动填零。
- 迁移只追加，编号实施时重新核对当前最高 019；不要占用空缺 018 或改写已有迁移。独立外键表优先，公开字段进入 Wails DTO 后重新生成绑定。
- 配额缓存以项目＋指标维度＋管理授权版本为键，账号详情和配额总览共享；本机观察单独标来源，外部调用无法据此补全。刷新频率／保留时间在真实延迟测试后确定，不能直接套现有余额 60 秒缓存。

## 最小充分验收

1. 升级保持 Google 内置 ID、已有账号、开关与凭据引用；目录分页／大小写／输出限制／无效模型和 Key 轮换覆盖。
2. Chat 非流式、真正增量 SSE、最终 usage、无 usage、截断流、取消、重定向、413、401／403／402／429／5xx 均有契约测试；统计不重复或假零。
3. 两个 Key 同项目只出现一个额度池；两个项目分离；项目／Key／Cloud 授权／计费绑定变更失效，未知绑定不冒充独立额度。
4. 太平洋时间夏令时转换、RPD 与 RPM 不混淆、RetryInfo 缺失、单位／指标不匹配、旧快照、权限撤销和查询队列上限覆盖。
5. 官方数据与 AI Studio／Cloud 同一时间窗人工对照，明确观测延迟与外部流量；无指标和缺权限保持未知，不显示 100% 或 0。
6. Agent 阶段至少完整工具调用 → 工具结果 → 后续回复，覆盖多工具／并行／思考签名／流式；真实 Codex 和 Claude 客户端验收后才声明支持。
7. 日志／URL／目录／快照中无 Key、令牌或签名正文，管理 API 只读；回归已有 OpenAI、DeepSeek、Atria 和 GPT 长上下文，再跑 `task verify`。

## 实施前需要确认

- 真实 Google 项目与 Key 的关联、档位和实际启用的模型；请通过应用保存凭据，不在聊天中提供 Key。
- 是否启用 P3 的 Cloud 只读授权，以及可用 OAuth 客户端／项目 IAM 权限；这影响官方项目额度可查询程度，不影响 P1。
- P4 是否要求首批完成 Codex／Claude 工具任务、是否接受未来增量桥接／受控签名缓存设计；OAuth／缓存新边界需 ADR。
- 费用监控是否只提供官方入口，还是读取已经配置的 Billing Export；首版不新建付费资源、不修改消费上限。

# Implementation Result

已完成官方资料核对、main 扩展点检查、推荐方案、四阶段范围、数据模型与验收条件。仅生成规划文档，未实施 API、迁移、OAuth 或界面。

# Verification

- 已打开用户提供页面，并用 Google 官方开发文档与 Cloud REST 参考核对认证、协议、项目限流、预付／消费上限、管理 API scope 和错误语义。
- 已核对 main 的 Google 预设、受保护账号凭据、实际网关白名单、目录、用量观察和配额页类型／缓存；没有沿用旧 Atria 规划工作区。
- 仅文档变更，不运行 `task verify` 或推理；13 个官方链接／引用已人工核对，UTF-8 无 BOM 检查和 `git diff --check` PASS。

# Git

`efb4972` docs(plan): 规划 Gemini API 与项目额度接入。

# Result

规划已完成。推荐先交付 P1／P2，再根据只读 Cloud 授权能力完成 P3；完整 Agent 桥接与费用扩展为 P4。所有未知数据与权限边界已明确，接入代码尚未实施。

# Completed

2026-10-14（Asia/Singapore）；完成状态仅指本轮规划。
