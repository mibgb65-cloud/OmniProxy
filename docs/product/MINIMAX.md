# MiniMax 接入

沿用 `builtin_minimax` 和现有图标，账号选择国内／海外、普通按量 API／订阅 Key。单个、批量和编辑均支持；新账号默认国内按量，一批账号共用选择。订阅仍为 `auth_method=api_key`，不新增网页登录授权。

| 平台 | OpenAI 基础地址 | Messages 完整路径 | 订阅资源查询 |
| --- | --- | --- | --- |
| 国内 | `https://api.minimax.cn/v1` | `https://api.minimax.cn/anthropic/v1/messages` | `https://www.minimax.cn/v1/token_plan/remains` |
| 海外 | `https://api.minimax.io/v1` | `https://api.minimax.io/anthropic/v1/messages` | `https://www.minimax.io/v1/token_plan/remains` |

Key 须与来源平台和计费类型一致；已知 `sk-api-`／`sk-cp-` 前缀误配会拒绝，不根据不透明旧 Key 猜平台。按量与订阅共用 API 地址，真实资源权限仍由 MiniMax 确认。不会因为认证失败、限流、额度不足或网络故障换平台／换按量 Key。订阅额度之外，上游可能继续使用用户已购积分包，网关不会改变该规则。

## 升级兼容与账号身份

迁移 024 保存账号级选项和版本。旧空地址按原 `.io` 默认保留海外地区，识别旧 `api.minimaxi.com` 和新 `.cn` 为国内；未知自定义地址不猜地区。旧账号的计费类型无法由 URL 判断，统一提示编辑确认后才能转发，凭据引用、名字和启停保持。

旧国内文档本次已跳转 `.cn`，但不能据此断言旧 API 已失效。实际转发固定使用现行官方域名，禁止携 Key 跟随 HTTP 重定向，不做新旧域名失败回退。

修改地区／Key 类型递增配置版本并失效模型目录；凭据轮换沿用已有凭据版本。目录同步、网关解析和读取 Key 均校验账号快照，避免请求解析后被切到另一通道。请求历史保存当次地区／计费类型，账号编辑不会改写历史。

## 模型、目录与路由

| 模型 | 官方标称上下文 | 原生协议 |
| --- | --- | --- |
| `MiniMax-M3` | 1,000,000 | Chat／Messages／Responses |
| `MiniMax-M2.7`、`MiniMax-M2.7-highspeed` | 204,800 | Chat／Messages／Responses |
| `MiniMax-M2.5`、`MiniMax-M2.5-highspeed` | 204,800 | Chat／Messages／Responses |
| `MiniMax-M3.1-Flash-Preview` | 1,000,000 | 订阅账号同步发现后可用，三协议 |

首版只公布文本、思考和函数工具能力；语音、图片／视频输入、生成、文件、MCP 不在本版。模型卡片是官方候选，不证明任何 Key 的权限。M3.1 当前官方仅经 M Plan／MiniMax Code 提供，普通按量目录排除它，旧订阅预设也不直接加入。

账号可同步所在地区 `/v1/models`，只收已支持的对话模型并去重、有界校验。成功快照优先；首次同步前提供五款普通模型的 `preset` 候选，无伪造查询时间。模型列表的定义是平台模型，成功返回仍不保证当前套餐授权或真实长窗口能力。

四类通道提供独立别名，例如：

- `minimax-cn-api/MiniMax-M3`
- `minimax-global-api/MiniMax-M3`
- `minimax-cn-plan/MiniMax-M3`
- `minimax-global-plan/MiniMax-M3`

同一通道内复用已有账号选择；模型名发给上游前恢复真实 ID。用户精确规则优先，裸模型名遇到多个地区／计费通道返回 `ambiguous_minimax_pool`，使用通道名或固定账号规则消除歧义。客户端目录与路由预览沿用真实决策。

## 原生协议与能力边界

三协议保持原始 JSON／增量 SSE，不将完整结果收集后伪造流。保留思考字段、`<think>`、thinking 签名和工具 ID；Chat 流开启最终 usage，统计沿用现有观察器，缓存和推理 Token 不额外重复累加。

