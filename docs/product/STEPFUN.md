# 阶跃星辰接入

阶跃星辰沿用一个内置提供商 `builtin_stepfun`，每个 API Key 账号选择平台和计费方式。单个添加、批量添加和编辑均支持这两项；一批账号共用选择。Key 保存于受保护凭证存储，公开接口只返回选项与版本。

账号详情页提供与 MiMo 同款的设置入口。「默认使用顺序」可选择 `Step Plan → API Key（按量）` 或 `API Key（按量） → Step Plan`，初始 Step Plan 优先；选择后自动保存，对下一次路由选择生效。读取失败可重试，保存失败保留原值；与 MiMo 的偏好独立。

| 平台 | 普通按量 API | Step Plan |
| --- | --- | --- |
| 国内 | `https://api.stepfun.com/v1` | `https://api.stepfun.com/step_plan/v1` |
| 海外 | `https://api.stepfun.ai/v1` | `https://api.stepfun.ai/step_plan/v1` |

Key 必须与签发平台匹配。Step Plan 账号仍是 `auth_method=api_key`，需在对应平台有有效订阅。账号地区和计费选择决定实际端点；提供商卡片上的默认国内地址及无凭证探测不能证明账号认证、模型权限或订阅额度。

## 模型和协议

首版支持文本、思考历史和函数工具调用的原生 HTTP／增量 SSE，保留三种协议的上游响应及事件。

| 模型 | Chat Completions | Messages | Responses |
| --- | --- | --- | --- |
| `step-5-preview` | 支持 | 支持 | 支持 |
| `step-3.7-flash` | 支持 | 支持 | 支持 |
| `step-3.5-flash` | 支持 | 支持 | 待确认 |
| `step-3.5-flash-2603` | 支持 | 支持 | 待确认 |
| `step-router-v1` | 国内 Step Plan | 国内 Step Plan | 待确认 |

Step 5 保存官方 1M 标称上下文和 64k 输出上限，Step 3.7 保存 256k 标称上下文；没有真实长输入验证时，不标为已验证。Router 是上游模型，不能从其底层引擎最大窗口推断每次请求的能力；首版不声明其上下文上限或推理档位。

图片、视频、语音、文件／知识库、内置搜索工具和 StepSearch MCP 尚未接入。Responses 必须携带完整历史，拒绝 `store=true`、后台任务、服务端响应 ID 续接及未支持的上下文管理；`store=false`／`background=false` 在验证后移除。不能用 Chat 转换来宣称未知 Responses 能力。

Messages 完整路径是 `/v1/messages` 或 `/step_plan/v1/messages`，与 MiMo 的 `/anthropic/v1/messages` 不同。上游统一使用 Bearer Key；客户端认证头不透传，HTTP 重定向被禁止。

## 模型目录与路由

普通按量账号可手动同步 `/v1/models`，只把已支持文本模型纳入目录；失败保留旧目录。同一账号第一次同步前使用明确标记的官方预设，不伪造查询时间。Step Plan 使用订阅预设，界面不提供普通目录同步，不向未确认的 `/step_plan/v1/models` 发请求。

公共模型列表、网关模型发现及受管客户端目录只发布原模型及用户显式别名，不再自动列出四类通道副本。旧通道模型名仍兼容解析，例如：

- `stepfun-cn-api/step-5-preview`
- `stepfun-global-api/step-5-preview`
- `stepfun-cn-plan/step-5-preview`
- `stepfun-global-plan/step-5-preview`

这些名称由结构化账号配置生成，匹配后仅把真实模型 ID 发往上游。旧 Claude `[1m]`／桌面编码 ID 也保留能力校验后的兼容解析。同一通道可有多个账号；选择可用账号后，整个请求固定该账号，不做请求失败重试。用户保存的精确规则优先于生成的通道别名。

裸模型名先遵循服务商优先级，每个服务商内按设置选有已启用且支持模型账号的首选计费方式，无合格首选账号时使用另一方式；国内／海外及同通道账号继续按创建时间、ID 稳定选择。`step-router-v1` 仍仅国内 Step Plan 可用，不因偏好或去重扩展协议、地区权限。固定账号与显式通道优先于默认顺序。套餐耗尽、认证失败、限流或网络失败均不会触发跨通道重试。禁用手工规则仍沿用已有匹配语义；不代表禁用某个通道。

迁移 027 保存 StepFun 默认顺序，缺失或损坏时报错，避免静默改变计费。保存时刷新既有受管客户端目录；外部客户端可能需要重新加载列表。账号选项、凭据和已选定目标的在途请求保持原有身份。

迁移 021 新增账号选项与目录版本视图；旧可识别官方地址保持原地区及服务语义，未知自定义地址保留数据并要求明确配置。修改地区／服务使配置版本递增，并失效旧模型目录；同步完成前再次校验版本，旧请求解析结果不能跟随到新计费池。重命名保持配置身份，换 Key 仍使用既有凭据版本失效。

## 余额、额度和用量

普通账号使用所在区域的 `/v1/accounts` 查询余额，详情与配额页共享有界查询队列和缓存。按接口定义分别展示账户类型、可用余额、总充值与总赠送；后两项不是剩余分项。金额保留十进制文本，不用浮点重新计算。国内金额已由用户通过控制台截图确认为人民币，显示 CNY／¥；海外币种未确认，继续显示“币种待确认”；负后付费余额不自动判定资源耗尽。没有稳定账户主体 ID，不汇总多个 Key 的余额。

Step Plan 仅显示剩余 Credits 未知与对应官方平台入口，不查询普通余额冒充订阅剩余、不由本机 Token 推算官方 Credit 池或周期。套餐及加油包查询 API、旧套餐真实权益仍待确认。

请求历史保存当次地区／计费方式；账号之后变更不改写历史。账号用量身份包含该通道，以免更换计费方式后合并不同资源池。本机用量仍依据上游三协议 usage，累计流式快照只记差值；不代表平台完整消费或外部调用用量。

错误标识仅接受已公开的 `insufficient_credit`、`project_credit_limit_exceeded`、`member_project_credit_limit_exceeded` 等分类，原始错误消息不进入日志。组织额度错误与个人 Step Plan 资源保持区别；本版不实现组织管理。

## 验证边界

测试使用独立数据库和伪凭证，覆盖四组合原生调用、工具历史、SSE、计费隔离、目录版本、迁移、余额及前端选择。真实阶跃 Key、套餐权限、海外金额币种／分项语义、长上下文和实际 Codex／Claude 工具任务尚未验收，不能把预设和契约测试当作真实平台授权证明。

## 官方依据

- [国内接入](https://platform.stepfun.com/docs/zh/step-plan/integrations/reasoning-api)、[海外接入](https://platform.stepfun.ai/docs/en/step-plan/integrations/reasoning-api)
- [原生 Messages](https://platform.stepfun.com/docs/zh/api-reference/chat/messages-create)、[Responses](https://platform.stepfun.ai/docs/en/api-reference/responses/responses-create)
- [国内 Codex／Step Plan Responses](https://platform.stepfun.com/docs/zh/guides/developer/integrations/codex)
- [模型列表](https://platform.stepfun.com/docs/zh/api-reference/models/list)、[余额接口](https://platform.stepfun.com/docs/zh/api-reference/accounts/get)
- [Step Plan 资源](https://platform.stepfun.com/docs/zh/step-plan/overview)、[组织额度](https://platform.stepfun.com/docs/zh/guides/organization/credits)

核对日期：2026-10-14。实施规划见 TASK-20261014-015，实施与验收见 TASK-20261014-016；默认顺序及目录去重见 TASK-20261016-004、ADR-011。
