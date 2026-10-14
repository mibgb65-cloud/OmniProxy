# TASK-20261014-015: 阶跃星辰国内／海外 API 与 Step Plan 接入规划

## Status

In Progress

## Created

2026-10-14（Asia/Singapore）

## Request

用户希望接入阶跃星辰 API Key，先看看如何安排，并询问是否分国内、海外平台；随后明确普通按量 API Key 和 Step Plan 订阅两种都需要。

## Goal

基于官方文档和现有 main，形成国内／海外 × 按量 API／Step Plan 的接入方案、实施顺序和验收条件。

## Scope

平台与端点、密钥和计费边界、模型／协议／工具调用、账号级配置、路由及用量／配额查询、兼容性与验证方案。

## Out of Scope

本轮为接入规划，不实施适配器、迁移或界面，不读取真实 Key、不调用付费推理、不修改用户账号或客户端配置。

# Analysis

已核对 main `ce64997` 的账号、网关、模型目录、路由、用量及配额实现，并阅读产品、架构、技术栈、编码、Git 和测试规范。

## 官方事实（2026-10-14 核实）

国内与海外是独立平台，Key 必须匹配签发平台，不能凭 Key 外观猜地区。[官方模型仓库](https://github.com/stepfun-ai/Step-3.7-Flash)

| 平台 | 服务 | OpenAI 风格 API Base URL | 计费资源 |
| --- | --- | --- | --- |
| 国内 `.com` | 普通按量 API | `https://api.stepfun.com/v1` | 按量账户资源 |
| 海外 `.ai` | 普通按量 API | `https://api.stepfun.ai/v1` | 按量账户资源 |
| 国内 `.com` | Step Plan | `https://api.stepfun.com/step_plan/v1` | Step Plan Credits |
| 海外 `.ai` | Step Plan | `https://api.stepfun.ai/step_plan/v1` | Step Plan Credits |

普通入口见[国内 Chat API](https://platform.stepfun.com/docs/zh/api-reference/chat/chat-completion-create)和[海外平台](https://platform.stepfun.ai/)；订阅入口见[国内接入文档](https://platform.stepfun.com/docs/zh/step-plan/integrations/reasoning-api)和[海外接入文档](https://platform.stepfun.ai/docs/en/step-plan/integrations/reasoning-api)。两种服务均使用 Step API Key；选择订阅不等于新增 OAuth 登录，账号需实际拥有相应套餐及模型权限。

### 协议与模型

- 普通及 Step Plan 均有原生 Chat 和 Messages。完整 Messages 路径分别为 `/v1/messages`、`/step_plan/v1/messages`；Anthropic SDK 配置根地址分别是主机根、`/step_plan`，SDK 自己追加 `/v1/messages`。[国内 Messages](https://platform.stepfun.com/docs/zh/api-reference/chat/messages-create)、[海外 Messages](https://platform.stepfun.ai/docs/en/api-reference/chat/messages-create)
- 普通 Responses 路径为 `/v1/responses`；Step Plan 的 `/step_plan/v1/responses` 已在国内、海外 Codex 接入文档确认。[Responses](https://platform.stepfun.ai/docs/en/api-reference/responses/responses-create)、[国内 Codex 接入](https://platform.stepfun.com/docs/zh/guides/developer/integrations/codex)、[海外 Codex 接入](https://platform.stepfun.ai/docs/en/guides/developer/integrations/codex)
- 首批文本候选为 `step-5-preview`、`step-3.7-flash`、`step-3.5-flash`、`step-3.5-flash-2603`。Responses 参考明确列出前两款；集成文档出现其他模型不代表每种协议都已确认。按“地区＋服务＋模型＋协议”开放能力，不能用一份全局模型白名单。[Step Plan 模型](https://platform.stepfun.com/docs/zh/step-plan/integrations/reasoning-api)
- 国内 `step-router-v1` 仅支持 Step Plan，会在 DeepSeek 和 Step 模型间由上游选择；Messages 的图片／文档／内置 web_search 不支持，`output_config.effort` 被忽略。它不等于 OmniProxy 自身路由，也不能据底层最大值声明所有请求具有 1M 窗口。海外目录及各协议能力另行验证。[Router](https://platform.stepfun.com/docs/zh/guides/models/step-router)、[Messages 字段限制](https://platform.stepfun.com/docs/zh/api-reference/chat/messages-create)
- `GET /v1/models` 为普通模型发现，模型对象仅含 ID／创建时间／所属组织等，不返回完整能力或上下文。未确认 `/step_plan/v1/models`，不得机械追加路径，也不能把普通目录当作订阅授权证明。[国内模型列表](https://platform.stepfun.com/docs/zh/api-reference/models/list)、[模型对象](https://platform.stepfun.com/docs/zh/api-reference/models/object)
- Step 5 文档标称 1M 上下文／输入、64k 输出；Step 3.7 标称 256k 上下文。保留官方来源，客户端有效窗口及输出预算分别建模，禁止相加，实际接入后验证客户端容量。[Step 5](https://platform.stepfun.com/docs/zh/guides/models/step-5-preview)、[Step 3.7](https://github.com/stepfun-ai/Step-3.7-Flash)

### 用量与资源

- 当前 Step Plan 为 Credit 月池，并有各自到期的加油包；旧 Coding Plan 用户可能继续旧权益。Credits 不是 Token 数，套餐资源不能套 Codex 5h／周窗口，也不能硬编码新套餐描述覆盖旧账号。[国内概述](https://platform.stepfun.com/docs/zh/step-plan/overview)、[海外升级公告](https://platform.stepfun.ai/docs/en/step-plan/upgrade-notice)
- 按量余额与 Step Plan 独立，订阅调用不扣按量余额。耗尽、认证失败或网络失败均不允许自动更换域名／转到付费通道。[海外概述](https://platform.stepfun.ai/docs/en/step-plan/overview)
- 国内、海外均公开 `GET /v1/accounts`，支持 Bearer Key。字段包括 `type=prepaid/postpaid`、`balance`、`total_cash_balance`、`total_voucher_balance`；后两项文档称总充值／总赠送，不能冒充剩余分项，也不承诺二者之和等于余额。接口未给币种及稳定账户 ID，实际含义和币种需区域实测确认。[国内账户 API](https://platform.stepfun.com/docs/zh/api-reference/accounts/get)、[海外账户 API](https://platform.stepfun.ai/docs/en/api-reference/accounts/get)
- 已检查双方公开文档索引，未找到可确认的 Step Plan 剩余 Credit／到期／加油包查询 API；标记待确认。`/v1/accounts` 不能填充订阅剩余。首版展示本机 Token 用量、未知的官方套餐资源和对应官方入口。
- 国内另有组织项目 Key、组织共享 Credit／项目及成员月上限；这与个人 Step Plan 不同。返回 429 时需分速率限制与 `project_credit_limit_exceeded`／`member_project_credit_limit_exceeded`，402 也不能一律解释为个人余额零。[组织额度](https://platform.stepfun.com/docs/zh/guides/organization/credits)、[错误码](https://platform.stepfun.com/docs/zh/api-reference/error-codes)

## 现有实现与影响范围

| 模块 | 当前事实与建议扩展 |
| --- | --- |
| `application/provider/catalog.go`、迁移 002、品牌图标 | 已有 `builtin_stepfun`／`kind=stepfun`，默认国内按量地址；卡片模型为展示名称，不是可请求 ID。保留提供商 ID 和图标。 |
| `application/account`、账号新增／编辑／批量界面 | 账号只有名称、Key 及通用认证信息，没有账号级区域／计费方式。复用 DPAPI 和 `auth_method=api_key`，新增 Step 专用公开选项。 |
| `application/gateway/{service,agent,routing}.go` | 实际支持 DeepSeek、订阅 OpenAI、Atria、Google Chat、MiMo；目标直接取提供商 Base URL，无法同时承载同一品牌的四种入口。 |
| `provider/mimo`、`provider/atria` | 现有标准库原生 HTTP／SSE 是参考；MiMo Messages 路径为 `/anthropic/v1/messages`，不能直接复制给 Step。新增 `provider/stepfun`。 |
| `application/modelcatalog`、`application/routing` | 现有目录版本为 `credential_ref:auth_account_id`，不包含未来账号选项；需使区域／服务变更失效，路由候选先按通道过滤。 |
| `core/proxy`、`application/activity` | 可复用三协议 usage 观察与请求累计；Chat 流可能每块携带累计 usage，必须确认差值算法不重复累加。 |
| `application/balance`、`frontend/features/quota` | 余额 DTO 绑定 DeepSeek 分项语义；配额类型包含 subscription／balance／google／unsupported，需新增 Step 资源展示而非强套现有 DTO。 |
| Wails 绑定、`frontend/services`、接入检查与客户端目录 | 输入和公开元数据须同步更新并重新生成绑定；新增能力判断，原生协议通过验证后才开放相应客户端模型。 |

## 方案比较与推荐

1. 只修改提供商 Base URL：改动小，但整个品牌只能选一个平台／服务，多组 Key 混用会认证错误或误计费。
2. 增加四张独立内置卡片：可复用提供商路由隔离，但会重复品牌、影响当前十家预设和按 kind 展示逻辑，后续维护分散。
3. **推荐一个品牌入口＋账号级平台／服务选项**：保留 `builtin_stepfun`；新账号选“国内／海外”和“按量 API／Step Plan”，四种组合可同时保存。后端统一解析固定地址，目录、转发和资源查询复用同一解析结果。批量添加整批共用这两项，编辑明确展示当前选择。

混合账号必须增加路由隔离；仅加两个下拉框不构成完整接入。首版用现有固定账号规则为四类服务建立独立别名，如 `stepfun-cn-plan/step-5-preview`，改写时仅把真实模型 ID 发给上游。裸模型名在多平台／服务并存时要求明确默认账号，否则报告配置歧义。后续自动选账号只能在明确指定的同区域、同计费池中进行，不能按账号创建顺序隐式改变付费方式。

### 建议数据模型与兼容性

- 新增 Step 专用账号选项表：`account_id` 外键、`region=cn/global`、`billing_mode=payg/step_plan`、单调 `config_version`；两个枚举由 Go 验证，仅 Step API Key 账号允许写入。无新增 SDK，无通用插件配置系统。
- 创建／批量写入选项与账号元数据共用事务，失败清理新凭证；修改选项使用并发版本校验，不要求重新输入未更换的 Key，也不回传密钥。配置改变不影响正在完成的请求快照，但新请求须使用新版本。
- 老账号在空 Base URL 或可识别官方 `/v1` 地址时保持原地区／按量语义；未知自定义地址保留数据并标“需确认配置”，不擅自改成国内或订阅。升级不重置启停、名字或 Key 引用。
- 目录和资源快照绑定凭证版本＋配置版本；相关查询、保存前复查和读取过滤统一更新，防止同步中途切地区后写入旧结果。名字修改不改变配置身份。
- 路由范围用结构化选项或现有固定账号表达，公开别名只负责展示，不通过解析用户字符串猜服务。请求历史保存当次地区／计费方式，旧历史保持未知；同名模型的统计应能区分服务。
- 最新迁移为 020，实施时重新核对并追加下一个编号；不要改迁移 002 或补空缺 018。若未来跨账号重试或资源池身份需要新架构决策，单独 ADR，本轮不扩大范围。

## 风险与待验证项

公开兼容协议不保证每款模型、每个地区的参数完全一致；原生协议优先，未支持字段明确报错，不能静默丢思考／工具历史。四组合的真实 Key、套餐权限、Responses 模型范围、余额币种／分项口径、同账户多 Key 的共享资源身份、旧套餐和 Step Plan 查询接口均需实施期验收。没有稳定主体 ID 时不汇总多个 Key 的余额／Credits，避免重复相加；不根据相同余额推断同一账户。

# Implementation Plan

1. 核实官方四类端点、认证、模型发现、协议和计费能力。
2. 检查现有账号、网关、模型目录、协议桥接和配额扩展点。
3. 比较账号与提供商层配置方案，记录推荐数据模型、实施阶段和验收标准。
4. 人工核对文档、检查差异，提交并归档规划结果。

## 后续实施阶段（本轮不执行）

### P1：四组合账号配置

落地专用选项表、单个／批量／编辑选项、区域与计费标识、兼容升级及 DTO／Wails 绑定。认证沿用 API Key。固定端点解析集中在后端；普通推理、订阅推理和普通管理查询分别选明确路径，避免重复 `/v1` 或丢 `/step_plan`。先覆盖迁移、非法枚举、批量回滚、并发更新和四种地址。

### P2：模型、路由和原生调用闭环

- `provider/stepfun` 实现 Chat／Messages／Responses HTTP 与真正增量 SSE，官方 HTTPS、取消／超时、响应及 JSON 大小边界、禁止重定向和客户端认证头透传；真实 Key 仅在 `UseAPIKey` 回调内使用。自定义代理按现有自定义提供商授权处理，不接收正文指定 URL。
- 普通通道发现 `/v1/models`；Step Plan 先用明确标记的官方订阅预设，不伪造已发现／已验证时间。未来探测订阅模型 API 时，失败不能回落普通目录制造权限。限制目录数量／ID／元数据，非文本接口模型不自动成为 Agent 目标。
- 完成四池路由隔离、默认账号／独立别名、客户端目录和接入检查，保留三协议原始思考与工具字段。Responses 首批以 Step 5／Step 3.7 为已确认候选，其余按协议验证；Router 先明确 Chat／Messages 限制，不能假设支持 Responses。
- 复用上游 usage 观察：区分累计与增量、缓存及推理 Token、截断流与无 usage；不把思考 Token 再加进总数。记录本机用量来源及当次地区／服务，无正文和凭证日志。

### P3：余额与订阅资源展示

- 调用所选区域普通管理路径 `/v1/accounts`，严格解析数值／类型／长度，金额使用十进制值；校验币种后展示可用余额，保留 prepaid／postpaid，负余额不能直接套预付耗尽判断。
- 不把总充值／总赠送写进 DeepSeek 的剩余余额分项。按量资源、Step Plan Credits、组织 Credit 独立表示；订阅账号查询到普通余额也不能将其标为套餐剩余。
- Step Plan 官方剩余、周期和加油包 API 继续待确认；首版可显示未知及区域官方订阅入口。获得公开受支持接口并真实验证后再接入，不抓登录 Cookie 或未公开控制台 RPC，不用本机 Tokens 反推官方剩余。
- 账号与配额总览共享缓存，刷新有界且版本失效；失败保留旧值并标过期。错误按已确认代码区分认证、余额／Credit、项目上限、短期限流和服务器故障；不自动购套餐、充值或启用按量回落。

### P4：真实客户端与容量验收

在用户通过应用保存相应 Key 后，以短提示和受限输出完成四组合真实调用、SSE、完整两轮工具调用、思考和用量对照。Codex 使用已确认 Responses，Claude 使用原生 Messages；不改现有客户端认证身份。长窗口单独做容量验证，未实测前区分官方标称与验证状态。图像／视频、语音、文件／知识库、StepSearch MCP、购买和企业组织管理另行安排。

## 最小充分验证

1. 升级保留内置 ID、旧账号及未知地址；两类认证 UI 不混淆。四组合同时存在，单个／批量／编辑可正确保存，非法输入及失败不留新凭证。
2. 请求命中正确主机／路径；套餐耗尽、401、429、网络失败均不跨平台或跨计费池；模型裸名歧义明确显示，别名不会发给上游。
3. 模型／资源同步期间切地区、计费方式或 Key，旧结果不能写回或成为路由候选；启停、固定账号和客户端目录一致。
4. 三协议 HTTP／SSE、两轮工具调用、思考字段、累计 usage、无 usage、截断／取消、413、401／403／402／429／5xx 和重定向有契约测试；按模型能力分别验收。
5. 余额币种、prepaid／postpaid、未知／负数／超界、分项语义、同主体多 Key 及失效快照覆盖；Credits 与 Token、按量余额、组织额度不会互相代填。
6. 原生深浅主题验证选项、区域／服务标识、混合账号及未知配额提示；回归已有提供商，实际开发实施后运行 `task verify` 并记录结果。

# Progress

- 已核对官方四类地址、Step Plan 原生 Responses、普通余额接口及 Credit 边界。
- 网页检索读取国内文档失败后，直接读取官方公开 Markdown，成功确认国内端点及 Router 限制；没有登录、读取 Key 或发送推理。
- 形成一个品牌入口、账号级选项、计费池隔离的推荐方案与四阶段验收安排。

# Implementation Result

已完成官方资料和代码核对、方案比较、账号／路由／版本模型建议、实施阶段及验收清单。仅修改本任务文档与当前任务看板；代码、数据库、账号、客户端及开发进程未改变。

# Verification

- 官方链接已打开或读取其同路径公开 Markdown，人工核对四类地址、Messages 根路径差异、Step Plan Responses、模型发现和账户字段；未发送真实推理。
- 任务文档 163 行，UTF-8 无 BOM；人工复查方案、已知／未知边界和实施条件。
- `git diff --check` PASS；提交前再次检查暂存区，仅包含本轮两份文档。
- 本轮仅文档改动，无可执行代码／配置示例，不运行 `task verify`；实际接入后按上方验证清单运行质量门。

# Documentation

规划只同步本任务记录和 `docs/plans/CURRENT.md`，PRD／架构继续描述现有实现，不将未实现能力写成已支持。

# Deviations

用户补充普通 API 和 Step Plan 都需要，因此规划扩展为四组合；实施范围仍为规划。国内公开网页抓取失败，改读官方提供的公开 Markdown，未使用第三方文档替代官方结论。

# Git

待提交。

# Result

规划完成，推荐一个阶跃星辰入口、四类账号独立配置，按 P1–P4 完成接入与验收。接入代码尚未实施；Step Plan 官方剩余查询保持待确认，不阻碍调用能力的后续实现。