M3 支持思考开关；M3.1 始终思考，公布 low／medium／high／xhigh／max，默认 max。M2.x 不允许网关将其标成关闭思考。协议原生参数仍按各自格式处理，不能把 M3 的 Responses 默认套到 Chat 上。

Responses 使用完整历史，只支持已确认的函数工具；拒绝依赖服务端状态的 `store=true`、后台任务、历史 ID 续接及未支持的上下文管理。namespace、custom／apply_patch、服务端内置工具和 WebSocket 尚未开放；实际 Codex 工具任务需要单独验证，不能仅凭 Responses 端点存在宣称全部 Agent 功能可用。Claude 使用原生 Messages。

## 用量与订阅资源

账号详情、列表与配额总览使用同一有界查询队列和 Query 缓存；缓存身份包含当前账号更新时间、地区、类型和版本。查询后再次校验账号，编辑期间的旧结果不作为新资源发布；查询失败保留旧值并显示错误和查询时间。

订阅通过对应地区的固定 `www` 主机查询，显示官方资源分组、当前／周窗口、剩余百分比、重置时间和周额度倍率。优先使用明确的 `remaining_percent`；旧次数型数据按照官方 CLI 口径将 `usage_count` 解释为剩余次数，不能再次相减。总数与次数均零且无百分比时为未知，不显示满额；无限额是独立状态。重置字段按官方 Unix 毫秒解析，不把 `remains_time` 解释成剩余 Token。

窗口倍率单独显示，不将基础百分比、多个模型分组或多把 Key 相加。积分包详细余额／到期和稳定资源池身份未确认，留官方入口查看。本机 Token 用量与官方套餐剩余分开显示，无法推算平台其他工具调用的消费。

普通按量余额本版未接入，不调用未经确认的控制台 RPC，也不以订阅剩余替代普通余额。配额页只对明确的 MiniMax 订阅账号查询；普通账号显示未接入及官方平台入口。

## 验证边界

契约测试使用临时数据库与伪凭证，覆盖四通道三协议、路由／用量身份、思考／工具历史、增量 SSE、取消、无跨通道回退、目录失效、旧账号迁移和订阅资源解析。前端测试覆盖新增／批量／编辑、未知配置、官方百分比和按量不查询订阅。

真实 Key、M Plan／旧 Token Plan 实际权限、官方资源生产响应、真实长窗口和客户端工具任务尚未验收。字段依据当前官方 CLI，不把契约测试当作真实授权证明。质量门与原生窗口检查见实施任务记录。

## 官方依据

- [国内 OpenAI SDK](https://platform.minimax.cn/docs/api-reference/text-openai-api)、[海外 OpenAI SDK](https://platform.minimax.io/docs/api-reference/text-openai-api)
- [国内 Messages](https://platform.minimax.cn/docs/api-reference/text-chat-anthropic)、[海外 Anthropic SDK](https://platform.minimax.io/docs/api-reference/text-anthropic-api)
- [国内 Responses](https://platform.minimax.cn/docs/api-reference/responses-create)、[海外 Responses](https://platform.minimax.io/docs/api-reference/responses-create)
- [国内模型目录](https://platform.minimax.cn/docs/api-reference/models/openai/list-models)、[海外模型目录](https://platform.minimax.io/docs/api-reference/models/openai/list-models)
- [国内 M Plan FAQ](https://platform.minimax.cn/docs/m-plan/faq)、[海外 M Plan FAQ](https://platform.minimax.io/docs/m-plan/faq)
- [官方 CLI 资源类型](https://github.com/MiniMax-AI/cli/blob/main/src/types/api.ts)、[旧版计数口径](https://github.com/MiniMax-AI/cli/blob/main/src/utils/quota.ts)、[官方资源测试样本](https://github.com/MiniMax-AI/cli/blob/main/test/fixtures/quota-response.json)

核对日期：2026-10-15。规划 TASK-20261015-003，实施 TASK-20261015-004。
